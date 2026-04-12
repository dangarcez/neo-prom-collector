package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func LoadEnv(path string) (EnvConfig, error) {
	if err := loadDotEnvFile(path); err != nil {
		return EnvConfig{}, err
	}

	timeoutSeconds, err := getEnvInt("NEO4J_TIMEOUT_SECONDS", 10)
	if err != nil {
		return EnvConfig{}, err
	}

	verifyConnectivity, err := getEnvBool("NEO4J_VERIFY_CONNECTIVITY", true)
	if err != nil {
		return EnvConfig{}, err
	}

	maxDatapointWorkers, err := getEnvInt("APP_MAX_DATAPOINT_WORKERS", 4)
	if err != nil {
		return EnvConfig{}, err
	}
	if maxDatapointWorkers <= 0 {
		maxDatapointWorkers = 1
	}

	return EnvConfig{
		ConfigPath:          getEnv("APP_CONFIG_PATH", "configs/config.demo.yaml"),
		Neo4jURI:            getEnv("NEO4J_URI", "bolt://localhost:7687"),
		Neo4jDatabase:       getEnv("NEO4J_DATABASE", "neo4j"),
		Neo4jUsername:       getEnv("NEO4J_USERNAME", "neo4j"),
		Neo4jPassword:       strings.TrimSpace(os.Getenv("NEO4J_PASSWORD")),
		Neo4jTimeout:        time.Duration(timeoutSeconds) * time.Second,
		VerifyConnectivity:  verifyConnectivity,
		MaxDatapointWorkers: maxDatapointWorkers,
		LogLevel:            getEnv("APP_LOG_LEVEL", "info"),
		LogFormat:           getEnv("APP_LOG_FORMAT", "text"),
	}, nil
}

func loadDotEnvFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok, parseErr := parseEnvLine(line)
		if parseErr != nil {
			return fmt.Errorf("parse env file line %d: %w", lineNumber, parseErr)
		}
		if !ok {
			continue
		}

		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("set env %s: %w", key, err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan env file: %w", err)
	}

	return nil
}

func parseEnvLine(line string) (string, string, bool, error) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(line, "export "))
	if trimmed == "" {
		return "", "", false, nil
	}

	parts := strings.SplitN(trimmed, "=", 2)
	if len(parts) != 2 {
		return "", "", false, fmt.Errorf("invalid env assignment %q", line)
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" {
		return "", "", false, fmt.Errorf("empty env key")
	}

	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}

	return key, value, true, nil
}

func getEnv(key, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return defaultValue
}

func getEnvInt(key string, defaultValue int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid integer for %s: %w", key, err)
	}

	return value, nil
}

func getEnvBool(key string, defaultValue bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("invalid boolean for %s: %w", key, err)
	}

	return value, nil
}
