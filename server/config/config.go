package config

import (
	"github.com/spf13/viper"
)

type ServerConfig struct {
	Port            int    `mapstructure:"port"`
	LogFile         string `mapstructure:"log_file"`
	AgentUpdateFreq int    `mapstructure:"agent_update_freq"`
	TLSEnabled      bool   `mapstructure:"tls_enabled"`
	TLSCert         string `mapstructure:"tls_cert"`
	TLSKey          string `mapstructure:"tls_key"`
}

func LoadConfig() (*ServerConfig, error) {
	viper.SetConfigName("server_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	// Default values
	viper.SetDefault("port", 8080)
	viper.SetDefault("log_file", "server.log")
	viper.SetDefault("agent_update_freq", 30)
	viper.SetDefault("tls_enabled", false)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Create default config
			return &ServerConfig{
				Port:            8080,
				LogFile:         "server.log",
				AgentUpdateFreq: 30,
				TLSEnabled:      false,
			}, nil
		}
		return nil, err
	}

	var config ServerConfig
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func SaveConfig(config *ServerConfig) error {
	viper.Set("port", config.Port)
	viper.Set("log_file", config.LogFile)
	viper.Set("agent_update_freq", config.AgentUpdateFreq)
	viper.Set("tls_enabled", config.TLSEnabled)
	viper.Set("tls_cert", config.TLSCert)
	viper.Set("tls_key", config.TLSKey)

	return viper.WriteConfig()
}
