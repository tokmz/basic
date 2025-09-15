# 微服务部署指南

## 概述

本文档详细介绍如何在不同环境中部署基于 Chi 框架的微服务应用，包括容器化部署、Kubernetes集群部署、服务网格集成以及CI/CD流水线配置。

## 部署架构

### 整体部署架构
```
┌─────────────────────────────────────────────────────────────────┐
│                    Production Environment                        │
│                                                                 │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐          │
│  │    CDN      │   │   Load      │   │  API        │          │
│  │             │   │ Balancer    │   │ Gateway     │          │
│  └─────────────┘   └─────────────┘   └─────────────┘          │
│          │                 │                 │                 │
│          └─────────────────┼─────────────────┘                 │
│                            │                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │              Kubernetes Cluster                            ││
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐          ││
│  │  │ User    │ │ Order   │ │Payment  │ │ Notice  │          ││
│  │  │Service  │ │Service  │ │Service  │ │Service  │          ││
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘          ││
│  └─────────────────────────────────────────────────────────────┘│
│                            │                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │              Infrastructure Services                       ││
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐          ││
│  │  │Database │ │Message  │ │ Cache   │ │Monitoring│          ││
│  │  │Cluster  │ │ Queue   │ │ Redis   │ │& Logging │          ││
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘          ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

## 容器化部署

### Docker镜像构建

#### 多阶段构建Dockerfile

```dockerfile
# Build stage
FROM golang:1.19-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要工具
RUN apk add --no-cache git ca-certificates tzdata

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o main ./cmd/service

# Final stage
FROM scratch

# 从builder阶段复制必要文件
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/main /main

# 复制配置文件
COPY --from=builder /app/config /config

# 设置环境变量
ENV TZ=UTC
ENV GIN_MODE=release

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/main", "healthcheck"]

# 运行应用
ENTRYPOINT ["/main"]
```

#### 优化的生产环境Dockerfile

```dockerfile
FROM golang:1.19-alpine AS builder

# 设置代理和工具
ARG GOPROXY=https://proxy.golang.org
RUN apk add --no-cache git ca-certificates tzdata upx

WORKDIR /app

# 缓存依赖层
COPY go.mod go.sum ./
RUN go mod download

# 构建应用
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -tags netgo -installsuffix netgo \
    -o main ./cmd/service

# 压缩二进制文件
RUN upx --best --lzma main

# 最小化最终镜像
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/main /main
COPY --from=builder /app/config /config

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/main"]
```

### Docker Compose部署

#### 开发环境配置

```yaml
# docker-compose.dev.yml
version: '3.8'

services:
  # 用户服务
  user-service:
    build:
      context: ./services/user-service
      dockerfile: Dockerfile.dev
    ports:
      - "8081:8080"
    environment:
      - CHI_ENV=development
      - DB_HOST=postgres
      - REDIS_HOST=redis
      - CONSUL_HOST=consul
    depends_on:
      - postgres
      - redis
      - consul
    volumes:
      - ./services/user-service:/app
      - /app/vendor
    networks:
      - chi-network

  # 订单服务
  order-service:
    build:
      context: ./services/order-service
      dockerfile: Dockerfile.dev
    ports:
      - "8082:8080"
    environment:
      - CHI_ENV=development
      - DB_HOST=postgres
      - REDIS_HOST=redis
      - CONSUL_HOST=consul
    depends_on:
      - postgres
      - redis
      - consul
    networks:
      - chi-network

  # API网关
  api-gateway:
    build:
      context: ./gateway
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - CHI_ENV=development
      - CONSUL_HOST=consul
    depends_on:
      - consul
      - user-service
      - order-service
    networks:
      - chi-network

  # 基础设施服务
  postgres:
    image: postgres:13
    environment:
      POSTGRES_DB: chi_db
      POSTGRES_USER: chi_user
      POSTGRES_PASSWORD: chi_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./scripts/init-db.sql:/docker-entrypoint-initdb.d/init.sql
    ports:
      - "5432:5432"
    networks:
      - chi-network

  redis:
    image: redis:6-alpine
    command: redis-server --requirepass redis_password
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    networks:
      - chi-network

  consul:
    image: consul:1.15
    command: consul agent -dev -client=0.0.0.0 -ui
    ports:
      - "8500:8500"
    environment:
      - CONSUL_BIND_INTERFACE=eth0
    volumes:
      - consul_data:/consul/data
    networks:
      - chi-network

  # 监控服务
  jaeger:
    image: jaegertracing/all-in-one:1.35
    ports:
      - "16686:16686"
      - "14268:14268"
    environment:
      - COLLECTOR_OTLP_ENABLED=true
    networks:
      - chi-network

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
    networks:
      - chi-network

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana_data:/var/lib/grafana
      - ./monitoring/grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./monitoring/grafana/datasources:/etc/grafana/provisioning/datasources
    networks:
      - chi-network

volumes:
  postgres_data:
  redis_data:
  consul_data:
  prometheus_data:
  grafana_data:

networks:
  chi-network:
    driver: bridge
```

#### 生产环境配置

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  user-service:
    image: chi/user-service:${VERSION}
    deploy:
      replicas: 3
      update_config:
        parallelism: 1
        delay: 10s
        order: start-first
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 256M
    environment:
      - CHI_ENV=production
      - DB_HOST=postgres-cluster
      - REDIS_HOST=redis-cluster
      - CONSUL_HOST=consul-cluster
    secrets:
      - db_password
      - redis_password
      - jwt_secret
    healthcheck:
      test: ["/main", "healthcheck"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    networks:
      - chi-network
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  order-service:
    image: chi/order-service:${VERSION}
    deploy:
      replicas: 3
      update_config:
        parallelism: 1
        delay: 10s
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
    environment:
      - CHI_ENV=production
    secrets:
      - db_password
      - redis_password
    networks:
      - chi-network

  api-gateway:
    image: chi/api-gateway:${VERSION}
    ports:
      - "80:8080"
      - "443:8443"
    deploy:
      replicas: 2
      update_config:
        parallelism: 1
        delay: 5s
      placement:
        constraints:
          - node.role == manager
    environment:
      - CHI_ENV=production
    secrets:
      - tls_cert
      - tls_key
    networks:
      - chi-network

secrets:
  db_password:
    external: true
  redis_password:
    external: true
  jwt_secret:
    external: true
  tls_cert:
    external: true
  tls_key:
    external: true

networks:
  chi-network:
    external: true
```

## Kubernetes部署

### 基础配置

#### Namespace配置

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: chi-system
  labels:
    name: chi-system
    istio-injection: enabled
```

#### ConfigMap配置

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: chi-config
  namespace: chi-system
data:
  config.yaml: |
    service:
      environment: "production"
    server:
      port: 8080
      mode: "release"
    database:
      host: "postgres-service"
      port: 5432
      max_open_conns: 50
    redis:
      host: "redis-service"
      port: 6379
    service_discovery:
      type: "kubernetes"
    observability:
      tracing:
        enabled: true
        jaeger_endpoint: "http://jaeger-collector:14268/api/traces"
      metrics:
        enabled: true
      logging:
        level: "info"
        format: "json"
```

#### Secret配置

```yaml
# secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: chi-secrets
  namespace: chi-system
type: Opaque
data:
  db-password: <base64-encoded-password>
  redis-password: <base64-encoded-password>
  jwt-secret: <base64-encoded-secret>
  tls-cert: <base64-encoded-cert>
  tls-key: <base64-encoded-key>
```

### 服务部署

#### 用户服务部署

```yaml
# user-service.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
  namespace: chi-system
  labels:
    app: user-service
    version: v1
spec:
  replicas: 3
  selector:
    matchLabels:
      app: user-service
      version: v1
  template:
    metadata:
      labels:
        app: user-service
        version: v1
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: chi-service-account
      containers:
      - name: user-service
        image: chi/user-service:1.0.0
        imagePullPolicy: Always
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: CHI_ENV
          value: "production"
        - name: SERVICE_NAME
          value: "user-service"
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: chi-secrets
              key: db-password
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: chi-secrets
              key: redis-password
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: chi-secrets
              key: jwt-secret
        volumeMounts:
        - name: config
          mountPath: /config
          readOnly: true
        - name: tls-certs
          mountPath: /etc/ssl/certs
          readOnly: true
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
        lifecycle:
          preStop:
            exec:
              command: ["/bin/sh", "-c", "sleep 15"]
      volumes:
      - name: config
        configMap:
          name: chi-config
      - name: tls-certs
        secret:
          secretName: chi-secrets
          items:
          - key: tls-cert
            path: tls.crt
          - key: tls-key
            path: tls.key
      terminationGracePeriodSeconds: 30

---
apiVersion: v1
kind: Service
metadata:
  name: user-service
  namespace: chi-system
  labels:
    app: user-service
spec:
  selector:
    app: user-service
  ports:
  - name: http
    port: 80
    targetPort: 8080
    protocol: TCP
  - name: metrics
    port: 9090
    targetPort: 9090
    protocol: TCP
  type: ClusterIP

---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: user-service-hpa
  namespace: chi-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: user-service
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 15
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
```

#### API网关部署

```yaml
# api-gateway.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway
  namespace: chi-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app: api-gateway
  template:
    metadata:
      labels:
        app: api-gateway
    spec:
      containers:
      - name: api-gateway
        image: chi/api-gateway:1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: CHI_ENV
          value: "production"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"

---
apiVersion: v1
kind: Service
metadata:
  name: api-gateway
  namespace: chi-system
spec:
  selector:
    app: api-gateway
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer

---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-gateway-ingress
  namespace: chi-system
  annotations:
    kubernetes.io/ingress.class: "nginx"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/rate-limit-window: "1m"
spec:
  tls:
  - hosts:
    - api.yourdomain.com
    secretName: api-gateway-tls
  rules:
  - host: api.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: api-gateway
            port:
              number: 80
```

### 基础设施服务

#### PostgreSQL集群

```yaml
# postgres-cluster.yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: postgres-cluster
  namespace: chi-system
spec:
  instances: 3
  primaryUpdateStrategy: unsupervised

  postgresql:
    parameters:
      max_connections: "200"
      shared_buffers: "256MB"
      effective_cache_size: "1GB"
      wal_buffers: "8MB"
      checkpoint_completion_target: "0.9"
      random_page_cost: "1.1"

  bootstrap:
    initdb:
      database: chi_db
      owner: chi_user
      secret:
        name: postgres-credentials

  storage:
    size: 100Gi
    storageClass: fast-ssd

  monitoring:
    enabled: true

  backup:
    retentionPolicy: "30d"
    barmanObjectStore:
      destinationPath: "s3://chi-backups/postgres"
      s3Credentials:
        accessKeyId:
          name: backup-credentials
          key: ACCESS_KEY_ID
        secretAccessKey:
          name: backup-credentials
          key: SECRET_ACCESS_KEY
      wal:
        retention: "5d"
      data:
        retention: "30d"

---
apiVersion: v1
kind: Secret
metadata:
  name: postgres-credentials
  namespace: chi-system
type: kubernetes.io/basic-auth
data:
  username: <base64-encoded-username>
  password: <base64-encoded-password>
```

#### Redis集群

```yaml
# redis-cluster.yaml
apiVersion: redis.redis.opstreelabs.in/v1beta1
kind: RedisCluster
metadata:
  name: redis-cluster
  namespace: chi-system
spec:
  clusterSize: 6
  kubernetesConfig:
    image: redis:6.2.7
    imagePullPolicy: IfNotPresent
    resources:
      requests:
        cpu: 101m
        memory: 128Mi
      limits:
        cpu: 101m
        memory: 128Mi
  storage:
    volumeClaimTemplate:
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 10Gi
        storageClassName: fast-ssd
  redisExporter:
    enabled: true
    image: oliver006/redis_exporter:latest
  redisConfig:
    additionalRedisConfig: |
      maxmemory 100mb
      maxmemory-policy allkeys-lru
```

## 服务网格集成

### Istio集成

#### Gateway配置

```yaml
# istio-gateway.yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: chi-gateway
  namespace: chi-system
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - api.yourdomain.com
    tls:
      httpsRedirect: true
  - port:
      number: 443
      name: https
      protocol: HTTPS
    tls:
      mode: SIMPLE
      credentialName: api-gateway-tls
    hosts:
    - api.yourdomain.com

---
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: chi-virtualservice
  namespace: chi-system
spec:
  hosts:
  - api.yourdomain.com
  gateways:
  - chi-gateway
  http:
  - match:
    - uri:
        prefix: "/api/users"
    route:
    - destination:
        host: user-service
        port:
          number: 80
      weight: 100
    fault:
      delay:
        percentage:
          value: 0.1
        fixedDelay: 5s
    retries:
      attempts: 3
      perTryTimeout: 10s
  - match:
    - uri:
        prefix: "/api/orders"
    route:
    - destination:
        host: order-service
        port:
          number: 80
      weight: 100
```

#### 流量策略

```yaml
# traffic-policy.yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: user-service-destination
  namespace: chi-system
spec:
  host: user-service
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 10
      http:
        http1MaxPendingRequests: 10
        http2MaxRequests: 100
        maxRequestsPerConnection: 2
        maxRetries: 3
        consecutiveGatewayErrors: 5
        interval: 30s
        baseEjectionTime: 30s
    loadBalancer:
      simple: LEAST_CONN
    circuitBreaker:
      consecutiveGatewayErrors: 5
      interval: 30s
      baseEjectionTime: 30s
      maxEjectionPercent: 50
  subsets:
  - name: v1
    labels:
      version: v1
  - name: v2
    labels:
      version: v2

---
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: chi-system
spec:
  mtls:
    mode: STRICT

---
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: chi-authz
  namespace: chi-system
spec:
  selector:
    matchLabels:
      app: user-service
  rules:
  - from:
    - source:
        principals: ["cluster.local/ns/chi-system/sa/api-gateway"]
  - to:
    - operation:
        methods: ["GET", "POST"]
  - when:
    - key: source.ip
      notValues: ["10.0.0.0/8"]
```

## CI/CD流水线

### GitLab CI配置

```yaml
# .gitlab-ci.yml
stages:
  - test
  - build
  - security
  - deploy-staging
  - deploy-production

variables:
  DOCKER_DRIVER: overlay2
  DOCKER_TLS_CERTDIR: "/certs"
  REGISTRY: registry.gitlab.com
  IMAGE_NAME: $REGISTRY/$CI_PROJECT_PATH

# 测试阶段
test:
  stage: test
  image: golang:1.19-alpine
  services:
    - postgres:13
    - redis:6-alpine
  variables:
    POSTGRES_DB: test_db
    POSTGRES_USER: test_user
    POSTGRES_PASSWORD: test_password
    REDIS_URL: redis://redis:6379
  before_script:
    - apk add --no-cache git
    - go mod download
  script:
    - go test -v -race -coverprofile=coverage.out ./...
    - go tool cover -html=coverage.out -o coverage.html
  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: coverage.xml
    paths:
      - coverage.html
    expire_in: 1 week
  coverage: '/coverage: \d+\.\d+% of statements/'

# 构建阶段
build:
  stage: build
  image: docker:20.10.16
  services:
    - docker:20.10.16-dind
  before_script:
    - echo $CI_REGISTRY_PASSWORD | docker login -u $CI_REGISTRY_USER --password-stdin $CI_REGISTRY
  script:
    - docker build -t $IMAGE_NAME:$CI_COMMIT_SHA .
    - docker build -t $IMAGE_NAME:latest .
    - docker push $IMAGE_NAME:$CI_COMMIT_SHA
    - docker push $IMAGE_NAME:latest
  only:
    - main
    - develop

# 安全扫描
security_scan:
  stage: security
  image: docker:20.10.16
  services:
    - docker:20.10.16-dind
  script:
    - docker pull $IMAGE_NAME:$CI_COMMIT_SHA
    - docker run --rm -v /var/run/docker.sock:/var/run/docker.sock
      aquasec/trivy image --exit-code 1 --severity HIGH,CRITICAL $IMAGE_NAME:$CI_COMMIT_SHA
  allow_failure: false
  only:
    - main
    - develop

# 部署到预发环境
deploy_staging:
  stage: deploy-staging
  image: bitnami/kubectl:latest
  script:
    - kubectl config use-context $STAGING_CONTEXT
    - sed -i "s|IMAGE_TAG|$CI_COMMIT_SHA|g" k8s/staging/*.yaml
    - kubectl apply -f k8s/staging/
    - kubectl rollout status deployment/user-service -n chi-staging
  environment:
    name: staging
    url: https://staging-api.yourdomain.com
  only:
    - develop

# 部署到生产环境
deploy_production:
  stage: deploy-production
  image: bitnami/kubectl:latest
  script:
    - kubectl config use-context $PRODUCTION_CONTEXT
    - sed -i "s|IMAGE_TAG|$CI_COMMIT_SHA|g" k8s/production/*.yaml
    - kubectl apply -f k8s/production/
    - kubectl rollout status deployment/user-service -n chi-system
  environment:
    name: production
    url: https://api.yourdomain.com
  when: manual
  only:
    - main
```

### GitHub Actions配置

```yaml
# .github/workflows/deploy.yml
name: Build and Deploy

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:13
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: test_db
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

      redis:
        image: redis:6
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 6379:6379

    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.19

    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-

    - name: Download dependencies
      run: go mod download

    - name: Run tests
      run: |
        go test -v -race -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage.html

    - name: Upload coverage reports
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out

  build-and-push:
    needs: test
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
    - name: Checkout repository
      uses: actions/checkout@v3

    - name: Log in to Container Registry
      uses: docker/login-action@v2
      with:
        registry: ${{ env.REGISTRY }}
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}

    - name: Extract metadata
      id: meta
      uses: docker/metadata-action@v4
      with:
        images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
        tags: |
          type=ref,event=branch
          type=ref,event=pr
          type=sha,prefix={{branch}}-

    - name: Build and push Docker image
      uses: docker/build-push-action@v4
      with:
        context: .
        push: true
        tags: ${{ steps.meta.outputs.tags }}
        labels: ${{ steps.meta.outputs.labels }}

  security-scan:
    needs: build-and-push
    runs-on: ubuntu-latest
    steps:
    - name: Run Trivy vulnerability scanner
      uses: aquasecurity/trivy-action@master
      with:
        image-ref: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.sha }}
        format: 'sarif'
        output: 'trivy-results.sarif'

    - name: Upload Trivy scan results
      uses: github/codeql-action/upload-sarif@v2
      with:
        sarif_file: 'trivy-results.sarif'

  deploy-staging:
    needs: [build-and-push, security-scan]
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/develop'
    environment: staging

    steps:
    - name: Checkout repository
      uses: actions/checkout@v3

    - name: Configure kubectl
      uses: azure/k8s-set-context@v1
      with:
        method: kubeconfig
        kubeconfig: ${{ secrets.STAGING_KUBECONFIG }}

    - name: Deploy to staging
      run: |
        sed -i "s|IMAGE_TAG|${{ github.sha }}|g" k8s/staging/*.yaml
        kubectl apply -f k8s/staging/
        kubectl rollout status deployment/user-service -n chi-staging

  deploy-production:
    needs: [build-and-push, security-scan]
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    environment: production

    steps:
    - name: Checkout repository
      uses: actions/checkout@v3

    - name: Configure kubectl
      uses: azure/k8s-set-context@v1
      with:
        method: kubeconfig
        kubeconfig: ${{ secrets.PRODUCTION_KUBECONFIG }}

    - name: Deploy to production
      run: |
        sed -i "s|IMAGE_TAG|${{ github.sha }}|g" k8s/production/*.yaml
        kubectl apply -f k8s/production/
        kubectl rollout status deployment/user-service -n chi-system
```

## 监控和日志

### 监控配置

```yaml
# monitoring.yaml
apiVersion: v1
kind: ServiceMonitor
metadata:
  name: chi-services
  namespace: chi-system
spec:
  selector:
    matchLabels:
      monitoring: enabled
  endpoints:
  - port: metrics
    path: /metrics
    interval: 30s

---
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: chi-alerts
  namespace: chi-system
spec:
  groups:
  - name: chi.rules
    rules:
    - alert: ServiceDown
      expr: up{job="chi-services"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Chi service is down"
        description: "Service {{ $labels.instance }} has been down for more than 5 minutes."

    - alert: HighErrorRate
      expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
      for: 2m
      labels:
        severity: warning
      annotations:
        summary: "High error rate detected"
        description: "Error rate is {{ $value }} errors per second."

    - alert: HighLatency
      expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High latency detected"
        description: "95th percentile latency is {{ $value }} seconds."
```

### 日志收集

```yaml
# logging.yaml
apiVersion: logging.coreos.com/v1
kind: ClusterLogForwarder
metadata:
  name: chi-log-forwarder
  namespace: openshift-logging
spec:
  outputs:
  - name: elasticsearch
    type: elasticsearch
    url: https://elasticsearch.logging.svc.cluster.local:9200
    secret:
      name: elasticsearch-secret
  - name: splunk
    type: splunk
    url: https://splunk.company.com:8088
    secret:
      name: splunk-secret

  pipelines:
  - name: chi-application-logs
    inputRefs:
    - application
    filterRefs:
    - chi-filter
    outputRefs:
    - elasticsearch
    - splunk

  filters:
  - name: chi-filter
    type: json
    json:
      javascript: |
        const log = record.log;
        if (log && log.level && log.service) {
          record.structured = log;
        }
        return record;
```

通过以上完整的部署配置，Chi框架的微服务应用可以在生产环境中稳定、高效地运行，同时具备完整的监控、日志和安全保障。

<function_calls>
<invoke name="TodoWrite">
<parameter name="todos">[{"content": "\u521b\u5eba\u5fae\u670d\u52a1\u67b6\u6784\u6587\u6863", "status": "completed", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u67b6\u6784\u6587\u6863"}, {"content": "\u521b\u5eba\u5fae\u670d\u52a1\u901a\u4fe1\u6587\u6863", "status": "completed", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u901a\u4fe1\u6587\u6863"}, {"content": "\u521b\u5eba\u5fae\u670d\u52a1\u914d\u7f6e\u7ba1\u7406\u6587\u6863", "status": "completed", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u914d\u7f6e\u7ba1\u7406\u6587\u6863"}, {"content": "\u521b\u5eba\u5fae\u670d\u52a1\u90e8\u7f72\u6587\u6863", "status": "completed", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u90e8\u7f72\u6587\u6863"}, {"content": "\u521b\u5eba\u5fae\u670d\u52a1\u6700\u4f73\u5b9e\u8df5\u6587\u6863", "status": "in_progress", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u6700\u4f73\u5b9e\u8df5\u6587\u6863"}, {"content": "\u66f4\u65b0\u4e3bREADME\u6dfb\u52a0\u5fae\u670d\u52a1\u7279\u6027", "status": "pending", "activeForm": "\u66f4\u65b0\u4e3bREADME\u6dfb\u52a0\u5fae\u670d\u52a1\u7279\u6027"}]