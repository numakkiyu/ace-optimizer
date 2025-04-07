package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	GamePath      string `json:"game_path"`
	LaunchTimeout int    `json:"launch_timeout"`
	AutoClose     bool   `json:"auto_close"`
	LastMode      string `json:"last_mode"`
}

func getConfigPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "config.json")
}

func loadConfig() (*Config, error) {
	configPath := getConfigPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{
			LaunchTimeout: 30,
			AutoClose:     false,
			LastMode:      "normal",
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func saveConfig(config *Config) error {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(getConfigPath(), data, 0644)
}
