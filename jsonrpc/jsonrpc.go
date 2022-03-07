package jsonrpc

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-hclog"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type serverType int

const (
	serverIPC serverType = iota
	serverHTTP
	serverWS
)

func (s serverType) String() string {
	switch s {
	case serverIPC:
		return "ipc"
	case serverHTTP:
		return "http"
	case serverWS:
		return "ws"
	default:
		panic("BUG: Not expected")
	}
}

// JSONRPC is an API backend
type JSONRPC struct {
	logger     hclog.Logger
	config     *Config
	dispatcher dispatcher
}

type dispatcher interface {
	HandleWs(reqBody []byte, conn wsConn) ([]byte, error)
	Handle(reqBody []byte, txn *newrelic.Transaction) ([]byte, error)
}

// JSONRPCStore defines all the methods required
// by all the JSON RPC endpoints
type JSONRPCStore interface {
	ethStore
	networkStore
	txPoolStore
	filterManagerStore
}

type RPCNrConfig struct {
	RPCNrAppName    string
	RPCNrLicenseKey string
}

type Config struct {
	Store                    JSONRPCStore
	Addr                     *net.TCPAddr
	ChainID                  uint64
	RPCNrConfig              *RPCNrConfig
	AccessControlAllowOrigin []string
}

// NewJSONRPC returns the JSONRPC http server
func NewJSONRPC(logger hclog.Logger, config *Config) (*JSONRPC, error) {
	srv := &JSONRPC{
		logger:     logger.Named("jsonrpc"),
		config:     config,
		dispatcher: newDispatcher(logger, config.Store, config.ChainID),
	}

	// start http server
	if err := srv.setupHTTP(); err != nil {
		return nil, err
	}

	return srv, nil
}

func (j *JSONRPC) setupHTTP() error {
	j.logger.Info("http server started", "addr", j.config.Addr.String())

	lis, err := net.Listen("tcp", j.config.Addr.String())
	if err != nil {
		return err
	}

	mux := http.DefaultServeMux

	if j.config.RPCNrConfig != nil && j.config.RPCNrConfig.RPCNrLicenseKey != "" {
		newRelicApp, err := newrelic.NewApplication(
			newrelic.ConfigAppName(j.config.RPCNrConfig.RPCNrAppName),
			newrelic.ConfigLicense(j.config.RPCNrConfig.RPCNrLicenseKey),
			func(cfg *newrelic.Config) {
				cfg.ErrorCollector.RecordPanics = true
			},
		)

		if err != nil {
			return err
		}

		jsonRPCHandler := http.HandlerFunc(j.handle)
		wrapCorsHandler := middlewareFactory(j.config)(jsonRPCHandler)
		mux.Handle(newrelic.WrapHandle(newRelicApp, "/", wrapCorsHandler))
		mux.HandleFunc(newrelic.WrapHandleFunc(newRelicApp, "/ws", j.handleWs))
		mux.HandleFunc(newrelic.WrapHandleFunc(newRelicApp, "/health", j.handleHealth))
	} else {
		// The middleware factory returns a handler, so we need to wrap the handler function properly.
		jsonRPCHandler := http.HandlerFunc(j.handle)
		mux.Handle("/", middlewareFactory(j.config)(jsonRPCHandler))
		mux.HandleFunc("/ws", j.handleWs)
		mux.HandleFunc("/health", j.handleHealth)
	}

	srv := http.Server{
		Handler: mux,
	}

	go func() {
		if err := srv.Serve(lis); err != nil {
			j.logger.Error("closed http connection", "err", err)
		}
	}()

	return nil
}

// The middlewareFactory builds a middleware which enables CORS using the provided config.
func middlewareFactory(config *Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			for _, allowedOrigin := range config.AccessControlAllowOrigin {
				if allowedOrigin == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// wsUpgrader defines upgrade parameters for the WS connection
var wsUpgrader = websocket.Upgrader{
	// Uses the default HTTP buffer sizes for Read / Write buffers.
	// Documentation specifies that they are 4096B in size.
	// There is no need to have them be 4x in size when requests / responses
	// shouldn't exceed 1024B
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// wsWrapper is a wrapping object for the web socket connection and logger
type wsWrapper struct {
	ws        *websocket.Conn // the actual WS connection
	logger    hclog.Logger    // module logger
	writeLock sync.Mutex      // writer lock
}

// WriteMessage writes out the message to the WS peer
func (w *wsWrapper) WriteMessage(messageType int, data []byte) error {
	w.writeLock.Lock()
	defer w.writeLock.Unlock()
	writeErr := w.ws.WriteMessage(messageType, data)

	if writeErr != nil {
		w.logger.Error(
			fmt.Sprintf("Unable to write WS message, %s", writeErr.Error()),
		)
	}

	return writeErr
}

// isSupportedWSType returns a status indicating if the message type is supported
func isSupportedWSType(messageType int) bool {
	return messageType == websocket.TextMessage ||
		messageType == websocket.BinaryMessage
}

func (j *JSONRPC) handleWs(w http.ResponseWriter, req *http.Request) {
	// CORS rule - Allow requests from anywhere
	wsUpgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// Upgrade the connection to a WS one
	ws, err := wsUpgrader.Upgrade(w, req, nil)
	if err != nil {
		j.logger.Error(fmt.Sprintf("Unable to upgrade to a WS connection, %s", err.Error()))

		return
	}

	// Defer WS closure
	defer func(ws *websocket.Conn) {
		err = ws.Close()
		if err != nil {
			j.logger.Error(
				fmt.Sprintf("Unable to gracefully close WS connection, %s", err.Error()),
			)
		}
	}(ws)

	wrapConn := &wsWrapper{ws: ws, logger: j.logger}

	j.logger.Info("Websocket connection established")
	// Run the listen loop
	for {
		// Read the incoming message
		msgType, message, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure,
				websocket.CloseAbnormalClosure,
			) {
				// Accepted close codes
				j.logger.Info("Closing WS connection gracefully")
			} else {
				j.logger.Error(fmt.Sprintf("Unable to read WS message, %s", err.Error()))
				j.logger.Info("Closing WS connection with error")
			}

			break
		}

		if isSupportedWSType(msgType) {
			go func() {
				resp, handleErr := j.dispatcher.HandleWs(message, wrapConn)
				if handleErr != nil {
					j.logger.Error(fmt.Sprintf("Unable to handle WS request, %s", handleErr.Error()))

					_ = wrapConn.WriteMessage(
						msgType,
						[]byte(fmt.Sprintf("WS Handle error: %s", handleErr.Error())),
					)
				} else {
					_ = wrapConn.WriteMessage(msgType, resp)
				}
			}()
		}
	}
}

func (j *JSONRPC) handleHealth(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers",
		"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	if (*req).Method == "OPTIONS" {
		return
	}

	if req.Method != "GET" {
		w.WriteHeader(http.StatusBadRequest)
		//nolint
		w.Write([]byte("method " + req.Method + " not allowed"))

		return
	}

	if j.config.Store.IsIbftStateStale() {
		w.WriteHeader(http.StatusTooEarly)
		//nolint
		w.Write([]byte("IBFT node in stale state, try again shortly.."))

		return
	}

	w.WriteHeader(http.StatusOK)
	//nolint
	w.Write(nil)
}

func (j *JSONRPC) handle(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set(
		"Access-Control-Allow-Headers",
		"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization",
	)

	txn := newrelic.FromContext(req.Context())
	txn.SetName("JSON-RPC Method")

	if (*req).Method == "OPTIONS" {
		return
	}

	if req.Method == "GET" {
		//nolint
		w.Write([]byte("Polygon Edge JSON-RPC"))

		return
	}

	if req.Method != "POST" {
		//nolint
		w.Write([]byte("method " + req.Method + " not allowed"))

		return
	}

	data, err := ioutil.ReadAll(req.Body)

	if err != nil {
		txn.NoticeError(err)
		//nolint
		w.Write([]byte(err.Error()))

		return
	}

	// log request
	j.logger.Debug("handle", "request", string(data))

	resp, err := j.dispatcher.Handle(data, txn)

	if err != nil {
		txn.NoticeError(err)
		//nolint
		w.Write([]byte(err.Error()))
	} else {
		//nolint
		w.Write(resp)
	}

	j.logger.Debug("handle", "response", string(resp))
}
