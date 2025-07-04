package config

import (
	"fmt"
	"time"
	
	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Scanner   ScannerConfig   `mapstructure:"scanner"`
	WebSocket WebSocketConfig `mapstructure:"websocket"`
	API       APIConfig       `mapstructure:"api"`
	Log       LogConfig       `mapstructure:"log"`
	Security  SecurityConfig  `mapstructure:"security"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Env     string `mapstructure:"env"`
	Debug   bool   `mapstructure:"debug"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type            string        `mapstructure:"type"`
	DSN             string        `mapstructure:"dsn"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	LogLevel        string        `mapstructure:"log_level"`
}

// ScannerConfig 扫码枪配置
type ScannerConfig struct {
	TimeoutMS  int  `mapstructure:"timeout_ms"`
	MinLength  int  `mapstructure:"min_length"`
	MaxLength  int  `mapstructure:"max_length"`
	EnableHook bool `mapstructure:"enable_hook"`
}

// WebSocketConfig WebSocket配置
type WebSocketConfig struct {
	Path            string        `mapstructure:"path"`
	ReadBufferSize  int           `mapstructure:"read_buffer_size"`
	WriteBufferSize int           `mapstructure:"write_buffer_size"`
	CheckOrigin     bool          `mapstructure:"check_origin"`
	PingPeriod      time.Duration `mapstructure:"ping_period"`
	PongWait        time.Duration `mapstructure:"pong_wait"`
	WriteWait       time.Duration `mapstructure:"write_wait"`
}

// APIConfig API配置
type APIConfig struct {
	Prefix      string      `mapstructure:"prefix"`
	EnableCORS  bool        `mapstructure:"enable_cors"`
	CORSOrigins []string    `mapstructure:"cors_origins"`
	RateLimit   RateLimit   `mapstructure:"rate_limit"`
}

// RateLimit 限流配置
type RateLimit struct {
	Enable             bool `mapstructure:"enable"`
	RequestsPerMinute  int  `mapstructure:"requests_per_minute"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	EnableAuth bool          `mapstructure:"enable_auth"`
	JWTSecret  string        `mapstructure:"jwt_secret"`
	JWTExpire  time.Duration `mapstructure:"jwt_expire"`
	APIKey     string        `mapstructure:"api_key"`
}

// Load 加载配置
func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")
	
	// 设置环境变量前缀
	viper.SetEnvPrefix("SCANNER")
	viper.AutomaticEnv()
	
	// 设置默认值
	setDefaults()
	
	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	
	return &config, nil
}

// setDefaults 设置默认值
func setDefaults() {
	// App defaults
	viper.SetDefault("app.name", "Barcode Scanner Monitor")
	viper.SetDefault("app.version", "2.0.0")
	viper.SetDefault("app.env", "development")
	viper.SetDefault("app.debug", true)
	
	// Server defaults
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "60s")
	
	// Database defaults
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.dsn", "./data/scanner.db")
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.max_open_conns", 100)
	viper.SetDefault("database.conn_max_lifetime", "3600s")
	viper.SetDefault("database.log_level", "info")
	
	// Scanner defaults
	viper.SetDefault("scanner.timeout_ms", 100)
	viper.SetDefault("scanner.min_length", 3)
	viper.SetDefault("scanner.max_length", 50)
	viper.SetDefault("scanner.enable_hook", true)
	
	// WebSocket defaults
	viper.SetDefault("websocket.path", "/ws")
	viper.SetDefault("websocket.read_buffer_size", 1024)
	viper.SetDefault("websocket.write_buffer_size", 1024)
	viper.SetDefault("websocket.check_origin", true)
	viper.SetDefault("websocket.ping_period", "54s")
	viper.SetDefault("websocket.pong_wait", "60s")
	viper.SetDefault("websocket.write_wait", "10s")
	
	// API defaults
	viper.SetDefault("api.prefix", "/api/v1")
	viper.SetDefault("api.enable_cors", true)
	viper.SetDefault("api.cors_origins", []string{"*"})
	viper.SetDefault("api.rate_limit.enable", true)
	viper.SetDefault("api.rate_limit.requests_per_minute", 100)
	
	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("log.file_path", "./logs/app.log")
	viper.SetDefault("log.max_size", 100)
	viper.SetDefault("log.max_backups", 3)
	viper.SetDefault("log.max_age", 28)
	viper.SetDefault("log.compress", true)
	
	// Security defaults
	viper.SetDefault("security.enable_auth", false)
	viper.SetDefault("security.jwt_secret", "your-secret-key")
	viper.SetDefault("security.jwt_expire", "24h")
	viper.SetDefault("security.api_key", "your-api-key")
}

// GetServerAddr 获取服务器地址
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// IsDevelopment 是否为开发环境
func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

// IsProduction 是否为生产环境
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}