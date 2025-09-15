# 服务发现与负载均衡

## 概述

Chi 框架提供了完整的服务发现与负载均衡解决方案，支持多种服务注册中心，实现智能的服务路由和负载分发，确保系统的高可用性和扩展性。

## 服务发现架构

### 整体架构图
```
┌──────────────────────────────────────────────────────────────┐
│                    Service Discovery                         │
│                                                              │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐        │
│  │   Consul    │   │    Etcd     │   │    Nacos    │        │
│  │  Registry   │   │  Registry   │   │  Registry   │        │
│  └─────────────┘   └─────────────┘   └─────────────┘        │
│         │                 │                 │                │
│         └─────────────────┼─────────────────┘                │
│                           │                                  │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              Registry Adapter                           │ │
│  └─────────────────────────────────────────────────────────┘ │
│                           │                                  │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │               Service Manager                           │ │
│  └─────────────────────────────────────────────────────────┘ │
│                           │                                  │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐        │
│  │   Service   │   │  Load       │   │  Health     │        │
│  │  Discovery  │   │  Balancer   │   │  Checker    │        │
│  └─────────────┘   └─────────────┘   └─────────────┘        │
└──────────────────────────────────────────────────────────────┘
```

### 核心组件

1. **Registry Adapter**: 统一的注册中心适配器
2. **Service Manager**: 服务生命周期管理
3. **Service Discovery**: 服务发现客户端
4. **Load Balancer**: 负载均衡器
5. **Health Checker**: 健康检查组件

## 服务注册

### 基础服务注册

```go
package main

import (
    "context"
    "time"
    "github.com/your-org/basic/pkg/chi"
    "github.com/your-org/basic/pkg/chi/discovery"
)

func main() {
    // 创建Chi应用
    app := chi.New()

    // 配置服务注册
    serviceConfig := discovery.ServiceConfig{
        Name:    "user-service",
        Version: "v1.0.0",
        Address: "127.0.0.1",
        Port:    8080,
        Tags:    []string{"api", "user", "v1"},
        Meta: map[string]string{
            "env":     "production",
            "region":  "us-east-1",
            "team":    "backend",
        },
        HealthCheck: discovery.HealthCheck{
            HTTP:     "http://127.0.0.1:8080/health",
            Interval: 10 * time.Second,
            Timeout:  3 * time.Second,
        },
    }

    // 注册服务到Consul
    consulRegistry := discovery.NewConsulRegistry(&discovery.ConsulConfig{
        Address: "127.0.0.1:8500",
        Token:   "consul-token",
    })

    serviceManager := discovery.NewServiceManager(consulRegistry)
    err := serviceManager.Register(context.Background(), serviceConfig)
    if err != nil {
        panic(err)
    }

    // 启动应用
    app.Run(":8080")
}
```

### 高级服务注册

```go
// 多注册中心配置
func setupMultiRegistry() {
    app := chi.New()

    // Consul注册中心
    consulRegistry := discovery.NewConsulRegistry(&discovery.ConsulConfig{
        Address: "127.0.0.1:8500",
        Token:   "consul-token",
    })

    // Etcd注册中心
    etcdRegistry := discovery.NewEtcdRegistry(&discovery.EtcdConfig{
        Endpoints: []string{"127.0.0.1:2379"},
        Username:  "etcd-user",
        Password:  "etcd-pass",
    })

    // Nacos注册中心
    nacosRegistry := discovery.NewNacosRegistry(&discovery.NacosConfig{
        ServerConfigs: []discovery.NacosServerConfig{
            {
                IpAddr:      "127.0.0.1",
                Port:        8848,
                ContextPath: "/nacos",
            },
        },
        ClientConfig: discovery.NacosClientConfig{
            NamespaceId: "public",
            Username:    "nacos",
            Password:    "nacos",
        },
    })

    // 创建多注册中心管理器
    multiRegistry := discovery.NewMultiRegistry(
        consulRegistry,
        etcdRegistry,
        nacosRegistry,
    )

    serviceManager := discovery.NewServiceManager(multiRegistry)

    // 同时注册到所有注册中心
    serviceConfig := discovery.ServiceConfig{
        Name: "user-service",
        // ... 其他配置
    }

    serviceManager.Register(context.Background(), serviceConfig)
}
```

### 动态服务注册

```go
// 动态注册服务实例
func dynamicServiceRegistration() {
    serviceManager := chi.GetServiceManager()

    // 监听服务扩缩容事件
    go func() {
        for event := range getScalingEvents() {
            switch event.Type {
            case "scale_up":
                // 注册新实例
                for _, instance := range event.NewInstances {
                    config := discovery.ServiceConfig{
                        Name:    event.ServiceName,
                        Address: instance.Address,
                        Port:    instance.Port,
                        // ... 其他配置
                    }

                    err := serviceManager.Register(context.Background(), config)
                    if err != nil {
                        log.Error("Failed to register new instance:", err)
                    }
                }

            case "scale_down":
                // 注销实例
                for _, instance := range event.RemovedInstances {
                    err := serviceManager.Deregister(
                        context.Background(),
                        event.ServiceName,
                        instance.ID,
                    )
                    if err != nil {
                        log.Error("Failed to deregister instance:", err)
                    }
                }
            }
        }
    }()
}
```

## 服务发现

### 基础服务发现

```go
// 服务发现客户端
func serviceDiscoveryExample() {
    discoveryClient := chi.GetDiscoveryClient()

    // 发现服务实例
    instances, err := discoveryClient.Discover("user-service")
    if err != nil {
        log.Error("Failed to discover service:", err)
        return
    }

    fmt.Printf("Found %d instances of user-service:\n", len(instances))
    for _, instance := range instances {
        fmt.Printf("- %s:%d (Health: %s)\n",
                   instance.Address,
                   instance.Port,
                   instance.HealthStatus)
    }
}

// 带筛选条件的服务发现
func serviceDiscoveryWithFilter() {
    discoveryClient := chi.GetDiscoveryClient()

    // 根据标签筛选服务
    filter := discovery.Filter{
        Tags: []string{"v1", "production"},
        Meta: map[string]string{
            "region": "us-east-1",
        },
        HealthyOnly: true,
    }

    instances, err := discoveryClient.DiscoverWithFilter("user-service", filter)
    if err != nil {
        log.Error("Failed to discover service with filter:", err)
        return
    }

    for _, instance := range instances {
        fmt.Printf("Filtered instance: %s:%d\n", instance.Address, instance.Port)
    }
}
```

### 服务发现缓存

```go
// 配置服务发现缓存
func setupDiscoveryCache() {
    cacheConfig := discovery.CacheConfig{
        TTL:             30 * time.Second,
        RefreshInterval: 10 * time.Second,
        MaxSize:         1000,
        EnableMetrics:   true,
    }

    discoveryClient := discovery.NewDiscoveryClient(
        registry,
        discovery.WithCache(cacheConfig),
    )

    // 预热缓存
    services := []string{"user-service", "order-service", "payment-service"}
    for _, service := range services {
        go discoveryClient.WarmUpCache(service)
    }
}
```

### 服务监听

```go
// 监听服务变更
func watchServiceChanges() {
    discoveryClient := chi.GetDiscoveryClient()

    // 监听用户服务变更
    watcher, err := discoveryClient.Watch("user-service")
    if err != nil {
        log.Error("Failed to create watcher:", err)
        return
    }

    defer watcher.Close()

    for {
        select {
        case event := <-watcher.Events():
            switch event.Type {
            case discovery.EventServiceAdded:
                log.Info("New service instance added:", event.Instance)
                updateLoadBalancer(event.Service, event.Instance)

            case discovery.EventServiceRemoved:
                log.Info("Service instance removed:", event.Instance)
                removeFromLoadBalancer(event.Service, event.Instance)

            case discovery.EventServiceUpdated:
                log.Info("Service instance updated:", event.Instance)
                updateLoadBalancer(event.Service, event.Instance)
            }

        case err := <-watcher.Errors():
            log.Error("Service watcher error:", err)
        }
    }
}
```

## 负载均衡

### 负载均衡策略

```go
// 轮询负载均衡
type RoundRobinBalancer struct {
    instances []discovery.Instance
    current   int64
    mutex     sync.Mutex
}

func (rb *RoundRobinBalancer) Select(instances []discovery.Instance) (*discovery.Instance, error) {
    if len(instances) == 0 {
        return nil, errors.New("no instances available")
    }

    rb.mutex.Lock()
    defer rb.mutex.Unlock()

    rb.instances = instances
    next := atomic.AddInt64(&rb.current, 1)
    index := int(next % int64(len(instances)))

    return &instances[index], nil
}

// 随机负载均衡
type RandomBalancer struct {
    rand *rand.Rand
}

func NewRandomBalancer() *RandomBalancer {
    return &RandomBalancer{
        rand: rand.New(rand.NewSource(time.Now().UnixNano())),
    }
}

func (rb *RandomBalancer) Select(instances []discovery.Instance) (*discovery.Instance, error) {
    if len(instances) == 0 {
        return nil, errors.New("no instances available")
    }

    index := rb.rand.Intn(len(instances))
    return &instances[index], nil
}

// 加权轮询负载均衡
type WeightedRoundRobinBalancer struct {
    instances []WeightedInstance
    mutex     sync.Mutex
}

type WeightedInstance struct {
    Instance      discovery.Instance
    Weight        int
    CurrentWeight int
}

func (wrb *WeightedRoundRobinBalancer) Select(instances []discovery.Instance) (*discovery.Instance, error) {
    if len(instances) == 0 {
        return nil, errors.New("no instances available")
    }

    wrb.mutex.Lock()
    defer wrb.mutex.Unlock()

    // 更新权重实例
    wrb.updateWeightedInstances(instances)

    var selected *WeightedInstance
    totalWeight := 0

    // 计算当前权重并选择实例
    for i := range wrb.instances {
        instance := &wrb.instances[i]
        instance.CurrentWeight += instance.Weight
        totalWeight += instance.Weight

        if selected == nil || instance.CurrentWeight > selected.CurrentWeight {
            selected = instance
        }
    }

    if selected != nil {
        selected.CurrentWeight -= totalWeight
        return &selected.Instance, nil
    }

    return nil, errors.New("failed to select instance")
}
```

### 最少连接负载均衡

```go
type LeastConnectionsBalancer struct {
    connections map[string]int64
    mutex       sync.RWMutex
}

func NewLeastConnectionsBalancer() *LeastConnectionsBalancer {
    return &LeastConnectionsBalancer{
        connections: make(map[string]int64),
    }
}

func (lcb *LeastConnectionsBalancer) Select(instances []discovery.Instance) (*discovery.Instance, error) {
    if len(instances) == 0 {
        return nil, errors.New("no instances available")
    }

    lcb.mutex.RLock()
    defer lcb.mutex.RUnlock()

    var selected *discovery.Instance
    minConnections := int64(math.MaxInt64)

    for i := range instances {
        instance := &instances[i]
        key := fmt.Sprintf("%s:%d", instance.Address, instance.Port)
        connections := lcb.connections[key]

        if connections < minConnections {
            minConnections = connections
            selected = instance
        }
    }

    // 增加连接计数
    if selected != nil {
        key := fmt.Sprintf("%s:%d", selected.Address, selected.Port)
        atomic.AddInt64(&lcb.connections[key], 1)
    }

    return selected, nil
}

func (lcb *LeastConnectionsBalancer) OnConnectionClose(instance *discovery.Instance) {
    lcb.mutex.Lock()
    defer lcb.mutex.Unlock()

    key := fmt.Sprintf("%s:%d", instance.Address, instance.Port)
    if count := lcb.connections[key]; count > 0 {
        atomic.AddInt64(&lcb.connections[key], -1)
    }
}
```

### 一致性哈希负载均衡

```go
type ConsistentHashBalancer struct {
    hashRing *consistent.Consistent
    mutex    sync.RWMutex
}

func NewConsistentHashBalancer() *ConsistentHashBalancer {
    config := consistent.Config{
        PartitionCount:    271,
        ReplicationFactor: 20,
        Load:             1.25,
        Hasher:           hasher{},
    }

    return &ConsistentHashBalancer{
        hashRing: consistent.New(nil, config),
    }
}

func (chb *ConsistentHashBalancer) Select(instances []discovery.Instance, key string) (*discovery.Instance, error) {
    if len(instances) == 0 {
        return nil, errors.New("no instances available")
    }

    chb.mutex.Lock()
    defer chb.mutex.Unlock()

    // 更新hash环
    members := make([]consistent.Member, len(instances))
    for i, instance := range instances {
        members[i] = hashMember{
            instance: instance,
            id:       fmt.Sprintf("%s:%d", instance.Address, instance.Port),
        }
    }

    chb.hashRing.Set(members)

    // 根据key选择实例
    member := chb.hashRing.LocateKey([]byte(key))
    if member != nil {
        if hashMember, ok := member.(hashMember); ok {
            return &hashMember.instance, nil
        }
    }

    return nil, errors.New("failed to locate instance")
}

type hashMember struct {
    instance discovery.Instance
    id       string
}

func (hm hashMember) String() string {
    return hm.id
}
```

## 健康检查

### HTTP健康检查

```go
// HTTP健康检查器
type HTTPHealthChecker struct {
    client  *http.Client
    timeout time.Duration
}

func NewHTTPHealthChecker(timeout time.Duration) *HTTPHealthChecker {
    return &HTTPHealthChecker{
        client: &http.Client{
            Timeout: timeout,
        },
        timeout: timeout,
    }
}

func (hc *HTTPHealthChecker) Check(instance discovery.Instance) discovery.HealthStatus {
    url := fmt.Sprintf("http://%s:%d%s",
                       instance.Address,
                       instance.Port,
                       instance.HealthCheck.HTTP)

    resp, err := hc.client.Get(url)
    if err != nil {
        return discovery.HealthStatusUnhealthy
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        return discovery.HealthStatusHealthy
    }

    return discovery.HealthStatusUnhealthy
}
```

### TCP健康检查

```go
// TCP健康检查器
type TCPHealthChecker struct {
    timeout time.Duration
}

func NewTCPHealthChecker(timeout time.Duration) *TCPHealthChecker {
    return &TCPHealthChecker{timeout: timeout}
}

func (tc *TCPHealthChecker) Check(instance discovery.Instance) discovery.HealthStatus {
    address := fmt.Sprintf("%s:%d", instance.Address, instance.Port)
    conn, err := net.DialTimeout("tcp", address, tc.timeout)
    if err != nil {
        return discovery.HealthStatusUnhealthy
    }
    conn.Close()

    return discovery.HealthStatusHealthy
}
```

### gRPC健康检查

```go
// gRPC健康检查器
type GRPCHealthChecker struct {
    timeout time.Duration
}

func NewGRPCHealthChecker(timeout time.Duration) *GRPCHealthChecker {
    return &GRPCHealthChecker{timeout: timeout}
}

func (gc *GRPCHealthChecker) Check(instance discovery.Instance) discovery.HealthStatus {
    address := fmt.Sprintf("%s:%d", instance.Address, instance.Port)

    ctx, cancel := context.WithTimeout(context.Background(), gc.timeout)
    defer cancel()

    conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure())
    if err != nil {
        return discovery.HealthStatusUnhealthy
    }
    defer conn.Close()

    client := healthpb.NewHealthClient(conn)
    resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{
        Service: instance.Name,
    })

    if err != nil {
        return discovery.HealthStatusUnhealthy
    }

    if resp.Status == healthpb.HealthCheckResponse_SERVING {
        return discovery.HealthStatusHealthy
    }

    return discovery.HealthStatusUnhealthy
}
```

## 服务路由

### 基于版本的路由

```go
// 版本路由器
type VersionRouter struct {
    defaultVersion string
}

func NewVersionRouter(defaultVersion string) *VersionRouter {
    return &VersionRouter{defaultVersion: defaultVersion}
}

func (vr *VersionRouter) Route(serviceName string, request *http.Request) ([]discovery.Instance, error) {
    version := request.Header.Get("X-Service-Version")
    if version == "" {
        version = vr.defaultVersion
    }

    discoveryClient := chi.GetDiscoveryClient()
    filter := discovery.Filter{
        Tags: []string{version},
        HealthyOnly: true,
    }

    return discoveryClient.DiscoverWithFilter(serviceName, filter)
}
```

### 基于地理位置的路由

```go
// 地理位置路由器
type GeoRouter struct {
    regions map[string][]string // region -> availability zones
}

func NewGeoRouter(regions map[string][]string) *GeoRouter {
    return &GeoRouter{regions: regions}
}

func (gr *GeoRouter) Route(serviceName string, clientRegion string) ([]discovery.Instance, error) {
    discoveryClient := chi.GetDiscoveryClient()

    // 优先选择同地区的实例
    filter := discovery.Filter{
        Meta: map[string]string{
            "region": clientRegion,
        },
        HealthyOnly: true,
    }

    instances, err := discoveryClient.DiscoverWithFilter(serviceName, filter)
    if err != nil {
        return nil, err
    }

    // 如果同地区没有健康实例，则选择邻近地区
    if len(instances) == 0 {
        nearbyRegions := gr.getNearbyRegions(clientRegion)
        for _, region := range nearbyRegions {
            filter.Meta["region"] = region
            instances, err = discoveryClient.DiscoverWithFilter(serviceName, filter)
            if err == nil && len(instances) > 0 {
                break
            }
        }
    }

    return instances, nil
}

func (gr *GeoRouter) getNearbyRegions(region string) []string {
    // 实现地理位置邻近算法
    // 这里简化处理，返回所有其他地区
    var nearby []string
    for r := range gr.regions {
        if r != region {
            nearby = append(nearby, r)
        }
    }
    return nearby
}
```

### 智能路由

```go
// 智能路由器，结合多种策略
type SmartRouter struct {
    versionRouter  *VersionRouter
    geoRouter     *GeoRouter
    loadBalancer  discovery.LoadBalancer
}

func NewSmartRouter() *SmartRouter {
    return &SmartRouter{
        versionRouter: NewVersionRouter("v1"),
        geoRouter:    NewGeoRouter(getRegionConfig()),
        loadBalancer: discovery.NewWeightedRoundRobinBalancer(),
    }
}

func (sr *SmartRouter) SelectInstance(serviceName string, request *http.Request) (*discovery.Instance, error) {
    // 1. 基于版本筛选
    instances, err := sr.versionRouter.Route(serviceName, request)
    if err != nil {
        return nil, err
    }

    // 2. 基于地理位置进一步筛选
    clientRegion := sr.getClientRegion(request)
    if clientRegion != "" {
        geoInstances, err := sr.geoRouter.Route(serviceName, clientRegion)
        if err == nil && len(geoInstances) > 0 {
            instances = geoInstances
        }
    }

    // 3. 使用负载均衡选择最终实例
    return sr.loadBalancer.Select(instances)
}

func (sr *SmartRouter) getClientRegion(request *http.Request) string {
    // 从请求头或IP地址推断客户端地区
    if region := request.Header.Get("X-Client-Region"); region != "" {
        return region
    }

    // 可以根据IP地址进行地理位置解析
    clientIP := sr.getClientIP(request)
    return sr.resolveRegionByIP(clientIP)
}
```

## 配置示例

### 完整配置文件

```yaml
# chi-config.yaml
service_discovery:
  # 服务注册配置
  registration:
    name: "user-service"
    version: "v1.0.0"
    address: "0.0.0.0"
    port: 8080
    tags:
      - "api"
      - "user"
      - "v1"
    meta:
      env: "production"
      region: "us-east-1"
      datacenter: "dc1"

    # 健康检查配置
    health_check:
      http: "/health"
      interval: "10s"
      timeout: "3s"
      deregister_critical_service_after: "30s"

  # 注册中心配置
  registries:
    # Consul配置
    consul:
      enabled: true
      address: "127.0.0.1:8500"
      token: ""
      datacenter: ""

    # Etcd配置
    etcd:
      enabled: false
      endpoints:
        - "127.0.0.1:2379"
      username: ""
      password: ""

    # Nacos配置
    nacos:
      enabled: false
      server_configs:
        - ip: "127.0.0.1"
          port: 8848
          context_path: "/nacos"
      client_config:
        namespace_id: "public"
        username: "nacos"
        password: "nacos"

  # 服务发现配置
  discovery:
    # 缓存配置
    cache:
      ttl: "30s"
      refresh_interval: "10s"
      max_size: 1000
      enable_metrics: true

    # 重试配置
    retry:
      max_attempts: 3
      initial_interval: "1s"
      max_interval: "10s"
      multiplier: 2.0

  # 负载均衡配置
  load_balancer:
    strategy: "weighted_round_robin" # round_robin, random, least_connections, consistent_hash
    health_check_enabled: true

    # 一致性哈希配置(当strategy为consistent_hash时)
    consistent_hash:
      partition_count: 271
      replication_factor: 20
      load: 1.25

  # 路由配置
  routing:
    enable_version_routing: true
    enable_geo_routing: true
    default_version: "v1"

    # 地理位置配置
    regions:
      us-east-1:
        - "us-east-1a"
        - "us-east-1b"
        - "us-east-1c"
      us-west-2:
        - "us-west-2a"
        - "us-west-2b"
```

## 监控与指标

### 指标收集

```go
// 服务发现指标收集
func setupDiscoveryMetrics() {
    discoveryClient := chi.GetDiscoveryClient()

    // 注册Prometheus指标
    serviceDiscoveryRequests := prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "service_discovery_requests_total",
            Help: "Total number of service discovery requests",
        },
        []string{"service", "status"},
    )

    serviceDiscoveryLatency := prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "service_discovery_request_duration_seconds",
            Help: "Service discovery request latency",
        },
        []string{"service"},
    )

    loadBalancerSelections := prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "load_balancer_selections_total",
            Help: "Total number of load balancer selections",
        },
        []string{"service", "instance", "strategy"},
    )

    prometheus.MustRegister(
        serviceDiscoveryRequests,
        serviceDiscoveryLatency,
        loadBalancerSelections,
    )

    // 收集指标
    discoveryClient.OnRequest(func(service string, duration time.Duration, err error) {
        status := "success"
        if err != nil {
            status = "error"
        }

        serviceDiscoveryRequests.WithLabelValues(service, status).Inc()
        serviceDiscoveryLatency.WithLabelValues(service).Observe(duration.Seconds())
    })
}
```

通过以上完整的服务发现与负载均衡方案，Chi框架能够提供企业级的微服务治理能力，确保系统的高可用性、扩展性和性能。