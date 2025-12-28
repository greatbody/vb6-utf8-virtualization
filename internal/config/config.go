package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	PhysicalPath      string   `json:"physical_path"`
	MountPoint        string   `json:"mount_point"`
	AllowedProcesses  []string `json:"allowed_processes"`
	AllowedExtensions []string `json:"allowed_extensions"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		PhysicalPath:      "F:\\",
		MountPoint:        "Z:",
		AllowedProcesses:  []string{"a.exe"},
		AllowedExtensions: []string{".txt", ".csv", ".log", ".ini", ".conf", ".properties", ".bas", ".cls", ".frm", ".vbp"},
	}
}
