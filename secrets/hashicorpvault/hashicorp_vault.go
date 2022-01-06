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
	err := vaultManager.Setup()
	if err != nil {
		return nil, err
	}

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

	// Set the namespace
	client.SetNamespace(v.namespace)

	v.client = client

	token, err := v.login()
	if err != nil {
		return fmt.Errorf("login authentication error: %v", err)
	}

	// Set the access token
	client.SetToken(token.Auth.ClientToken)

	return nil
}

// AWS ec2 auth type
func (v *VaultSecretsManager) login() (*vault.Secret, error) {

	awsAuth, err := auth.NewAWSAuth(
		auth.WithEC2Auth(),
		auth.WithRole("foundation-node-role"),
		// override dynamic nonce generation
		auth.WithNonce("static-nonce"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize AWS auth method: %w", err)
	}

	authInfo, err := v.client.Auth().Login(context.Background(), awsAuth)
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
