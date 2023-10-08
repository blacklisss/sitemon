package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/namsral/flag"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Timing struct {
		Delay time.Duration `yaml:"delay" json:"delay"`
	} `yaml:"timing" json:"timing"`
	Notification struct {
		BotAPI string `yaml:"BotAPI" json:"BotAPI"`
		ChatID int64  `yaml:"ChatID" json:"ChatID"`
	} `yaml:"notification" json:"Notification"`
	Domains []string `yaml:"domains" json:"domains"`
}

func Load(configFile string) (config *Config, err error) {
	config = NewConfig()

	switch filepath.Ext(configFile) {
	case ".json":
		if err = LoadJSONConfig(&configFile, config); err != nil {
			return
		}
	case ".yaml":
		if err = LoadYamlConfig(&configFile, config); err != nil {
			return
		}
	default:
		return nil, fmt.Errorf("invalid format of configuration file")
	}

	return config, nil
}

func LoadJSONConfig(configFile *string, config *Config) error {
	contents, err := os.ReadFile(*configFile)
	if err != nil {
		fmt.Printf("Error read config file: %s\n", err)
		os.Exit(1)
	}

	err = json.Unmarshal(contents, config)
	if err != nil {
		return fmt.Errorf("invalid json: %s", err)
	}

	if !validateURL(config.Domains) {
		return fmt.Errorf("необходимо ввести валидный URL")
	}

	return nil
}

func LoadYamlConfig(configFile *string, config *Config) error {
	contents, err := os.ReadFile(*configFile)
	if err != nil {
		fmt.Printf("Error read config file: %s\n", err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(contents, config)
	if err != nil {
		return fmt.Errorf("invalid yaml: %s", err) // ST1005: error strings should not end with punctuation or a newline (stylecheck)
	}

	if !validateURL(config.Domains) {
		return fmt.Errorf("необходимо ввести валидный URL")
	}

	return nil
}

func ParseFlags() (string, error) {
	var configPath string

	flag.StringVar(&configPath, "config-file", "./config.yaml", "path to config file")

	flag.Parse()

	if err := ValidateConfigPath(configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

func ValidateConfigPath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a normal file", path)
	}
	return nil
}

func isURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func validateURL(domains []string) bool {
	for _, d := range domains {
		if !isURL(d) {
			return false
		}
	}

	return true
}

func NewConfig() *Config {
	return &Config{}
}
