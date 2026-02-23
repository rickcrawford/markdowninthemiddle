package config

import (
	"time"

	"github.com/spf13/viper"
)

// Config holds all proxy configuration.
type Config struct {
	Proxy       ProxyConfig      `mapstructure:"proxy"`
	TLS         TLSConfig        `mapstructure:"tls"`
	Conversion  ConversionConfig `mapstructure:"conversion"`
	MaxBodySize int64            `mapstructure:"max_body_size"`
	Cache       CacheConfig      `mapstructure:"cache"`
	Output      OutputConfig     `mapstructure:"output"`
	LogLevel    string           `mapstructure:"log_level"`
}

type ProxyConfig struct {
	Addr         string        `mapstructure:"addr"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type TLSConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	CertFile     string `mapstructure:"cert_file"`
	KeyFile      string `mapstructure:"key_file"`
	AutoCert     bool   `mapstructure:"auto_cert"`
	AutoCertHost string `mapstructure:"auto_cert_host"`
	AutoCertDir  string `mapstructure:"auto_cert_dir"`
	Insecure     bool   `mapstructure:"insecure"`
}

type ConversionConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	TiktokenEncoding string `mapstructure:"tiktoken_encoding"`
	NegotiateOnly    bool   `mapstructure:"negotiate_only"`
}

type CacheConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	Dir            string `mapstructure:"dir"`
	RespectHeaders bool   `mapstructure:"respect_headers"`
}

type OutputConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Dir     string `mapstructure:"dir"`
}

// Load reads configuration from the given file path (or default locations)
// and environment variables, then unmarshals into a Config struct.
func Load(cfgFile string) (*Config, error) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.markdowninthemiddle")
		viper.AddConfigPath("/etc/markdowninthemiddle")
	}

	// Environment variable overrides (e.g. MITM_PROXY_ADDR)
	viper.SetEnvPrefix("MITM")
	viper.AutomaticEnv()

	// Defaults
	viper.SetDefault("proxy.addr", ":8080")
	viper.SetDefault("proxy.read_timeout", "30s")
	viper.SetDefault("proxy.write_timeout", "30s")
	viper.SetDefault("tls.enabled", false)
	viper.SetDefault("tls.auto_cert", true)
	viper.SetDefault("tls.auto_cert_host", "localhost")
	viper.SetDefault("tls.auto_cert_dir", "./certs")
	viper.SetDefault("tls.insecure", false)
	viper.SetDefault("conversion.enabled", true)
	viper.SetDefault("conversion.tiktoken_encoding", "cl100k_base")
	viper.SetDefault("conversion.negotiate_only", false)
	viper.SetDefault("max_body_size", 10485760)
	viper.SetDefault("cache.enabled", false)
	viper.SetDefault("cache.respect_headers", true)
	viper.SetDefault("output.enabled", false)
	viper.SetDefault("output.dir", "")
	viper.SetDefault("log_level", "info")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
