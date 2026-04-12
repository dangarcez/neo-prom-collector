package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadFile(path string) (FileConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return FileConfig{}, fmt.Errorf("open config file: %w", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	var cfg FileConfig
	if err := decoder.Decode(&cfg); err != nil {
		return FileConfig{}, fmt.Errorf("decode config file: %w", err)
	}

	cfg.Normalize()

	if err := ValidateFileConfig(cfg); err != nil {
		return FileConfig{}, err
	}

	return cfg, nil
}
