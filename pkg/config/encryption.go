package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// Encryptor 加密器接口
type Encryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// AESEncryptor AES加密器
type AESEncryptor struct {
	key []byte
}

// NewAESEncryptor 创建AES加密器
func NewAESEncryptor(key string) (*AESEncryptor, error) {
	if key == "" {
		return nil, fmt.Errorf("encryption key cannot be empty")
	}
	
	// 使用SHA256生成32字节的密钥
	hash := sha256.Sum256([]byte(key))
	
	return &AESEncryptor{
		key: hash[:],
	}, nil
}

// Encrypt 加密字符串
func (e *AESEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	
	// 创建GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	
	// 创建随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	// 加密
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	
	// 返回base64编码的结果
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密字符串
func (e *AESEncryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	
	// 解码base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	
	// 创建GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	
	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}
	
	return string(plaintext), nil
}

// EncryptedViperManager 支持加密的Viper配置管理器
type EncryptedViperManager struct {
	*ViperManager
	encryptor      Encryptor
	encryptedKeys  map[string]bool // 需要加密的键
}

// NewEncryptedViperManager 创建支持加密的配置管理器
func NewEncryptedViperManager(config *Config, encryptedKeys []string) (*EncryptedViperManager, error) {
	baseManager := NewViperManager(config)
	
	var encryptor Encryptor
	if config.EncryptionKey != "" {
		var err error
		encryptor, err = NewAESEncryptor(config.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create encryptor: %w", err)
		}
	}
	
	keyMap := make(map[string]bool)
	for _, key := range encryptedKeys {
		keyMap[key] = true
	}
	
	return &EncryptedViperManager{
		ViperManager:  baseManager,
		encryptor:     encryptor,
		encryptedKeys: keyMap,
	}, nil
}

// GetString 获取字符串配置（自动解密）
func (e *EncryptedViperManager) GetString(key string) string {
	value := e.ViperManager.GetString(key)
	
	// 如果是需要解密的键且有加密器
	if e.encryptedKeys[key] && e.encryptor != nil {
		if decrypted, err := e.encryptor.Decrypt(value); err == nil {
			return decrypted
		}
	}
	
	return value
}

// SetString 设置字符串配置（自动加密）
func (e *EncryptedViperManager) SetString(key, value string) error {
	// 如果是需要加密的键且有加密器
	if e.encryptedKeys[key] && e.encryptor != nil {
		encrypted, err := e.encryptor.Encrypt(value)
		if err != nil {
			return fmt.Errorf("failed to encrypt value for key %s: %w", key, err)
		}
		value = encrypted
	}
	
	e.ViperManager.Set(key, value)
	return nil
}

// GetDecrypted 获取解密后的配置值
func (e *EncryptedViperManager) GetDecrypted(key string) (string, error) {
	value := e.ViperManager.GetString(key)
	
	if e.encryptor == nil {
		return value, nil
	}
	
	return e.encryptor.Decrypt(value)
}

// SetEncrypted 设置加密后的配置值
func (e *EncryptedViperManager) SetEncrypted(key, value string) error {
	if e.encryptor == nil {
		return fmt.Errorf("no encryptor configured")
	}
	
	encrypted, err := e.encryptor.Encrypt(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt value: %w", err)
	}
	
	e.ViperManager.Set(key, encrypted)
	return nil
}

// IsEncryptedKey 检查是否为加密键
func (e *EncryptedViperManager) IsEncryptedKey(key string) bool {
	return e.encryptedKeys[key]
}

// AddEncryptedKey 添加需要加密的键
func (e *EncryptedViperManager) AddEncryptedKey(key string) {
	e.encryptedKeys[key] = true
}

// RemoveEncryptedKey 移除加密键
func (e *EncryptedViperManager) RemoveEncryptedKey(key string) {
	delete(e.encryptedKeys, key)
}

// ProcessSecretValues 处理配置中的敏感值
// 支持以下格式的敏感值标记：
// - ${SECRET:actual_value} - 运行时加密
// - ${ENCRYPTED:encrypted_value} - 已加密值
func (e *EncryptedViperManager) ProcessSecretValues() error {
	settings := e.GetAllSettings()
	
	for key, value := range settings {
		if strValue, ok := value.(string); ok {
			if processed, err := e.processSecretValue(strValue); err == nil && processed != strValue {
				e.Set(key, processed)
			}
		}
	}
	
	return nil
}

// processSecretValue 处理单个敏感值
func (e *EncryptedViperManager) processSecretValue(value string) (string, error) {
	if !strings.Contains(value, "${") {
		return value, nil
	}
	
	// 处理 ${SECRET:...} 格式
	if strings.HasPrefix(value, "${SECRET:") && strings.HasSuffix(value, "}") {
		secretValue := value[9 : len(value)-1] // 去掉 "${SECRET:" 和 "}"
		if e.encryptor != nil {
			return e.encryptor.Encrypt(secretValue)
		}
		return secretValue, nil
	}
	
	// 处理 ${ENCRYPTED:...} 格式
	if strings.HasPrefix(value, "${ENCRYPTED:") && strings.HasSuffix(value, "}") {
		encryptedValue := value[12 : len(value)-1] // 去掉 "${ENCRYPTED:" 和 "}"
		if e.encryptor != nil {
			return e.encryptor.Decrypt(encryptedValue)
		}
		return encryptedValue, nil
	}
	
	return value, nil
}