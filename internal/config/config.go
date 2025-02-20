package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	models "github.com/rachel-lawrie/verus_backend_core/models"
	"github.com/spf13/viper"
)

func LoadConfig(env string) models.Config {
	// Create separate Viper instances for YAML and env
	yamlV := viper.New()
	envV := viper.New()

	// Load YAML config first
	yamlV.SetConfigName(env)
	yamlV.SetConfigType("yaml")
	yamlV.AddConfigPath("config/")
	if err := yamlV.ReadInConfig(); err != nil {
		log.Panicf("Error reading YAML config: %v", err)
	}

	var config models.Config
	if err := yamlV.Unmarshal(&config); err != nil {
		log.Panicf("Unable to decode YAML into struct: %v", err)
	}

	// Load .env separately
	envV.SetConfigName(".env")
	envV.SetConfigType("env")
	envV.AddConfigPath("/app")
	if err := envV.ReadInConfig(); err != nil {
		log.Printf("Error reading .env: %v", err)
	}

	// Now overlay env values in dev environment
	if env == "dev" {
		// Database credentials
		if username := envV.GetString("DB_USERNAME"); username != "" {
			config.Database.User = username
			log.Printf("Set Database.User to: %s", username)
		}
		if password := envV.GetString("DB_PASSWORD"); password != "" {
			config.Database.Password = password
			log.Printf("Set Database.Password (exists: %v)", password != "")
		}

		// AWS credentials
		if accessKey := envV.GetString("AWS_ACCESS_KEY_ID"); accessKey != "" {
			config.AWS.AccessKeyID = accessKey
			log.Printf("Set AWS.AccessKeyID")
		}
		if secretKey := envV.GetString("AWS_SECRET_ACCESS_KEY"); secretKey != "" {
			config.AWS.SecretAccessKey = secretKey
			log.Printf("Set AWS.SecretAccessKey")
		}
		if keyID := envV.GetString("AWS_KEY_ID"); keyID != "" {
			config.AWS.KeyID = keyID
			log.Printf("Set AWS.KeyID")
		}

		// Vendor credentials (from dev.yaml vendors.sumsub.webhookSecretKey)
		if webhookSecret := envV.GetString("WEBHOOK_SECRET_KEY"); webhookSecret != "" {
			if config.Vendors == nil {
				config.Vendors = make(map[string]models.VendorConfig)
			}
			vendor := config.Vendors["sumsub"]
			vendor.WebhookSecretKey = webhookSecret
			config.Vendors["sumsub"] = vendor
			log.Printf("Set Vendors.sumsub.WebhookSecretKey")
		}

		// Log final database config
		log.Printf("Final Database Config - Host: %s, Port: %d, User: %s, Password exists: %v",
			config.Database.Host,
			config.Database.Port,
			config.Database.User,
			config.Database.Password != "")
	}

	return config
}

// LoadSecretsFromAWS loads secrets from AWS Secrets Manager
func LoadSecretsFromAWS(secretName string) (map[string]string, error) {
	// Load AWS SDK configuration
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %v", err)
	}

	// Create Secrets Manager client
	svc := secretsmanager.NewFromConfig(cfg)

	// Get secret value
	result, err := svc.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get secret value: %v", err)
	}

	// Parse JSON secret string into map
	var secretMap map[string]string
	if err := json.Unmarshal([]byte(*result.SecretString), &secretMap); err != nil {
		return nil, fmt.Errorf("unable to parse secret JSON: %v", err)
	}

	return secretMap, nil
}
