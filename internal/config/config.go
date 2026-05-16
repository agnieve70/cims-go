package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppName       string
	Addr          string
	DatabaseURL   string
	SessionHash   string
	SessionBlock  string
	AdminUsername string
	AdminPassword string
	Location      *time.Location
}

func Load() (Config, error) {
	loadDotEnv()

	loc, err := time.LoadLocation("Asia/Manila")
	if err != nil {
		return Config{}, fmt.Errorf("load timezone: %w", err)
	}

	cfg := Config{
		AppName:       getEnv("APP_NAME", "Customized Information Management System"),
		Addr:          getEnv("ADDR", ":8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://cims:cims@localhost:5432/cims?sslmode=disable"),
		SessionHash:   getEnv("SESSION_HASH_KEY", "dev-session-hash-key-change-me-32-bytes"),
		SessionBlock:  getEnv("SESSION_BLOCK_KEY", "0123456789abcdef"),
		AdminUsername: getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		Location:      loc,
	}

	if _, err := strconv.ParseBool(getEnv("CIMS_ALLOW_DEV_KEYS", "true")); err != nil {
		return Config{}, fmt.Errorf("CIMS_ALLOW_DEV_KEYS: %w", err)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func loadDotEnv() {
	paths := []string{".env"}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), ".env"))
	}
	seen := map[string]struct{}{}
	for _, path := range paths {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		loadEnvFile(path)
	}
}

func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, value)
	}
}
