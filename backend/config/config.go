package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken     string
	AuthBotToken string
	GoogleClientID     string
	GoogleClientSecret string
	TZ           string
	Debug        bool
	DatabaseURL  string
	WebAddr      string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}

	authToken := os.Getenv("AUTH_BOT_TOKEN")
	if authToken == "" {
		return nil, fmt.Errorf("AUTH_BOT_TOKEN is required")
	}

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if googleClientID != "" && googleClientSecret == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_SECRET is required")
	}
	if googleClientSecret != "" && googleClientID == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID is required")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	tz := os.Getenv("TZ")
	if tz == "" {
		tz = "Asia/Yekaterinburg"
	}

	debug := os.Getenv("DEBUG") == "1"

	webAddr := os.Getenv("WEB_ADDR")
	if webAddr == "" {
		webAddr = ":8080"
	}

	return &Config{
		BotToken:     token,
		AuthBotToken: authToken,
		GoogleClientID:     googleClientID,
		GoogleClientSecret: googleClientSecret,
		TZ:           tz,
		Debug:        debug,
		DatabaseURL:  dbURL,
		WebAddr:      webAddr,
	}, nil
}
