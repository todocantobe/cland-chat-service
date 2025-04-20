package config

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"strconv"
)

// Config 应用配置
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Log    LogConfig    `mapstructure:"log"`
	Redis  RedisConfig  `mapstructure:"redis"`
	DB     DBConfig     `mapstructure:"db"`
	WS     WSConfig     `mapstructure:"ws"` // 新增
}

// WSConfig WebSocket 配置
type WSConfig struct {
	Port int `mapstructure:"port"` // WebSocket 端口
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Mode string `mapstructure:"mode"`
	Host string `mapstructure:"host"` // 新增
	Port int    `mapstructure:"port"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"` // 新增字段
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// DBConfig 数据库配置
type DBConfig struct {
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
}

// Load 加载配置
func Load() (*Config, error) {
	// 设置配置文件路径
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./conf")

	// 设置环境变量前缀
	viper.SetEnvPrefix("CLAND")
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// 解析配置
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	// 从环境变量覆盖配置
	overrideFromEnv(&cfg)

	return &cfg, nil
}

// overrideFromEnv 从环境变量覆盖配置
func overrideFromEnv(cfg *Config) {
	if port := os.Getenv("CLAND_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}

	if mode := os.Getenv("CLAND_SERVER_MODE"); mode != "" {
		cfg.Server.Mode = mode
	}

	if level := os.Getenv("CLAND_LOG_LEVEL"); level != "" {
		cfg.Log.Level = level
	}

	if compress := os.Getenv("CLAND_LOG_COMPRESS"); compress != "" {
		cfg.Log.Compress = compress == "true"
	}

	if host := os.Getenv("CLAND_SERVER_HOST"); host != "" {
		cfg.Server.Host = host
	}
}
