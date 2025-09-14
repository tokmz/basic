package crypto

import (
	"context"
	"time"
)

// CryptoAlgorithm 加密算法类型
type CryptoAlgorithm string

const (
	AlgorithmChaCha20Poly1305 CryptoAlgorithm = "chacha20poly1305"
	AlgorithmAESGCM           CryptoAlgorithm = "aes-gcm"
	AlgorithmXChaCha20        CryptoAlgorithm = "xchacha20poly1305"
)

// HashAlgorithm 哈希算法类型
type HashAlgorithm string

const (
	HashSHA256   HashAlgorithm = "sha256"
	HashSHA512   HashAlgorithm = "sha512"
	HashBlake2b  HashAlgorithm = "blake2b"
	HashArgon2id HashAlgorithm = "argon2id"
)

// SignatureAlgorithm 签名算法类型
type SignatureAlgorithm string

const (
	SignatureEd25519     SignatureAlgorithm = "ed25519"
	SignatureECDSAP256   SignatureAlgorithm = "ecdsa-p256"
	SignatureRSAPSS      SignatureAlgorithm = "rsa-pss"
	SignatureHybrid      SignatureAlgorithm = "hybrid"
)

// Encryptor 加密器接口
type Encryptor interface {
	// Encrypt 加密数据
	Encrypt(plaintext []byte) (*EncryptedData, error)

	// Decrypt 解密数据
	Decrypt(encrypted *EncryptedData) ([]byte, error)

	// Algorithm 获取加密算法
	Algorithm() CryptoAlgorithm
}

// Signer 签名器接口
type Signer interface {
	// Sign 对数据进行签名
	Sign(data []byte, options *SignOptions) (*Signature, error)

	// Verify 验证签名
	Verify(data []byte, signature *Signature, options *VerifyOptions) error

	// Algorithm 获取签名算法
	Algorithm() SignatureAlgorithm
}

// KeyDeriver 密钥派生器接口
type KeyDeriver interface {
	// DeriveKey 派生密钥
	DeriveKey(password []byte, salt []byte, keyLen int) ([]byte, error)

	// GenerateSalt 生成盐值
	GenerateSalt() ([]byte, error)

	// Algorithm 获取密钥派生算法
	Algorithm() HashAlgorithm
}

// SecureCrypto 安全加密器接口（组合接口）
type SecureCrypto interface {
	// Encrypt 加密数据
	Encrypt(plaintext []byte) (*EncryptedData, error)

	// Decrypt 解密数据
	Decrypt(encrypted *EncryptedData) ([]byte, error)

	// Sign 对数据进行签名
	Sign(data []byte, options *SignOptions) (*Signature, error)

	// Verify 验证签名
	Verify(data []byte, signature *Signature, options *VerifyOptions) error

	// DeriveKey 派生密钥
	DeriveKey(password []byte, salt []byte, keyLen int) ([]byte, error)

	// GenerateSalt 生成盐值
	GenerateSalt() ([]byte, error)

	// GenerateKeyPair 生成密钥对
	GenerateKeyPair() (*KeyPair, error)

	// SetMasterKey 设置主密钥
	SetMasterKey(key []byte) error

	// RotateKeys 轮换密钥
	RotateKeys() error

	// Algorithm 获取加密算法
	Algorithm() CryptoAlgorithm
}

// EncryptedData 加密数据结构
type EncryptedData struct {
	Algorithm  CryptoAlgorithm `json:"algorithm"`
	Nonce      []byte          `json:"nonce"`
	Ciphertext []byte          `json:"ciphertext"`
	Tag        []byte          `json:"tag"`
	Metadata   map[string]any  `json:"metadata,omitempty"`
	Version    uint32          `json:"version"`
	Timestamp  int64           `json:"timestamp"`
}

// Signature 签名数据结构
type Signature struct {
	Algorithm SignatureAlgorithm `json:"algorithm"`
	Value     []byte             `json:"value"`
	PublicKey []byte             `json:"public_key,omitempty"`
	Metadata  map[string]any     `json:"metadata,omitempty"`
	Timestamp int64              `json:"timestamp"`
	Nonce     []byte             `json:"nonce"`
	Version   uint32             `json:"version"`
}

// KeyPair 密钥对
type KeyPair struct {
	PrivateKey []byte            `json:"private_key"`
	PublicKey  []byte            `json:"public_key"`
	Algorithm  SignatureAlgorithm `json:"algorithm"`
	CreatedAt  time.Time         `json:"created_at"`
	Metadata   map[string]any    `json:"metadata,omitempty"`
}

// SignOptions 签名选项
type SignOptions struct {
	Timestamp    *int64         `json:"timestamp,omitempty"`
	Nonce        []byte         `json:"nonce,omitempty"`
	Context      []byte         `json:"context,omitempty"`
	TTL          time.Duration  `json:"ttl,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	AntiReplay   bool           `json:"anti_replay"`
}

// VerifyOptions 验证选项
type VerifyOptions struct {
	MaxAge       time.Duration  `json:"max_age,omitempty"`
	Context      []byte         `json:"context,omitempty"`
	AntiReplay   bool           `json:"anti_replay"`
	PublicKey    []byte         `json:"public_key,omitempty"`
	TrustedKeys  [][]byte       `json:"trusted_keys,omitempty"`
}

// Config 加密配置
type Config struct {
	// 基本配置
	Algorithm       CryptoAlgorithm    `json:"algorithm"`
	SignAlgorithm   SignatureAlgorithm `json:"sign_algorithm"`
	HashAlgorithm   HashAlgorithm      `json:"hash_algorithm"`

	// 性能配置
	EnableHardware  bool `json:"enable_hardware"`  // 启用硬件加速
	ParallelWorkers int  `json:"parallel_workers"` // 并行工作者数量

	// 安全配置
	EnableObfuscation bool          `json:"enable_obfuscation"` // 启用混淆
	KeyRotationPeriod time.Duration `json:"key_rotation_period"`
	MaxSignatureAge   time.Duration `json:"max_signature_age"`

	// 存储配置
	KeyStorePath    string `json:"key_store_path"`
	EnableKeystore  bool   `json:"enable_keystore"`

	// 调试配置
	DebugMode bool `json:"debug_mode"`
	LogLevel  int  `json:"log_level"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Algorithm:         AlgorithmChaCha20Poly1305,
		SignAlgorithm:     SignatureEd25519,
		HashAlgorithm:     HashBlake2b,
		EnableHardware:    true,
		ParallelWorkers:   4,
		EnableObfuscation: true,
		KeyRotationPeriod: time.Hour * 24,
		MaxSignatureAge:   time.Minute * 15,
		EnableKeystore:    true,
		DebugMode:         false,
		LogLevel:          1,
	}
}

// CryptoContext 加密上下文
type CryptoContext struct {
	Context   context.Context
	Timeout   time.Duration
	Metadata  map[string]any
	RequestID string
}

// NewCryptoContext 创建加密上下文
func NewCryptoContext(ctx context.Context) *CryptoContext {
	return &CryptoContext{
		Context:   ctx,
		Timeout:   time.Second * 30,
		Metadata:  make(map[string]any),
		RequestID: generateRequestID(),
	}
}

