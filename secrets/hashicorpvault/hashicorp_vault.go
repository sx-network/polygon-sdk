package hashicorpvault

import (
	"context"
	"errors"
	"fmt"

	"github.com/0xPolygon/polygon-sdk/secrets"
	"github.com/hashicorp/go-hclog"
	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/aws"
)

// VaultSecretsManager is a SecretsManager that
// stores secrets on a Hashicorp Vault instance
type VaultSecretsManager struct {
	// Logger object
	logger hclog.Logger

	// Token used for Vault instance authentication
	token string

	// The Server URL of the Vault instance
	serverURL string

	// The name of the current node, used for prefixing names of secrets
	name string

	// The base path to store the secrets in the KV-2 Vault storage
	basePath string

	// The HTTP client used for interacting with the Vault server
	client *vault.Client

	// The namespace under which the secrets are stored
	namespace string

	// approle connection parameters
	approleRoleID       string
	approleSecretIDFile string
}

// SecretsManagerFactory implements the factory method
func SecretsManagerFactory(
	config *secrets.SecretsManagerConfig,
	params *secrets.SecretsManagerParams,
) (secrets.SecretsManager, error) {
	// Set up the base object
	vaultManager := &VaultSecretsManager{
		logger: params.Logger.Named(string(secrets.HashicorpVault)),
	}

	// Check if the token is present
	if config.Token == "" {
		return nil, errors.New("no token specified for Vault secrets manager")
	}

	// Grab the token from the config
	vaultManager.token = config.Token

	// Check if the server URL is present
	if config.ServerURL == "" {
		return nil, errors.New("no server URL specified for Vault secrets manager")
	}

	// Grab the server URL from the config
	vaultManager.serverURL = config.ServerURL

	// Check if the node name is present
	if config.Name == "" {
		return nil, errors.New("no node name specified for Vault secrets manager")
	}

	// Grab the node name from the config
	vaultManager.name = config.Name

	// Grab the namespace from the config
	vaultManager.namespace = config.Namespace

	// Set the base path to store the secrets in the KV-2 Vault storage
	vaultManager.basePath = fmt.Sprintf("secret/data/%s", vaultManager.name)

	// Run the initial setup
	_ = vaultManager.Setup()

	return vaultManager, nil
}

// Setup sets up the Hashicorp Vault secrets manager
func (v *VaultSecretsManager) Setup() error {
	config := vault.DefaultConfig()

	// Set the server URL
	config.Address = v.serverURL
	client, err := vault.NewClient(config)
	if err != nil {
		return fmt.Errorf("unable to initialize Vault client: %v", err)
	}

	// Set the access token
	client.SetToken(v.token)

	// Set the namespace
	client.SetNamespace(v.namespace)

	//TODO v.token needs to be vault.Secret
	go v.RenewLoginPeriodically(context.Background()) // keep alive

	v.client = client

	return nil
}

func (v *VaultSecretsManager) RenewLoginPeriodically(ctx context.Context) {
	v.logger.Debug("dgk - RENEWING LOGIN PERIODICALLY")

	var currentAuthToken *vault.Secret = nil
	for {
		if err := v.renewUntilMaxTTL(ctx, currentAuthToken, "auth token"); err != nil {
			// break out when shutdown is requested
			if errors.Is(err, context.Canceled) {
				return
			}

			v.logger.Debug("auth token renew error: %v", err) // simplified error handling
		}

		// the auth token's lease has expired and needs to be renewed
		t, err := v.login(ctx)
		if err != nil {
			v.logger.Error("login authentication error: %v", err) // simplified error handling
		}

		currentAuthToken = t
	}
}

// renewUntilMaxTTL is a blocking helper function that uses LifetimeWatcher to
// periodically renew the given secret or token when it is close to its
// 'token_ttl' lease expiration time until it reaches its 'token_max_ttl' lease
// expiration time.
func (v *VaultSecretsManager) renewUntilMaxTTL(ctx context.Context, secret *vault.Secret, label string) error {
	watcher, err := v.client.NewLifetimeWatcher(&vault.LifetimeWatcherInput{
		Secret: secret,
	})

	if err != nil {
		return fmt.Errorf("unable to initialize %s lifetime watcher: %w", label, err)
	}

	go watcher.Start()
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return context.Canceled

		// DoneCh will return if renewal fails, or if the remaining lease
		// duration is under a built-in threshold and either renewing is not
		// extending it or renewing is disabled.  In both cases, the caller
		// should attempt a re-read of the secret. Clients should check the
		// return value of the channel to see if renewal was successful.
		case err := <-watcher.DoneCh():
			if err != nil {
				return fmt.Errorf("%s renewal failed: %w", label, err)
			}

			return nil

		// RenewCh is a channel that receives a message when a successful
		// renewal takes place and includes metadata about the renewal.
		case info := <-watcher.RenewCh():
			v.logger.Debug("%s: successfully renewed; remaining lease duration: %ds", label, info.Secret.LeaseDuration)
		}
	}
}

// AWS ec2 auth type
func (v *VaultSecretsManager) login(ctx context.Context) (*vault.Secret, error) {

	awsAuth, err := auth.NewAWSAuth(
		auth.WithEC2Auth(),
		auth.WithRole("foundation-node-role"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize AWS auth method: %w", err)
	}

	authInfo, err := v.client.Auth().Login(ctx, awsAuth)
	if err != nil {
		return nil, fmt.Errorf("unable to login to AWS auth method: %w", err)
	}
	if authInfo == nil {
		return nil, fmt.Errorf("no auth info was returned after login")
	}

	v.logger.Debug("logging in to vault with ec2 auth: success!")

	return authInfo, nil
}

// constructSecretPath is a helper method for constructing a path to the secret
func (v *VaultSecretsManager) constructSecretPath(name string) string {
	return fmt.Sprintf("%s/%s", v.basePath, name)
}

// GetSecret fetches a secret from the Hashicorp Vault server
func (v *VaultSecretsManager) GetSecret(name string) ([]byte, error) {
	secret, err := v.client.Logical().Read(v.constructSecretPath(name))
	if err != nil {
		return nil, fmt.Errorf("unable to read secret from Vault, %v", err)
	}

	if secret == nil {
		return nil, secrets.ErrSecretNotFound
	}

	// KV-2 (versioned key-value storage) in Vault stores data in the following format:
	// {
	// "data": {
	// 		key: value
	// 	}
	// }
	data, ok := secret.Data["data"]
	if !ok {
		return nil, fmt.Errorf(
			"unable to assert type for secret from Vault, %T %#v",
			secret.Data["data"],
			secret.Data["data"],
		)
	}

	// Check if the data is empty
	if data == nil {
		return nil, secrets.ErrSecretNotFound
	}

	// Grab the value
	value, ok := data.(map[string]interface{})[name]
	if !ok {
		return nil, secrets.ErrSecretNotFound
	}

	return []byte(value.(string)), nil
}

// SetSecret saves a secret to the Hashicorp Vault server
// Secrets saved in Vault need to have a string value (Base64)
func (v *VaultSecretsManager) SetSecret(name string, value []byte) error {
	// Check if overwrite is possible
	_, err := v.GetSecret(name)
	if err == nil {
		// Secret is present
		v.logger.Warn(fmt.Sprintf("Overwriting secret: %s", name))
	} else if !errors.Is(err, secrets.ErrSecretNotFound) {
		// An unrelated error occurred
		return err
	}

	// Construct the data wrapper
	data := make(map[string]string)
	data[name] = string(value)

	_, err = v.client.Logical().Write(v.constructSecretPath(name), map[string]interface{}{
		"data": data,
	})
	if err != nil {
		return fmt.Errorf("unable to store secret (%s), %v", name, err)
	}

	return nil
}

// HasSecret checks if the secret is present on the Hashicorp Vault server
func (v *VaultSecretsManager) HasSecret(name string) bool {
	_, err := v.GetSecret(name)

	return err == nil
}

// RemoveSecret removes a secret from the Hashicorp Vault server
func (v *VaultSecretsManager) RemoveSecret(name string) error {
	// Check if overwrite is possible
	_, err := v.GetSecret(name)
	if err != nil {
		return err
	}

	// Delete the secret from Vault storage
	_, err = v.client.Logical().Delete(v.constructSecretPath(name))
	if err != nil {
		return fmt.Errorf("unable to delete secret (%s), %v", name, err)
	}

	return nil
}
