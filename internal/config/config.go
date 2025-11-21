package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port int `mapstructure:"port"`
	} `mapstructure:"server"`

	Database DatabaseConfig `mapstructure:"db"`

	Exchages struct {
		Pacifica struct {
			BaseURL string `mapstructure:"base_url"`
		} `mapstructure:"pacifica"`
		Lighter struct {
			BaseURL string `mapstructure:"base_url"`
		} `mapstructure:"lighter"`
		Extended struct {
			BaseURL string `mapstructure:"base_url"`
		} `mapstructure:"extended"`
		Hibachi struct {
			BaseURL string `mapstructure:"base_url"`
		} `mapstructure:"hibachi"`
		Backpack struct {
			BaseURL string `mapstructure:"base_url"`
		} `mapstructure:"backpack"`
	} `mapstructure:"exchanges"`

	Proxy string `mapstructure:"proxy"`

	Tracker struct {
		UpdateInterval time.Duration `mapstructure:"update_interval"`
	} `mapstructure:"tracker"`

	Logger struct {
		Level    string `mapstructure:"level"`
		Encoding string `mapstructure:"encoding"`
	} `mapstructure:"log"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

func Load(path string) (*Config, error) {
	viper.AddConfigPath(path)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unmarshall config: %w", err)
	}

	return &config, nil
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}
