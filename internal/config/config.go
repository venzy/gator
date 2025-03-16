package config

import (
	"encoding/json"
	"os"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DbUrl string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

var badConfig Config = Config{"",""}

func Read() (Config, error) {
	// Open default config file for read
	configPath, err := getConfigFilePath()
	if err != nil {
		return badConfig, err
	}
	var cfg Config
	file, err := os.Open(configPath)
	if err != nil {
		return badConfig, err
	}
	defer file.Close()

	// Parse config
    decoder := json.NewDecoder(file)
    if err := decoder.Decode(&cfg); err != nil {
        return badConfig, err
    }
    return cfg, nil
}

func SetUser(cfg Config, user string) error {
	cfg.CurrentUserName = user
	return write(cfg)
}

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return homeDir + "/" + configFileName, nil
}

func write(cfg Config) error {
	// Open default config file for write
	configPath, err := getConfigFilePath()
	if err != nil {
		return err
	}
	file, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(cfg)
	return err
}