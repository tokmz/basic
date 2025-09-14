package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"sync"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

// HighPerformanceEncryptor 高性能加密器
type HighPerformanceEncryptor struct {
	algorithm CryptoAlgorithm
	key       []byte
	config    *Config
	gcm       cipher.AEAD
	chacha    cipher.AEAD
}

// NewHighPerformanceEncryptor 创建高性能加密器
func NewHighPerformanceEncryptor(algorithm CryptoAlgorithm, key []byte, config *Config) (*HighPerformanceEncryptor, error) {
	if config == nil {
		config = DefaultConfig()
	}

	encryptor := &HighPerformanceEncryptor{
		algorithm: algorithm,
		key:       make([]byte, len(key)),
		config:    config,
	}

	copy(encryptor.key, key)

	// 预初始化cipher以提高性能
	if err := encryptor.initCiphers(); err != nil {
		return nil, NewCryptoError("init_ciphers", string(algorithm), err)
	}

	return encryptor, nil
}

// initCiphers 初始化cipher实例
func (e *HighPerformanceEncryptor) initCiphers() error {
	switch e.algorithm {
	case AlgorithmAESGCM:
		if len(e.key) != 32 {
			// 如果密钥长度不对，派生32字节密钥
			hash := sha256.Sum256(e.key)
			e.key = hash[:]
		}

		block, err := aes.NewCipher(e.key)
		if err != nil {
			return err
		}

		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return err
		}
		e.gcm = gcm

	case AlgorithmChaCha20Poly1305:
		if len(e.key) != 32 {
			// 派生32字节密钥
			hash := sha256.Sum256(e.key)
			e.key = hash[:]
		}

		chacha, err := chacha20poly1305.New(e.key)
		if err != nil {
			return err
		}
		e.chacha = chacha

	case AlgorithmXChaCha20:
		if len(e.key) != 32 {
			hash := sha256.Sum256(e.key)
			e.key = hash[:]
		}

		xchacha, err := chacha20poly1305.NewX(e.key)
		if err != nil {
			return err
		}
		e.chacha = xchacha

	default:
		return NewCryptoError("init_ciphers", string(e.algorithm), ErrInvalidAlgorithm)
	}

	return nil
}

// Encrypt 加密数据
func (e *HighPerformanceEncryptor) Encrypt(plaintext []byte) (*EncryptedData, error) {
	if len(plaintext) == 0 {
		return nil, NewCryptoError("encrypt", string(e.algorithm), ErrInvalidInput)
	}

	perf := NewPerformance("encrypt_" + string(e.algorithm))
	defer perf.Stop()

	// 应用预处理混淆
	if e.config.EnableObfuscation {
		obfuscationKey, err := e.deriveObfuscationKey()
		if err != nil {
			return nil, NewCryptoError("derive_obfuscation_key", string(e.algorithm), err)
		}
		plaintext = Obfuscate(plaintext, obfuscationKey)
	}

	switch e.algorithm {
	case AlgorithmAESGCM:
		return e.encryptAESGCM(plaintext)
	case AlgorithmChaCha20Poly1305, AlgorithmXChaCha20:
		return e.encryptChaCha20(plaintext)
	default:
		return nil, NewCryptoError("encrypt", string(e.algorithm), ErrInvalidAlgorithm)
	}
}

// encryptAESGCM AES-GCM加密
func (e *HighPerformanceEncryptor) encryptAESGCM(plaintext []byte) (*EncryptedData, error) {
	// 生成随机nonce
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, NewCryptoError("generate_nonce", "aes-gcm", err)
	}

	// 加密
	ciphertext := e.gcm.Seal(nil, nonce, plaintext, nil)

	// AES-GCM的tag是附加在ciphertext末尾的
	tagSize := e.gcm.Overhead()
	if len(ciphertext) < tagSize {
		return nil, NewCryptoError("encrypt", "aes-gcm", ErrEncryptionFailed)
	}

	actualCiphertext := ciphertext[:len(ciphertext)-tagSize]
	tag := ciphertext[len(ciphertext)-tagSize:]

	return &EncryptedData{
		Algorithm:  AlgorithmAESGCM,
		Nonce:      nonce,
		Ciphertext: actualCiphertext,
		Tag:        tag,
		Version:    1,
		Timestamp:  time.Now().Unix(),
	}, nil
}

// encryptChaCha20 ChaCha20-Poly1305加密
func (e *HighPerformanceEncryptor) encryptChaCha20(plaintext []byte) (*EncryptedData, error) {
	// 生成随机nonce
	nonce := make([]byte, e.chacha.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, NewCryptoError("generate_nonce", "chacha20", err)
	}

	// 加密
	ciphertext := e.chacha.Seal(nil, nonce, plaintext, nil)

	// ChaCha20-Poly1305的tag也是附加在末尾的
	tagSize := e.chacha.Overhead()
	if len(ciphertext) < tagSize {
		return nil, NewCryptoError("encrypt", "chacha20", ErrEncryptionFailed)
	}

	actualCiphertext := ciphertext[:len(ciphertext)-tagSize]
	tag := ciphertext[len(ciphertext)-tagSize:]

	return &EncryptedData{
		Algorithm:  e.algorithm,
		Nonce:      nonce,
		Ciphertext: actualCiphertext,
		Tag:        tag,
		Version:    1,
		Timestamp:  time.Now().Unix(),
	}, nil
}

// Decrypt 解密数据
func (e *HighPerformanceEncryptor) Decrypt(encrypted *EncryptedData) ([]byte, error) {
	if encrypted == nil {
		return nil, NewCryptoError("decrypt", string(e.algorithm), ErrInvalidInput)
	}

	if encrypted.Algorithm != e.algorithm {
		return nil, NewCryptoError("decrypt", string(encrypted.Algorithm), ErrInvalidAlgorithm)
	}

	perf := NewPerformance("decrypt_" + string(e.algorithm))
	defer perf.Stop()

	var plaintext []byte
	var err error

	switch encrypted.Algorithm {
	case AlgorithmAESGCM:
		plaintext, err = e.decryptAESGCM(encrypted)
	case AlgorithmChaCha20Poly1305, AlgorithmXChaCha20:
		plaintext, err = e.decryptChaCha20(encrypted)
	default:
		return nil, NewCryptoError("decrypt", string(encrypted.Algorithm), ErrInvalidAlgorithm)
	}

	if err != nil {
		return nil, err
	}

	// 应用逆向混淆
	if e.config.EnableObfuscation {
		obfuscationKey, err := e.deriveObfuscationKey()
		if err != nil {
			return nil, NewCryptoError("derive_obfuscation_key", string(e.algorithm), err)
		}
		plaintext = Deobfuscate(plaintext, obfuscationKey)
	}

	return plaintext, nil
}

// decryptAESGCM AES-GCM解密
func (e *HighPerformanceEncryptor) decryptAESGCM(encrypted *EncryptedData) ([]byte, error) {
	// 重构完整的ciphertext（密文+tag）
	fullCiphertext := append(encrypted.Ciphertext, encrypted.Tag...)

	// 解密
	plaintext, err := e.gcm.Open(nil, encrypted.Nonce, fullCiphertext, nil)
	if err != nil {
		return nil, NewCryptoError("decrypt", "aes-gcm", ErrDecryptionFailed).WithDetails("auth_error", err)
	}

	return plaintext, nil
}

// decryptChaCha20 ChaCha20-Poly1305解密
func (e *HighPerformanceEncryptor) decryptChaCha20(encrypted *EncryptedData) ([]byte, error) {
	// 重构完整的ciphertext（密文+tag）
	fullCiphertext := append(encrypted.Ciphertext, encrypted.Tag...)

	// 解密
	plaintext, err := e.chacha.Open(nil, encrypted.Nonce, fullCiphertext, nil)
	if err != nil {
		return nil, NewCryptoError("decrypt", "chacha20", ErrDecryptionFailed).WithDetails("auth_error", err)
	}

	return plaintext, nil
}

// Algorithm 获取算法类型
func (e *HighPerformanceEncryptor) Algorithm() CryptoAlgorithm {
	return e.algorithm
}

// deriveObfuscationKey 派生混淆密钥
func (e *HighPerformanceEncryptor) deriveObfuscationKey() ([]byte, error) {
	// 使用主密钥和固定盐值派生混淆密钥
	salt := []byte("obfuscation_salt_v1")
	hash := sha256.New()
	hash.Write(e.key)
	hash.Write(salt)
	return hash.Sum(nil)[:16], nil // 使用前16字节作为混淆密钥
}

// UpdateKey 更新加密密钥
func (e *HighPerformanceEncryptor) UpdateKey(newKey []byte) error {
	if len(newKey) == 0 {
		return NewCryptoError("update_key", string(e.algorithm), ErrInvalidKey)
	}

	// 清零旧密钥
	SecureZero(e.key)

	// 设置新密钥
	e.key = make([]byte, len(newKey))
	copy(e.key, newKey)

	// 重新初始化cipher
	return e.initCiphers()
}

// KeySize 获取密钥大小
func (e *HighPerformanceEncryptor) KeySize() int {
	return len(e.key)
}

// NonceSize 获取nonce大小
func (e *HighPerformanceEncryptor) NonceSize() int {
	switch e.algorithm {
	case AlgorithmAESGCM:
		if e.gcm != nil {
			return e.gcm.NonceSize()
		}
		return 12 // 默认GCM nonce大小
	case AlgorithmChaCha20Poly1305:
		return chacha20poly1305.NonceSize
	case AlgorithmXChaCha20:
		return chacha20poly1305.NonceSizeX
	default:
		return 12
	}
}

// Cleanup 清理敏感数据
func (e *HighPerformanceEncryptor) Cleanup() {
	if e.key != nil {
		SecureZero(e.key)
		e.key = nil
	}
	e.gcm = nil
	e.chacha = nil
}

// 性能优化：预分配缓冲区池
type bufferPool struct {
	pool []*[]byte
	mu   sync.Mutex
}

func newBufferPool() *bufferPool {
	return &bufferPool{
		pool: make([]*[]byte, 0, 16),
	}
}

func (p *bufferPool) get(size int) []byte {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.pool) > 0 {
		buf := p.pool[len(p.pool)-1]
		p.pool = p.pool[:len(p.pool)-1]
		if cap(*buf) >= size {
			return (*buf)[:size]
		}
	}

	return make([]byte, size)
}

func (p *bufferPool) put(buf []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.pool) < 16 && cap(buf) >= 64 {
		// 清零缓冲区
		for i := range buf {
			buf[i] = 0
		}
		p.pool = append(p.pool, &buf)
	}
}

// 全局缓冲区池
var globalBufferPool = newBufferPool()

// GetBuffer 从池中获取缓冲区
func GetBuffer(size int) []byte {
	return globalBufferPool.get(size)
}

// PutBuffer 返回缓冲区到池中
func PutBuffer(buf []byte) {
	globalBufferPool.put(buf)
}