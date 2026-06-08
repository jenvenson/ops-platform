package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// JenkinsConfig Jenkins 全局配置
type JenkinsConfig struct {
	URL          string `mapstructure:"url"`           // Jenkins 基础地址
	Username     string `mapstructure:"username"`      // Jenkins 用户名
	Token        string `mapstructure:"token"`         // Jenkins API Token
	Timeout      int    `mapstructure:"timeout"`       // 请求超时（秒）
	PollInterval int    `mapstructure:"poll_interval"` // 状态轮询间隔（秒）
}

// GrafanaConfig Grafana 配置
type GrafanaConfig struct {
	URL      string `mapstructure:"url"`      // Grafana 地址
	Username string `mapstructure:"username"` // Grafana 用户名
	Password string `mapstructure:"password"` // Grafana 密码
}

type AssistantConfig struct {
	Enabled            bool    `mapstructure:"enabled"`
	Provider           string  `mapstructure:"provider"`
	OllamaBaseURL      string  `mapstructure:"ollama_base_url"`
	OllamaChatModel    string  `mapstructure:"ollama_chat_model"`
	OllamaEmbedModel   string  `mapstructure:"ollama_embed_model"`
	MaxContextMessages int     `mapstructure:"max_context_messages"`
	MaxMessageRunes    int     `mapstructure:"max_message_runes"`
	RequestTimeoutSec  int     `mapstructure:"request_timeout_sec"`
	RateLimitPerMinute int     `mapstructure:"rate_limit_per_minute"`
	TopK               int     `mapstructure:"top_k"`
	Temperature        float64 `mapstructure:"temperature"`
}

type Config struct {
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		Name     string `mapstructure:"name"`
	} `mapstructure:"database"`
	Redis struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Password string `mapstructure:"password"`
	} `mapstructure:"redis"`
	JWT struct {
		Secret string `mapstructure:"secret"`
	} `mapstructure:"jwt"`
	Jenkins   JenkinsConfig   `mapstructure:"jenkins"`
	Grafana   GrafanaConfig   `mapstructure:"grafana"`
	Assistant AssistantConfig `mapstructure:"assistant"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// 设置默认值
	viper.SetDefault("port", 8080)
	viper.SetDefault("log_level", "info")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("database.user", "root")
	// Jenkins 默认值
	viper.SetDefault("jenkins.timeout", 30)
	viper.SetDefault("jenkins.poll_interval", 5)
	viper.SetDefault("assistant.enabled", false)
	viper.SetDefault("assistant.provider", "ollama")
	viper.SetDefault("assistant.ollama_base_url", "http://127.0.0.1:11434")
	viper.SetDefault("assistant.ollama_chat_model", "qwen3:8b")
	viper.SetDefault("assistant.ollama_embed_model", "qwen3-embedding:4b")
	viper.SetDefault("assistant.max_context_messages", 12)
	viper.SetDefault("assistant.max_message_runes", 1000)
	viper.SetDefault("assistant.request_timeout_sec", 20)
	viper.SetDefault("assistant.rate_limit_per_minute", 30)
	viper.SetDefault("assistant.top_k", 4)
	viper.SetDefault("assistant.temperature", 0.2)

	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不存在，使用环境变量
		viper.AutomaticEnv()
	}

	var cfg Config
	viper.Unmarshal(&cfg)

	// 从环境变量读取配置（环境变量优先级高于配置文件）
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		cfg.Database.Host = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		var port int
		fmt.Sscanf(dbPort, "%d", &port)
		if port > 0 {
			cfg.Database.Port = port
		}
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		cfg.Database.User = dbUser
	}
	if dbPass := os.Getenv("DB_PASSWORD"); dbPass != "" {
		cfg.Database.Password = dbPass
	}
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		cfg.Redis.Host = redisHost
	}
	if redisPass := os.Getenv("REDIS_PASSWORD"); redisPass != "" {
		cfg.Redis.Password = redisPass
	}
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		cfg.JWT.Secret = jwtSecret
	}
	// Jenkins 配置从环境变量读取
	if jenkinsURL := os.Getenv("JENKINS_URL"); jenkinsURL != "" {
		cfg.Jenkins.URL = jenkinsURL
	}
	if jenkinsUser := os.Getenv("JENKINS_USERNAME"); jenkinsUser != "" {
		cfg.Jenkins.Username = jenkinsUser
	}
	if jenkinsToken := os.Getenv("JENKINS_TOKEN"); jenkinsToken != "" {
		cfg.Jenkins.Token = jenkinsToken
	}
	// Grafana 配置
	if grafanaURL := os.Getenv("GRAFANA_URL"); grafanaURL != "" {
		cfg.Grafana.URL = grafanaURL
	}
	if grafanaUser := os.Getenv("GRAFANA_USERNAME"); grafanaUser != "" {
		cfg.Grafana.Username = grafanaUser
	}
	if grafanaPass := os.Getenv("GRAFANA_PASSWORD"); grafanaPass != "" {
		cfg.Grafana.Password = grafanaPass
	}
	if assistantProvider := os.Getenv("ASSISTANT_PROVIDER"); assistantProvider != "" {
		cfg.Assistant.Provider = assistantProvider
	}
	if assistantEnabled := os.Getenv("ASSISTANT_ENABLED"); assistantEnabled != "" {
		switch strings.ToLower(strings.TrimSpace(assistantEnabled)) {
		case "1", "true", "yes", "on":
			cfg.Assistant.Enabled = true
		case "0", "false", "no", "off":
			cfg.Assistant.Enabled = false
		}
	}
	if ollamaBaseURL := os.Getenv("OLLAMA_BASE_URL"); ollamaBaseURL != "" {
		cfg.Assistant.OllamaBaseURL = ollamaBaseURL
	}
	if ollamaChatModel := os.Getenv("OLLAMA_CHAT_MODEL"); ollamaChatModel != "" {
		cfg.Assistant.OllamaChatModel = ollamaChatModel
	}
	if ollamaEmbedModel := os.Getenv("OLLAMA_EMBED_MODEL"); ollamaEmbedModel != "" {
		cfg.Assistant.OllamaEmbedModel = ollamaEmbedModel
	}
	if assistantMaxMessageRunes := os.Getenv("ASSISTANT_MAX_MESSAGE_RUNES"); assistantMaxMessageRunes != "" {
		fmt.Sscanf(assistantMaxMessageRunes, "%d", &cfg.Assistant.MaxMessageRunes)
	}
	if assistantTimeout := os.Getenv("ASSISTANT_REQUEST_TIMEOUT_SEC"); assistantTimeout != "" {
		fmt.Sscanf(assistantTimeout, "%d", &cfg.Assistant.RequestTimeoutSec)
	}
	if assistantRateLimit := os.Getenv("ASSISTANT_RATE_LIMIT_PER_MINUTE"); assistantRateLimit != "" {
		fmt.Sscanf(assistantRateLimit, "%d", &cfg.Assistant.RateLimitPerMinute)
	}

	// 验证配置
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Database.Name == "" {
		return errors.New("database name is required")
	}

	if cfg.JWT.Secret == "" {
		return errors.New("JWT_SECRET environment variable is required")
	}

	return nil
}
