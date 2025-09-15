# 微服务配置管理

## 概述

Chi 框架提供了强大的配置管理系统，支持多环境配置、动态配置更新、配置验证和配置中心集成。配置管理系统采用分层设计，支持从本地文件、环境变量、远程配置中心等多种来源加载配置。

## 配置架构

### 配置层次结构
```
┌─────────────────────────────────────────────────────────────────┐
│                Configuration Management                          │
│                                                                 │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐          │
│  │Environment  │   │   Local     │   │   Remote    │          │
│  │ Variables   │   │   Files     │   │  Config     │          │
│  │             │   │             │   │  Center     │          │
│  └─────────────┘   └─────────────┘   └─────────────┘          │
│          │                 │                 │                 │
│          └─────────────────┼─────────────────┘                 │
│                            │                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │              Configuration Loader                          ││
│  └─────────────────────────────────────────────────────────────┘│
│                            │                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │              Configuration Manager                         ││
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐          ││
│  │  │Validator│ │ Watcher │ │ Cache   │ │Resolver │          ││
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘          ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

### 配置优先级

1. **命令行参数** (最高优先级)
2. **环境变量**
3. **远程配置中心**
4. **本地配置文件**
5. **默认配置** (最低优先级)

## 配置结构定义

### 基础配置结构

```go
package config

import (
    "time"
)

// 应用配置
type Config struct {
    // 服务基础配置
    Service  ServiceConfig  `yaml:"service" mapstructure:"service"`
    Server   ServerConfig   `yaml:"server" mapstructure:"server"`
    Database DatabaseConfig `yaml:"database" mapstructure:"database"`
    Redis    RedisConfig    `yaml:"redis" mapstructure:"redis"`

    // 微服务配置
    ServiceDiscovery ServiceDiscoveryConfig `yaml:"service_discovery" mapstructure:"service_discovery"`
    Events          EventsConfig           `yaml:"events" mapstructure:"events"`
    Messaging       MessagingConfig        `yaml:"messaging" mapstructure:"messaging"`

    // 治理配置
    Governance    GovernanceConfig    `yaml:"governance" mapstructure:"governance"`
    Observability ObservabilityConfig `yaml:"observability" mapstructure:"observability"`

    // 安全配置
    Security SecurityConfig `yaml:"security" mapstructure:"security"`

    // 自定义配置
    Custom map[string]interface{} `yaml:"custom" mapstructure:"custom"`
}

// 服务配置
type ServiceConfig struct {
    Name        string            `yaml:"name" mapstructure:"name"`
    Version     string            `yaml:"version" mapstructure:"version"`
    Environment string            `yaml:"environment" mapstructure:"environment"`
    Description string            `yaml:"description" mapstructure:"description"`
    Tags        []string          `yaml:"tags" mapstructure:"tags"`
    Metadata    map[string]string `yaml:"metadata" mapstructure:"metadata"`
}

// 服务器配置
type ServerConfig struct {
    Host         string        `yaml:"host" mapstructure:"host"`
    Port         int           `yaml:"port" mapstructure:"port"`
    Mode         string        `yaml:"mode" mapstructure:"mode"` // debug, release, test
    ReadTimeout  time.Duration `yaml:"read_timeout" mapstructure:"read_timeout"`
    WriteTimeout time.Duration `yaml:"write_timeout" mapstructure:"write_timeout"`
    IdleTimeout  time.Duration `yaml:"idle_timeout" mapstructure:"idle_timeout"`
    MaxConns     int           `yaml:"max_conns" mapstructure:"max_conns"`
}

// 数据库配置
type DatabaseConfig struct {
    Driver          string        `yaml:"driver" mapstructure:"driver"`
    Host            string        `yaml:"host" mapstructure:"host"`
    Port            int           `yaml:"port" mapstructure:"port"`
    Username        string        `yaml:"username" mapstructure:"username"`
    Password        string        `yaml:"password" mapstructure:"password"`
    Database        string        `yaml:"database" mapstructure:"database"`
    SSLMode         string        `yaml:"sslmode" mapstructure:"sslmode"`
    MaxOpenConns    int           `yaml:"max_open_conns" mapstructure:"max_open_conns"`
    MaxIdleConns    int           `yaml:"max_idle_conns" mapstructure:"max_idle_conns"`
    ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" mapstructure:"conn_max_lifetime"`
    ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" mapstructure:"conn_max_idle_time"`
}

// Redis配置
type RedisConfig struct {
    Host        string        `yaml:"host" mapstructure:"host"`
    Port        int           `yaml:"port" mapstructure:"port"`
    Password    string        `yaml:"password" mapstructure:"password"`
    DB          int           `yaml:"db" mapstructure:"db"`
    PoolSize    int           `yaml:"pool_size" mapstructure:"pool_size"`
    MinIdleConns int          `yaml:"min_idle_conns" mapstructure:"min_idle_conns"`
    MaxConnAge  time.Duration `yaml:"max_conn_age" mapstructure:"max_conn_age"`
    PoolTimeout time.Duration `yaml:"pool_timeout" mapstructure:"pool_timeout"`
    IdleTimeout time.Duration `yaml:"idle_timeout" mapstructure:"idle_timeout"`
}

// 服务发现配置
type ServiceDiscoveryConfig struct {
    Type   string       `yaml:"type" mapstructure:"type"` // consul, etcd, nacos
    Consul ConsulConfig `yaml:"consul" mapstructure:"consul"`
    Etcd   EtcdConfig   `yaml:"etcd" mapstructure:"etcd"`
    Nacos  NacosConfig  `yaml:"nacos" mapstructure:"nacos"`
}

type ConsulConfig struct {
    Address    string        `yaml:"address" mapstructure:"address"`
    Token      string        `yaml:"token" mapstructure:"token"`
    Datacenter string        `yaml:"datacenter" mapstructure:"datacenter"`
    Timeout    time.Duration `yaml:"timeout" mapstructure:"timeout"`
}

type EtcdConfig struct {
    Endpoints   []string      `yaml:"endpoints" mapstructure:"endpoints"`
    Username    string        `yaml:"username" mapstructure:"username"`
    Password    string        `yaml:"password" mapstructure:"password"`
    DialTimeout time.Duration `yaml:"dial_timeout" mapstructure:"dial_timeout"`
}

type NacosConfig struct {
    ServerConfigs []NacosServerConfig `yaml:"server_configs" mapstructure:"server_configs"`
    ClientConfig  NacosClientConfig   `yaml:"client_config" mapstructure:"client_config"`
}

type NacosServerConfig struct {
    IpAddr      string `yaml:"ip" mapstructure:"ip"`
    Port        uint64 `yaml:"port" mapstructure:"port"`
    ContextPath string `yaml:"context_path" mapstructure:"context_path"`
}

type NacosClientConfig struct {
    NamespaceId string `yaml:"namespace_id" mapstructure:"namespace_id"`
    Username    string `yaml:"username" mapstructure:"username"`
    Password    string `yaml:"password" mapstructure:"password"`
}

// 事件配置
type EventsConfig struct {
    Type       string            `yaml:"type" mapstructure:"type"` // memory, redis, kafka, rabbitmq
    BufferSize int               `yaml:"buffer_size" mapstructure:"buffer_size"`
    Workers    int               `yaml:"workers" mapstructure:"workers"`
    Redis      RedisConfig       `yaml:"redis" mapstructure:"redis"`
    Kafka      KafkaConfig       `yaml:"kafka" mapstructure:"kafka"`
    RabbitMQ   RabbitMQConfig    `yaml:"rabbitmq" mapstructure:"rabbitmq"`
    Custom     map[string]interface{} `yaml:"custom" mapstructure:"custom"`
}

type KafkaConfig struct {
    Brokers       []string `yaml:"brokers" mapstructure:"brokers"`
    ConsumerGroup string   `yaml:"consumer_group" mapstructure:"consumer_group"`
    Version       string   `yaml:"version" mapstructure:"version"`
}

type RabbitMQConfig struct {
    URL string `yaml:"url" mapstructure:"url"`
}

// 治理配置
type GovernanceConfig struct {
    CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker" mapstructure:"circuit_breaker"`
    RateLimit      RateLimitConfig      `yaml:"rate_limit" mapstructure:"rate_limit"`
    Timeout        TimeoutConfig        `yaml:"timeout" mapstructure:"timeout"`
    Retry          RetryConfig          `yaml:"retry" mapstructure:"retry"`
    Bulkhead       BulkheadConfig       `yaml:"bulkhead" mapstructure:"bulkhead"`
}

// 可观测性配置
type ObservabilityConfig struct {
    Tracing     TracingConfig     `yaml:"tracing" mapstructure:"tracing"`
    Metrics     MetricsConfig     `yaml:"metrics" mapstructure:"metrics"`
    Logging     LoggingConfig     `yaml:"logging" mapstructure:"logging"`
    HealthCheck HealthCheckConfig `yaml:"health_check" mapstructure:"health_check"`
}

// 安全配置
type SecurityConfig struct {
    JWT        JWTConfig        `yaml:"jwt" mapstructure:"jwt"`
    TLS        TLSConfig        `yaml:"tls" mapstructure:"tls"`
    CORS       CORSConfig       `yaml:"cors" mapstructure:"cors"`
    RateLimit  RateLimitConfig  `yaml:"rate_limit" mapstructure:"rate_limit"`
    Encryption EncryptionConfig `yaml:"encryption" mapstructure:"encryption"`
}

type JWTConfig struct {
    Secret     string        `yaml:"secret" mapstructure:"secret"`
    Expiry     time.Duration `yaml:"expiry" mapstructure:"expiry"`
    RefreshTTL time.Duration `yaml:"refresh_ttl" mapstructure:"refresh_ttl"`
    Issuer     string        `yaml:"issuer" mapstructure:"issuer"`
    Audience   string        `yaml:"audience" mapstructure:"audience"`
}

type TLSConfig struct {
    Enabled  bool   `yaml:"enabled" mapstructure:"enabled"`
    CertFile string `yaml:"cert_file" mapstructure:"cert_file"`
    KeyFile  string `yaml:"key_file" mapstructure:"key_file"`
    CAFile   string `yaml:"ca_file" mapstructure:"ca_file"`
}

type CORSConfig struct {
    AllowOrigins     []string `yaml:"allow_origins" mapstructure:"allow_origins"`
    AllowMethods     []string `yaml:"allow_methods" mapstructure:"allow_methods"`
    AllowHeaders     []string `yaml:"allow_headers" mapstructure:"allow_headers"`
    ExposeHeaders    []string `yaml:"expose_headers" mapstructure:"expose_headers"`
    AllowCredentials bool     `yaml:"allow_credentials" mapstructure:"allow_credentials"`
    MaxAge           int      `yaml:"max_age" mapstructure:"max_age"`
}

type EncryptionConfig struct {
    Algorithm string `yaml:"algorithm" mapstructure:"algorithm"`
    Key       string `yaml:"key" mapstructure:"key"`
    IV        string `yaml:"iv" mapstructure:"iv"`
}
```

## 配置加载器

### 配置管理器

```go
package config

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"

    "github.com/spf13/viper"
    "github.com/fsnotify/fsnotify"
    "go.uber.org/zap"
)

// 配置管理器
type Manager struct {
    config   *Config
    viper    *viper.Viper
    logger   *zap.Logger
    watchers []ConfigWatcher
    mutex    sync.RWMutex

    // 远程配置
    remoteProvider RemoteConfigProvider
    updateChan     chan *Config
    stopChan       chan struct{}
}

// 配置监听器接口
type ConfigWatcher interface {
    OnConfigChange(oldConfig, newConfig *Config) error
}

// 远程配置提供者接口
type RemoteConfigProvider interface {
    Load(ctx context.Context) (*Config, error)
    Watch(ctx context.Context, callback func(*Config)) error
    Close() error
}

// 创建配置管理器
func NewManager(logger *zap.Logger) *Manager {
    v := viper.New()

    // 设置默认值
    setDefaults(v)

    return &Manager{
        viper:      v,
        logger:     logger,
        watchers:   make([]ConfigWatcher, 0),
        updateChan: make(chan *Config, 10),
        stopChan:   make(chan struct{}),
    }
}

// 加载配置
func (m *Manager) Load(configPaths ...string) (*Config, error) {
    m.mutex.Lock()
    defer m.mutex.Unlock()

    // 1. 设置配置文件路径
    for _, path := range configPaths {
        if path != "" {
            m.viper.SetConfigFile(path)
            break
        }
    }

    // 2. 设置默认搜索路径
    if m.viper.ConfigFileUsed() == "" {
        m.viper.SetConfigName("config")
        m.viper.SetConfigType("yaml")
        m.viper.AddConfigPath(".")
        m.viper.AddConfigPath("./config")
        m.viper.AddConfigPath("/etc/chi")
        m.viper.AddConfigPath("$HOME/.chi")
    }

    // 3. 设置环境变量
    m.viper.SetEnvPrefix("CHI")
    m.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    m.viper.AutomaticEnv()

    // 4. 读取配置文件
    if err := m.viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, fmt.Errorf("failed to read config file: %w", err)
        }
        m.logger.Warn("Config file not found, using defaults and environment variables")
    } else {
        m.logger.Info("Config file loaded", zap.String("file", m.viper.ConfigFileUsed()))
    }

    // 5. 解析配置
    var config Config
    if err := m.viper.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }

    // 6. 验证配置
    if err := m.validateConfig(&config); err != nil {
        return nil, fmt.Errorf("config validation failed: %w", err)
    }

    // 7. 处理敏感信息
    if err := m.processSensitiveData(&config); err != nil {
        return nil, fmt.Errorf("failed to process sensitive data: %w", err)
    }

    // 8. 从远程配置中心加载
    if m.remoteProvider != nil {
        remoteConfig, err := m.remoteProvider.Load(context.Background())
        if err != nil {
            m.logger.Warn("Failed to load remote config", zap.Error(err))
        } else {
            config = *m.mergeConfigs(&config, remoteConfig)
        }
    }

    m.config = &config
    return &config, nil
}

// 监听配置变化
func (m *Manager) Watch() error {
    // 监听本地文件变化
    m.viper.OnConfigChange(func(e fsnotify.Event) {
        m.logger.Info("Config file changed", zap.String("file", e.Name))

        var newConfig Config
        if err := m.viper.Unmarshal(&newConfig); err != nil {
            m.logger.Error("Failed to unmarshal changed config", zap.Error(err))
            return
        }

        if err := m.validateConfig(&newConfig); err != nil {
            m.logger.Error("Invalid config change", zap.Error(err))
            return
        }

        m.updateConfig(&newConfig)
    })
    m.viper.WatchConfig()

    // 监听远程配置变化
    if m.remoteProvider != nil {
        go m.remoteProvider.Watch(context.Background(), func(remoteConfig *Config) {
            m.mutex.RLock()
            localConfig := m.config
            m.mutex.RUnlock()

            mergedConfig := m.mergeConfigs(localConfig, remoteConfig)
            m.updateConfig(mergedConfig)
        })
    }

    // 启动配置更新处理协程
    go m.processConfigUpdates()

    return nil
}

// 更新配置
func (m *Manager) updateConfig(newConfig *Config) {
    select {
    case m.updateChan <- newConfig:
    default:
        m.logger.Warn("Config update channel is full, dropping update")
    }
}

// 处理配置更新
func (m *Manager) processConfigUpdates() {
    for {
        select {
        case newConfig := <-m.updateChan:
            m.mutex.Lock()
            oldConfig := m.config
            m.config = newConfig
            m.mutex.Unlock()

            // 通知所有监听器
            for _, watcher := range m.watchers {
                if err := watcher.OnConfigChange(oldConfig, newConfig); err != nil {
                    m.logger.Error("Config watcher error", zap.Error(err))
                }
            }

        case <-m.stopChan:
            return
        }
    }
}

// 添加配置监听器
func (m *Manager) AddWatcher(watcher ConfigWatcher) {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    m.watchers = append(m.watchers, watcher)
}

// 获取当前配置
func (m *Manager) GetConfig() *Config {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    return m.config
}

// 设置远程配置提供者
func (m *Manager) SetRemoteProvider(provider RemoteConfigProvider) {
    m.remoteProvider = provider
}

// 停止配置管理器
func (m *Manager) Stop() error {
    close(m.stopChan)
    if m.remoteProvider != nil {
        return m.remoteProvider.Close()
    }
    return nil
}

// 设置默认配置
func setDefaults(v *viper.Viper) {
    // 服务默认配置
    v.SetDefault("service.name", "chi-service")
    v.SetDefault("service.version", "1.0.0")
    v.SetDefault("service.environment", "development")

    // 服务器默认配置
    v.SetDefault("server.host", "0.0.0.0")
    v.SetDefault("server.port", 8080)
    v.SetDefault("server.mode", "debug")
    v.SetDefault("server.read_timeout", "30s")
    v.SetDefault("server.write_timeout", "30s")
    v.SetDefault("server.idle_timeout", "120s")

    // 数据库默认配置
    v.SetDefault("database.driver", "postgres")
    v.SetDefault("database.host", "localhost")
    v.SetDefault("database.port", 5432)
    v.SetDefault("database.max_open_conns", 50)
    v.SetDefault("database.max_idle_conns", 10)
    v.SetDefault("database.conn_max_lifetime", "1h")

    // Redis默认配置
    v.SetDefault("redis.host", "localhost")
    v.SetDefault("redis.port", 6379)
    v.SetDefault("redis.db", 0)
    v.SetDefault("redis.pool_size", 10)
    v.SetDefault("redis.pool_timeout", "4s")
    v.SetDefault("redis.idle_timeout", "5m")

    // 服务发现默认配置
    v.SetDefault("service_discovery.type", "consul")
    v.SetDefault("service_discovery.consul.address", "localhost:8500")
    v.SetDefault("service_discovery.consul.timeout", "10s")

    // 可观测性默认配置
    v.SetDefault("observability.tracing.enabled", true)
    v.SetDefault("observability.tracing.sample_ratio", 0.1)
    v.SetDefault("observability.metrics.enabled", true)
    v.SetDefault("observability.logging.level", "info")
    v.SetDefault("observability.logging.format", "json")
    v.SetDefault("observability.health_check.enabled", true)
    v.SetDefault("observability.health_check.endpoint", "/health")

    // 治理默认配置
    v.SetDefault("governance.circuit_breaker.enabled", true)
    v.SetDefault("governance.circuit_breaker.failure_threshold", 0.6)
    v.SetDefault("governance.circuit_breaker.timeout", "30s")
    v.SetDefault("governance.rate_limit.enabled", true)
    v.SetDefault("governance.rate_limit.requests_per_second", 100)
    v.SetDefault("governance.timeout.default", "30s")

    // 安全默认配置
    v.SetDefault("security.jwt.expiry", "24h")
    v.SetDefault("security.cors.allow_origins", []string{"*"})
    v.SetDefault("security.cors.allow_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
}

// 验证配置
func (m *Manager) validateConfig(config *Config) error {
    // 验证服务配置
    if config.Service.Name == "" {
        return fmt.Errorf("service name is required")
    }

    if config.Service.Version == "" {
        return fmt.Errorf("service version is required")
    }

    // 验证服务器配置
    if config.Server.Port <= 0 || config.Server.Port > 65535 {
        return fmt.Errorf("invalid server port: %d", config.Server.Port)
    }

    // 验证数据库配置
    if config.Database.Driver != "" {
        if config.Database.Host == "" {
            return fmt.Errorf("database host is required")
        }
        if config.Database.Port <= 0 {
            return fmt.Errorf("invalid database port: %d", config.Database.Port)
        }
    }

    // 验证JWT配置
    if config.Security.JWT.Secret == "" && config.Service.Environment == "production" {
        return fmt.Errorf("JWT secret is required in production")
    }

    return nil
}

// 处理敏感数据
func (m *Manager) processSensitiveData(config *Config) error {
    // 从环境变量或密钥管理系统加载敏感信息
    if config.Database.Password == "" {
        config.Database.Password = os.Getenv("DB_PASSWORD")
    }

    if config.Redis.Password == "" {
        config.Redis.Password = os.Getenv("REDIS_PASSWORD")
    }

    if config.Security.JWT.Secret == "" {
        config.Security.JWT.Secret = os.Getenv("JWT_SECRET")
    }

    return nil
}

// 合并配置
func (m *Manager) mergeConfigs(local, remote *Config) *Config {
    // 这里实现配置合并逻辑
    // 优先级：远程配置 > 本地配置

    merged := *local

    // 合并特定字段
    if remote.Database.Host != "" {
        merged.Database.Host = remote.Database.Host
    }

    if remote.Redis.Host != "" {
        merged.Redis.Host = remote.Redis.Host
    }

    // 合并自定义配置
    if remote.Custom != nil {
        if merged.Custom == nil {
            merged.Custom = make(map[string]interface{})
        }
        for key, value := range remote.Custom {
            merged.Custom[key] = value
        }
    }

    return &merged
}
```

## 远程配置中心

### Consul配置提供者

```go
package consul

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/hashicorp/consul/api"
    "github.com/your-org/basic/pkg/chi/config"
    "go.uber.org/zap"
)

// Consul配置提供者
type ConfigProvider struct {
    client     *api.Client
    keyPrefix  string
    logger     *zap.Logger
    watchIndex uint64
}

func NewConsulConfigProvider(consulAddr, keyPrefix string, logger *zap.Logger) (*ConfigProvider, error) {
    config := api.DefaultConfig()
    config.Address = consulAddr

    client, err := api.NewClient(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create consul client: %w", err)
    }

    return &ConfigProvider{
        client:    client,
        keyPrefix: keyPrefix,
        logger:    logger,
    }, nil
}

func (cp *ConfigProvider) Load(ctx context.Context) (*config.Config, error) {
    kv := cp.client.KV()

    // 获取配置数据
    pair, _, err := kv.Get(cp.keyPrefix, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to get config from consul: %w", err)
    }

    if pair == nil {
        return nil, fmt.Errorf("config not found in consul: %s", cp.keyPrefix)
    }

    var cfg config.Config
    if err := json.Unmarshal(pair.Value, &cfg); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }

    cp.logger.Info("Config loaded from Consul", zap.String("key", cp.keyPrefix))
    return &cfg, nil
}

func (cp *ConfigProvider) Watch(ctx context.Context, callback func(*config.Config)) error {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            default:
                cp.watchForChanges(ctx, callback)
            }
        }
    }()

    return nil
}

func (cp *ConfigProvider) watchForChanges(ctx context.Context, callback func(*config.Config)) {
    kv := cp.client.KV()

    queryOpts := &api.QueryOptions{
        WaitIndex: cp.watchIndex,
        WaitTime:  5 * time.Minute,
    }

    pair, meta, err := kv.Get(cp.keyPrefix, queryOpts.WithContext(ctx))
    if err != nil {
        cp.logger.Error("Failed to watch config changes", zap.Error(err))
        time.Sleep(5 * time.Second)
        return
    }

    if meta.LastIndex <= cp.watchIndex {
        return
    }

    cp.watchIndex = meta.LastIndex

    if pair != nil {
        var cfg config.Config
        if err := json.Unmarshal(pair.Value, &cfg); err != nil {
            cp.logger.Error("Failed to unmarshal updated config", zap.Error(err))
            return
        }

        cp.logger.Info("Config updated from Consul", zap.String("key", cp.keyPrefix))
        callback(&cfg)
    }
}

func (cp *ConfigProvider) Set(ctx context.Context, config *config.Config) error {
    data, err := json.Marshal(config)
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }

    kv := cp.client.KV()
    pair := &api.KVPair{
        Key:   cp.keyPrefix,
        Value: data,
    }

    _, err = kv.Put(pair, nil)
    if err != nil {
        return fmt.Errorf("failed to put config to consul: %w", err)
    }

    cp.logger.Info("Config saved to Consul", zap.String("key", cp.keyPrefix))
    return nil
}

func (cp *ConfigProvider) Close() error {
    // Consul client doesn't need explicit closing
    return nil
}
```

### Etcd配置提供者

```go
package etcd

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "go.etcd.io/etcd/clientv3"
    "github.com/your-org/basic/pkg/chi/config"
    "go.uber.org/zap"
)

// Etcd配置提供者
type ConfigProvider struct {
    client   *clientv3.Client
    key      string
    logger   *zap.Logger
}

func NewEtcdConfigProvider(endpoints []string, key string, logger *zap.Logger) (*ConfigProvider, error) {
    client, err := clientv3.New(clientv3.Config{
        Endpoints:   endpoints,
        DialTimeout: 5 * time.Second,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create etcd client: %w", err)
    }

    return &ConfigProvider{
        client: client,
        key:    key,
        logger: logger,
    }, nil
}

func (ep *ConfigProvider) Load(ctx context.Context) (*config.Config, error) {
    resp, err := ep.client.Get(ctx, ep.key)
    if err != nil {
        return nil, fmt.Errorf("failed to get config from etcd: %w", err)
    }

    if len(resp.Kvs) == 0 {
        return nil, fmt.Errorf("config not found in etcd: %s", ep.key)
    }

    var cfg config.Config
    if err := json.Unmarshal(resp.Kvs[0].Value, &cfg); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }

    ep.logger.Info("Config loaded from Etcd", zap.String("key", ep.key))
    return &cfg, nil
}

func (ep *ConfigProvider) Watch(ctx context.Context, callback func(*config.Config)) error {
    go func() {
        watchChan := ep.client.Watch(ctx, ep.key)
        for watchResp := range watchChan {
            for _, event := range watchResp.Events {
                if event.Type == clientv3.EventTypePut {
                    var cfg config.Config
                    if err := json.Unmarshal(event.Kv.Value, &cfg); err != nil {
                        ep.logger.Error("Failed to unmarshal updated config", zap.Error(err))
                        continue
                    }

                    ep.logger.Info("Config updated from Etcd", zap.String("key", ep.key))
                    callback(&cfg)
                }
            }
        }
    }()

    return nil
}

func (ep *ConfigProvider) Set(ctx context.Context, config *config.Config) error {
    data, err := json.Marshal(config)
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }

    _, err = ep.client.Put(ctx, ep.key, string(data))
    if err != nil {
        return fmt.Errorf("failed to put config to etcd: %w", err)
    }

    ep.logger.Info("Config saved to Etcd", zap.String("key", ep.key))
    return nil
}

func (ep *ConfigProvider) Close() error {
    return ep.client.Close()
}
```

## 配置使用示例

### 基础使用

```go
package main

import (
    "context"
    "log"

    "github.com/your-org/basic/pkg/chi/config"
    "github.com/your-org/basic/pkg/chi/config/consul"
    "go.uber.org/zap"
)

// 配置监听器示例
type ServiceConfigWatcher struct {
    logger *zap.Logger
}

func (scw *ServiceConfigWatcher) OnConfigChange(oldConfig, newConfig *config.Config) error {
    scw.logger.Info("Service config changed",
        zap.String("old_env", oldConfig.Service.Environment),
        zap.String("new_env", newConfig.Service.Environment),
    )

    // 这里可以实现配置变更的处理逻辑
    // 比如重新初始化数据库连接、更新缓存配置等

    return nil
}

func main() {
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    // 创建配置管理器
    configManager := config.NewManager(logger)

    // 设置远程配置提供者（可选）
    consulProvider, err := consul.NewConsulConfigProvider(
        "localhost:8500",
        "chi/services/my-service/config",
        logger,
    )
    if err != nil {
        log.Printf("Failed to create consul provider: %v", err)
    } else {
        configManager.SetRemoteProvider(consulProvider)
    }

    // 添加配置监听器
    configManager.AddWatcher(&ServiceConfigWatcher{logger: logger})

    // 加载配置
    cfg, err := configManager.Load("config.yaml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // 开始监听配置变化
    if err := configManager.Watch(); err != nil {
        log.Fatalf("Failed to watch config: %v", err)
    }

    // 使用配置
    log.Printf("Service: %s v%s", cfg.Service.Name, cfg.Service.Version)
    log.Printf("Server: %s:%d", cfg.Server.Host, cfg.Server.Port)
    log.Printf("Database: %s@%s:%d/%s",
               cfg.Database.Username,
               cfg.Database.Host,
               cfg.Database.Port,
               cfg.Database.Database)

    // 应用运行逻辑...

    // 程序退出时停止配置管理器
    defer configManager.Stop()
}
```

### 完整配置文件示例

```yaml
# config.yaml
service:
  name: "user-service"
  version: "1.2.0"
  environment: "production"
  description: "User management microservice"
  tags:
    - "user"
    - "authentication"
    - "microservice"
  metadata:
    team: "backend"
    owner: "john@company.com"

server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"
  max_conns: 10000

database:
  driver: "postgres"
  host: "db.company.com"
  port: 5432
  username: "user_service"
  password: "${DB_PASSWORD}"
  database: "user_db"
  sslmode: "require"
  max_open_conns: 100
  max_idle_conns: 20
  conn_max_lifetime: "1h"
  conn_max_idle_time: "10m"

redis:
  host: "redis.company.com"
  port: 6379
  password: "${REDIS_PASSWORD}"
  db: 0
  pool_size: 20
  min_idle_conns: 5
  max_conn_age: "1h"
  pool_timeout: "4s"
  idle_timeout: "5m"

service_discovery:
  type: "consul"
  consul:
    address: "consul.company.com:8500"
    token: "${CONSUL_TOKEN}"
    datacenter: "dc1"
    timeout: "10s"

events:
  type: "kafka"
  buffer_size: 10000
  workers: 10
  kafka:
    brokers:
      - "kafka1.company.com:9092"
      - "kafka2.company.com:9092"
      - "kafka3.company.com:9092"
    consumer_group: "user-service-group"
    version: "2.8.0"

messaging:
  type: "rabbitmq"
  rabbitmq:
    url: "amqp://user:${RABBITMQ_PASSWORD}@rabbitmq.company.com:5672/"

governance:
  circuit_breaker:
    enabled: true
    failure_threshold: 0.6
    timeout: "30s"
    max_requests: 10
    success_threshold: 5

  rate_limit:
    enabled: true
    requests_per_second: 100
    burst_size: 200
    strategies:
      - type: "ip"
        limit: 1000
        window: "1h"
      - type: "user"
        limit: 10000
        window: "1h"

  timeout:
    default: "30s"
    operations:
      database: "10s"
      cache: "2s"
      external_api: "15s"

  retry:
    enabled: true
    max_attempts: 3
    base_delay: "100ms"
    max_delay: "10s"
    multiplier: 2.0
    jitter: true

  bulkhead:
    enabled: true
    pools:
      database:
        max_active: 50
        max_idle: 10
        wait: true
        wait_timeout: "30s"
      cache:
        max_active: 20
        max_idle: 5
        wait: false

observability:
  tracing:
    enabled: true
    service_name: "user-service"
    service_version: "1.2.0"
    environment: "production"
    jaeger_endpoint: "http://jaeger.company.com:14268/api/traces"
    sample_ratio: 0.1
    batch_timeout: "5s"
    export_timeout: "30s"

  metrics:
    enabled: true
    namespace: "chi"
    subsystem: "user_service"
    prometheus:
      listen_address: ":9090"
      metrics_path: "/metrics"

  logging:
    level: "info"
    format: "json"
    outputs: ["stdout", "file"]
    rotation:
      max_size: 100
      max_age: 30
      max_backups: 10
      compress: true
    sampling:
      initial: 100
      thereafter: 100
    fields:
      service: "user-service"
      version: "1.2.0"
      environment: "production"

  health_check:
    enabled: true
    endpoint: "/health"
    check_interval: "30s"
    timeout: "5s"
    failure_threshold: 3
    success_threshold: 1

security:
  jwt:
    secret: "${JWT_SECRET}"
    expiry: "24h"
    refresh_ttl: "168h"
    issuer: "chi-auth"
    audience: "chi-services"

  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/service.crt"
    key_file: "/etc/ssl/private/service.key"
    ca_file: "/etc/ssl/certs/ca.crt"

  cors:
    allow_origins:
      - "https://app.company.com"
      - "https://admin.company.com"
    allow_methods:
      - "GET"
      - "POST"
      - "PUT"
      - "DELETE"
      - "OPTIONS"
    allow_headers:
      - "Content-Type"
      - "Authorization"
      - "X-Requested-With"
    expose_headers:
      - "X-Total-Count"
      - "X-Page-Count"
    allow_credentials: true
    max_age: 3600

  encryption:
    algorithm: "AES-256-GCM"
    key: "${ENCRYPTION_KEY}"

custom:
  business_rules:
    max_login_attempts: 5
    password_expiry_days: 90
    session_timeout: "30m"

  integrations:
    payment_gateway:
      endpoint: "https://payment.company.com/api"
      api_key: "${PAYMENT_API_KEY}"
      timeout: "30s"

    notification_service:
      endpoint: "https://notification.company.com/api"
      api_key: "${NOTIFICATION_API_KEY}"
      timeout: "10s"

  feature_flags:
    enable_new_auth_flow: true
    enable_advanced_logging: false
    maintenance_mode: false
```

通过这个完整的配置管理系统，Chi框架能够提供：

1. **多层级配置** - 支持文件、环境变量、远程配置中心
2. **动态配置** - 实时监听配置变化并自动更新
3. **配置验证** - 确保配置的正确性和完整性
4. **敏感信息保护** - 安全地处理密码、密钥等敏感数据
5. **环境隔离** - 支持开发、测试、生产等多环境配置

这为微服务的配置管理提供了企业级的解决方案。

<function_calls>
<invoke name="TodoWrite">
<parameter name="todos">[{"content": "\u521b\u5eba\u5fae\u670d\u52a1\u67b6\u6784\u6587\u6863", "status": "completed", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u67b6\u6784\u6587\u6863"}, {"content": "\u521b\u5eba\u5fae\u670d\u52a1\u901a\u4fe1\u6587\u6863", "status": "completed", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u901a\u4fe1\u6587\u6863"}, {"content": "\u521b\u5eba\u5fae\u670d\u52a1\u914d\u7f6e\u7ba1\u7406\u6587\u6863", "status": "completed", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u914d\u7f6e\u7ba1\u7406\u6587\u6863"}, {"content": "\u521b\u5eba\u5fae\u670d\u52a1\u90e8\u7f72\u6587\u6863", "status": "in_progress", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u90e8\u7f72\u6587\u6863"}, {"content": "\u521b\u5eba\u5fae\u670d\u52a1\u6700\u4f73\u5b9e\u8df5\u6587\u6863", "status": "pending", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u6700\u4f73\u5b9e\u8df5\u6587\u6863"}, {"content": "\u66f4\u65b0\u4e3bREADME\u6dfb\u52a0\u5fae\u670d\u52a1\u7279\u6027", "status": "pending", "activeForm": "\u66f4\u65b0\u4e3bREADME\u6dfb\u52a0\u5fae\u670d\u52a1\u7279\u6027"}]