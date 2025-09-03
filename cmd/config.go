package main

import "os"

type Config struct {
	// App
	Host string
	Port string
	// Orderspace Client
	OrderspaceBaseURL      string
	OrderspaceClientID     string
	OrderspaceClientSecret string
	// Woocommerce Client
	WooBaseURL        string
	WooConsumerKey    string
	WooConsumerSecret string
	// Database
	ConnectionString string
}

func getEnv() Config {
	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	orderspaceBaseURL := os.Getenv("ORDERSPACE_BASE_URL")
	orderspaceClientID := os.Getenv("ORDERSPACE_CLIENT_ID")
	orderspaceClientSecret := os.Getenv("ORDERSPACE_CLIENT_SECRET")

	wooBaseURL := os.Getenv("WOO_BASE_URL")
	wooConsumerKey := os.Getenv("WOO_CONSUMER_KEY")
	wooConsumerSecret := os.Getenv("WOO_CONSUMER_SECRET")

	dbConnectionString := os.Getenv("DB_CONNECTION_STRING")

	return Config{
		Host:					host,
		Port:                   port,
		OrderspaceBaseURL:      orderspaceBaseURL,
		OrderspaceClientID:     orderspaceClientID,
		OrderspaceClientSecret: orderspaceClientSecret,
		WooBaseURL:             wooBaseURL,
		WooConsumerKey:         wooConsumerKey,
		WooConsumerSecret:      wooConsumerSecret,
		ConnectionString:       dbConnectionString,
	}
}