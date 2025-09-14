# 高性能抗逆向加密包

一个高性能、抗逆向工程的Go加密库，提供加密、签名、密钥管理和数据混淆功能。

## 特性

- **高性能加密**: 支持ChaCha20-Poly1305、AES-GCM、XChaCha20-Poly1305算法
- **抗逆向签名**: Ed25519签名，内置多层混淆机制
- **智能密钥管理**: 自动轮换、安全存储、多用途密钥支持
- **数据混淆**: 四层变换保护，增强反编译难度
- **安全套件**: 一体化接口，简化复杂操作
- **性能优化**: 缓冲池、预初始化cipher、硬件加速支持

## 快速开始

### 基本安装

```go
go get github.com/tokmz/basic/pkg/crypto
```

### 简单加密解密

```go
package main

import (
    "fmt"
    "log"

    "github.com/tokmz/basic/pkg/crypto"
)

func main() {
    // 创建安全加密套件
    config := crypto.DefaultConfig()
    suite, err := crypto.NewSecureCryptoSuite(config)
    if err != nil {
        log.Fatal(err)
    }
    defer suite.Shutdown()

    // 加密数据
    plaintext := []byte("这是需要保护的重要数据")
    encrypted, err := suite.Encrypt(plaintext)
    if err != nil {
        log.Fatal(err)
    }

    // 解密数据
    decrypted, err := suite.Decrypt(encrypted)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("原文: %s\n", plaintext)
    fmt.Printf("解密: %s\n", decrypted)
}
```

### 数字签名

```go
// 签名数据
data := []byte("需要签名的文档内容")
signOptions := &crypto.SignOptions{
    AntiReplay: true,
    TTL:        time.Hour,
    Context:    []byte("document_v1"),
}

signature, err := suite.Sign(data, signOptions)
if err != nil {
    log.Fatal(err)
}

// 验证签名
verifyOptions := &crypto.VerifyOptions{
    AntiReplay: true,
    MaxAge:     time.Hour * 2,
    Context:    []byte("document_v1"),
}

err = suite.Verify(data, signature, verifyOptions)
if err != nil {
    log.Printf("签名验证失败: %v", err)
} else {
    fmt.Println("签名验证成功")
}
```

### 密钥派生

```go
// 从密码派生加密密钥
password := []byte("用户密码123")
salt, err := suite.GenerateSalt()
if err != nil {
    log.Fatal(err)
}

derivedKey, err := suite.DeriveKey(password, salt, 32)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("派生密钥长度: %d\n", len(derivedKey))
```

## 高级用法

### 自定义配置

```go
config := &crypto.Config{
    Algorithm:         crypto.AlgorithmChaCha20Poly1305,
    SignAlgorithm:     crypto.SignatureEd25519,
    HashAlgorithm:     crypto.HashArgon2id,
    EnableObfuscation: true,
    EnableHardware:    true,
    EnableKeystore:    true,
    KeyStorePath:      "./keys",
    KeyRotationPeriod: 24 * time.Hour,
    MaxSignatureAge:   time.Hour,
}

suite, err := crypto.NewSecureCryptoSuite(config)
```

### 独立组件使用

#### 高性能加密器

```go
masterKey := make([]byte, 32)
rand.Read(masterKey)

encryptor, err := crypto.NewHighPerformanceEncryptor(
    crypto.AlgorithmChaCha20Poly1305,
    masterKey,
    crypto.DefaultConfig(),
)
if err != nil {
    log.Fatal(err)
}
defer encryptor.Cleanup()

encrypted, err := encryptor.Encrypt([]byte("敏感数据"))
decrypted, err := encryptor.Decrypt(encrypted)
```

#### 抗逆向签名器

```go
keyPair, err := crypto.GenerateKeyPair(crypto.SignatureEd25519)
if err != nil {
    log.Fatal(err)
}

signer, err := crypto.NewAntiReverseSigner(
    crypto.SignatureEd25519,
    keyPair.PrivateKey,
    crypto.DefaultConfig(),
)
if err != nil {
    log.Fatal(err)
}
defer signer.Cleanup()

signature, err := signer.Sign(data, &crypto.SignOptions{
    AntiReplay: true,
})
```

#### 密钥管理器

```go
config := crypto.DefaultConfig()
config.EnableKeystore = true
config.KeyStorePath = "./secure_keys"

keyManager, err := crypto.NewKeyManager(config)
if err != nil {
    log.Fatal(err)
}
defer keyManager.Stop()

// 创建加密密钥
encKey, err := keyManager.CreateKey(
    "app-encryption-key",
    "symmetric",
    crypto.KeyUsageEncryption,
    crypto.WithExpiration(time.Now().Add(30*24*time.Hour)),
)

// 轮换密钥
newKey, err := keyManager.RotateKey(encKey.ID)
```

#### 数据混淆器

```go
masterKey := make([]byte, 32)
rand.Read(masterKey)

obfuscator := crypto.NewDataObfuscator(masterKey, crypto.DefaultConfig())
defer obfuscator.Cleanup()

obfuscated, err := obfuscator.Obfuscate([]byte("源代码片段"))
original, err := obfuscator.Deobfuscate(obfuscated)

// 测试混淆效果
results := obfuscator.TestObfuscation([]byte("测试数据"))
fmt.Printf("混淆效果: %+v\n", results)
```

## 安全特性

### 抗逆向工程

1. **多层数据混淆**：四层变换算法保护数据结构
2. **签名载荷混淆**：防止签名算法逆向分析
3. **时间戳保护**：防重放攻击，增加逆向难度
4. **随机化nonce**：每次操作使用不同随机数

### 防重放攻击

```go
signOptions := &crypto.SignOptions{
    AntiReplay: true,
    TTL:        time.Minute * 30,
    Context:    []byte("api_v2.1"),
}

verifyOptions := &crypto.VerifyOptions{
    AntiReplay: true,
    MaxAge:     time.Hour,
    Context:    []byte("api_v2.1"),
}
```

### 内存安全

- 敏感数据自动清零
- 安全随机数生成
- 常数时间比较函数
- 缓冲池内存重用

## 性能优化

### 缓冲池

内置缓冲池减少GC压力，提高高频操作性能。

### 预初始化Cipher

cipher实例预初始化，减少加密操作延迟。

### 硬件加速

启用硬件AES指令集支持（需要硬件支持）。

## 错误处理

```go
encrypted, err := suite.Encrypt(data)
if err != nil {
    if cryptoErr, ok := err.(*crypto.CryptoError); ok {
        fmt.Printf("操作: %s, 算法: %s, 原因: %v\n",
            cryptoErr.Operation,
            cryptoErr.Algorithm,
            cryptoErr.Cause)

        for key, value := range cryptoErr.Details {
            fmt.Printf("详情 %s: %v\n", key, value)
        }
    }
}
```

## 配置选项

| 选项 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| Algorithm | CryptoAlgorithm | ChaCha20Poly1305 | 加密算法 |
| SignAlgorithm | SignatureAlgorithm | Ed25519 | 签名算法 |
| HashAlgorithm | HashAlgorithm | Blake2b | 哈希算法 |
| EnableObfuscation | bool | true | 启用数据混淆 |
| EnableHardware | bool | true | 启用硬件加速 |
| EnableKeystore | bool | false | 启用密钥存储 |
| KeyStorePath | string | "" | 密钥存储路径 |
| KeyRotationPeriod | time.Duration | 0 | 密钥轮换周期 |
| MaxSignatureAge | time.Duration | 10分钟 | 签名最大有效期 |

## 最佳实践

1. **密钥管理**：使用密钥轮换，定期更新密钥
2. **错误处理**：妥善处理加密错误，不要泄露敏感信息
3. **配置安全**：生产环境启用所有安全特性
4. **存储安全**：密钥存储路径使用安全权限
5. **上下文隔离**：不同应用使用不同Context
6. **及时清理**：使用defer确保资源清理

## 注意事项

- 该库专为高安全要求场景设计
- 密钥轮换会影响旧数据解密，请妥善处理
- 混淆功能会增加一定性能开销
- 建议在生产环境进行充分测试

## 许可证

MIT License

## API稳定性

当前版本：v1.0
API状态：稳定

### 版本兼容性

- v1.x - 主要API保持向后兼容
- 破坏性更改将在新的主版本中引入
- 建议使用模块版本固定：`go get github.com/tokmz/basic/pkg/crypto@v1.0.0`

### 安全更新

定期关注安全更新通知：
- 密钥轮换策略应定期评估
- 混淆算法可能根据威胁模型更新
- 加密算法选择跟随行业最佳实践

## 常见问题

### Q: 混淆功能会影响性能吗？
A: 是的，混淆会增加约20-30%的计算开销，但能显著提高反编译难度。可通过`EnableObfuscation`配置选项控制。

### Q: 密钥轮换后旧数据怎么解密？
A: 当前实现中，密钥轮换后需要使用对应版本的密钥解密。建议在生产环境中实现密钥版本管理。

### Q: 支持硬件加速吗？
A: 支持，通过`EnableHardware`选项启用AES-NI等硬件指令集加速。

### Q: 如何处理大文件加密？
A: 对于大文件，建议分块处理或使用流式加密接口（未来版本将支持）。