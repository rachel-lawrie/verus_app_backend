package config

import (
	"bytes"
	"log"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Set up a temporary config file for testing
	configContent := `
server:
  port: "8080"
database:
  host: "localhost"
  port: 27017
  user: "testuser"
  password: "testpassword"
  name: "testdb"
aws:
  accessKeyID: "testAccessKeyID"
  secretAccessKey: "testSecretAccessKey"
  region: "us-west-2"
  bucketName: "testBucket"
`
	configDir := "config"
	configFile := "test.yaml"
	os.Mkdir(configDir, 0755)
	defer os.RemoveAll(configDir)
	os.WriteFile(configDir+"/"+configFile, []byte(configContent), 0644)

	// Set up viper to read the test config file
	viper.SetConfigName("test")
	viper.AddConfigPath(configDir)

	// Load the config
	config := LoadConfig("test")

	// Assert the config values
	assert.Equal(t, "8080", config.Server.Port)
	assert.Equal(t, "localhost", config.Database.Host)
	assert.Equal(t, 27017, config.Database.Port)
	assert.Equal(t, "testuser", config.Database.User)
	assert.Equal(t, "testpassword", config.Database.Password)
	assert.Equal(t, "testdb", config.Database.Name)
	assert.Equal(t, "testAccessKeyID", config.AWS.AccessKeyID)
	assert.Equal(t, "testSecretAccessKey", config.AWS.SecretAccessKey)
	assert.Equal(t, "us-west-2", config.AWS.Region)
	assert.Equal(t, "testBucket", config.AWS.BucketName)
}

func TestLoadConfigFileNotFound(t *testing.T) {
	// Set up viper to read a non-existent config file
	viper.SetConfigName("nonexistent")
	viper.AddConfigPath("config/")

	// Capture the log output
	var logOutput bytes.Buffer
	log.SetOutput(&logOutput)
	defer log.SetOutput(os.Stderr)

	// Load the config and expect a fatal error
	assert.Panics(t, func() {
		LoadConfig("nonexistent")
	}, "Expected LoadConfig to panic when config file is not found")

	// Check the log output for the expected error message
	assert.Contains(t, logOutput.String(), "Error reading config file: Config File \"nonexistent\" Not Found")
}
