package chi

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Config 应用配置
type Config struct {
	// 应用基本信息
	Name        string `json:"name"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
	BuildTime   string `json:"build_time,omitempty"`
	GitCommit   string `json:"git_commit,omitempty"`
	
	// 服务器配置
	Mode            string        `json:"mode"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	MaxHeaderBytes  int           `json:"max_header_bytes"`
	
	// 中间件开关
	EnableRecovery  bool `json:"enable_recovery"`
	EnableCORS      bool `json:"enable_cors"`
	EnableLogger    bool `json:"enable_logger"`
	EnableMetrics   bool `json:"enable_metrics"`
	EnableSecurity  bool `json:"enable_security"`
	EnableRequestID bool `json:"enable_request_id"`
	EnableRateLimit bool `json:"enable_rate_limit"`
	
	// 组件配置
	CORS      *CORSConfig      `json:"cors,omitempty"`
	Security  *SecurityConfig  `json:"security,omitempty"`
	RateLimit *RateLimitConfig `json:"rate_limit,omitempty"`
	WebSocket *WebSocketConfig `json:"websocket,omitempty"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Name:        "chi-app",
		Version:     "1.0.0",
		Environment: "development",
		Mode:        gin.ReleaseMode,
		
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
		
		EnableRecovery:  true,
		EnableCORS:      true,
		EnableLogger:    true,
		EnableMetrics:   true,
		EnableSecurity:  true,
		EnableRequestID: true,
		EnableRateLimit: false,
		
		CORS:      DefaultCORSConfig(),
		Security:  DefaultSecurityConfig(),
		RateLimit: DefaultRateLimitConfig(),
		WebSocket: DefaultWebSocketConfig(),
	}
}

// DevelopmentConfig 开发环境配置
func DevelopmentConfig() *Config {
	config := DefaultConfig()
	config.Mode = gin.DebugMode
	config.Environment = "development"
	config.EnableLogger = true
	config.CORS.AllowOrigins = []string{"*"}
	config.Security.ContentSecurityPolicy = ""
	return config
}

// ProductionConfig 生产环境配置
func ProductionConfig() *Config {
	config := DefaultConfig()
	config.Mode = gin.ReleaseMode
	config.Environment = "production"
	config.EnableLogger = true
	config.EnableRateLimit = true
	config.ReadTimeout = 10 * time.Second
	config.WriteTimeout = 10 * time.Second
	return config
}

// TestConfig 测试环境配置
func TestConfig() *Config {
	config := DefaultConfig()
	config.Mode = gin.TestMode
	config.Environment = "test"
	config.EnableLogger = false
	config.EnableMetrics = false
	return config
}

// Option 配置选项函数
type Option func(*Config)

// WithName 设置应用名称
func WithName(name string) Option {
	return func(c *Config) {
		c.Name = name
	}
}

// WithVersion 设置应用版本
func WithVersion(version string) Option {
	return func(c *Config) {
		c.Version = version
	}
}

// WithEnvironment 设置环境
func WithEnvironment(env string) Option {
	return func(c *Config) {
		c.Environment = env
	}
}

// WithMode 设置Gin模式
func WithMode(mode string) Option {
	return func(c *Config) {
		c.Mode = mode
	}
}

// WithBuildInfo 设置构建信息
func WithBuildInfo(buildTime, gitCommit string) Option {
	return func(c *Config) {
		c.BuildTime = buildTime
		c.GitCommit = gitCommit
	}
}

// WithTimeouts 设置超时配置
func WithTimeouts(read, write, idle time.Duration) Option {
	return func(c *Config) {
		c.ReadTimeout = read
		c.WriteTimeout = write
		c.IdleTimeout = idle
	}
}

// WithCORS 设置CORS配置
func WithCORS(cors *CORSConfig) Option {
	return func(c *Config) {
		c.EnableCORS = true
		c.CORS = cors
	}
}

// WithSecurity 设置安全配置
func WithSecurity(security *SecurityConfig) Option {
	return func(c *Config) {
		c.EnableSecurity = true
		c.Security = security
	}
}

// WithRateLimit 设置限流配置
func WithRateLimit(rateLimit *RateLimitConfig) Option {
	return func(c *Config) {
		c.EnableRateLimit = true
		c.RateLimit = rateLimit
	}
}

// WithWebSocket 设置WebSocket配置
func WithWebSocket(ws *WebSocketConfig) Option {
	return func(c *Config) {
		c.WebSocket = ws
	}
}

// DisableRecovery 禁用恢复中间件
func DisableRecovery() Option {
	return func(c *Config) {
		c.EnableRecovery = false
	}
}

// DisableCORS 禁用CORS中间件
func DisableCORS() Option {
	return func(c *Config) {
		c.EnableCORS = false
	}
}

// DisableLogger 禁用日志中间件
func DisableLogger() Option {
	return func(c *Config) {
		c.EnableLogger = false
	}
}

// DisableMetrics 禁用指标中间件
func DisableMetrics() Option {
	return func(c *Config) {
		c.EnableMetrics = false
	}
}

// DisableSecurity 禁用安全中间件
func DisableSecurity() Option {
	return func(c *Config) {
		c.EnableSecurity = false
	}
}

// DisableRequestID 禁用请求ID中间件
func DisableRequestID() Option {
	return func(c *Config) {
		c.EnableRequestID = false
	}
}

// EnableRateLimit 启用限流中间件
func EnableRateLimit() Option {
	return func(c *Config) {
		c.EnableRateLimit = true
	}
}

// CORSConfig CORS配置
type CORSConfig struct {
	AllowOrigins     []string      `json:"allow_origins"`
	AllowMethods     []string      `json:"allow_methods"`
	AllowHeaders     []string      `json:"allow_headers"`
	ExposeHeaders    []string      `json:"expose_headers"`
	AllowCredentials bool          `json:"allow_credentials"`
	MaxAge           time.Duration `json:"max_age"`
}

// DefaultCORSConfig 默认CORS配置
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://localhost:8080"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders: []string{
			"Origin", "Content-Length", "Content-Type", "Authorization",
			"X-Requested-With", "Accept", "Accept-Encoding", "X-CSRF-Token",
		},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	ContentTypeNosniff    bool   `json:"content_type_nosniff"`
	XFrameOptions         string `json:"x_frame_options"`
	XSSProtection         string `json:"xss_protection"`
	ContentSecurityPolicy string `json:"content_security_policy"`
	ReferrerPolicy        string `json:"referrer_policy"`
	HSTSMaxAge            int    `json:"hsts_max_age"`
	HSTSIncludeSubdomains bool   `json:"hsts_include_subdomains"`
}

// DefaultSecurityConfig 默认安全配置
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		ContentTypeNosniff:    true,
		XFrameOptions:         "DENY",
		XSSProtection:         "1; mode=block",
		ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubdomains: true,
	}
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Rate   int           `json:"rate"`    // 每秒请求数
	Burst  int           `json:"burst"`   // 突发请求数
	Window time.Duration `json:"window"`  // 时间窗口
}

// DefaultRateLimitConfig 默认限流配置
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Rate:   100,
		Burst:  50,
		Window: time.Minute,
	}
}

// WebSocketConfig WebSocket配置
type WebSocketConfig struct {
	ReadBufferSize    int           `json:"read_buffer_size"`
	WriteBufferSize   int           `json:"write_buffer_size"`
	WriteWait         time.Duration `json:"write_wait"`
	PongWait          time.Duration `json:"pong_wait"`
	PingPeriod        time.Duration `json:"ping_period"`
	MaxMessageSize    int64         `json:"max_message_size"`
	EnableCompression bool          `json:"enable_compression"`
}

// DefaultWebSocketConfig 默认WebSocket配置
func DefaultWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		WriteWait:         10 * time.Second,
		PongWait:          60 * time.Second,
		PingPeriod:        54 * time.Second,
		MaxMessageSize:    512,
		EnableCompression: false,
	}
}