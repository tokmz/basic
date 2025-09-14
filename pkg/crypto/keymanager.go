package crypto

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/blake2b"
)

// KeyManager 密钥管理器
type KeyManager struct {
	config       *Config
	masterKey    []byte
	keys         map[string]*ManagedKey
	keyStore     KeyStore
	deriver      KeyDeriver
	mu           sync.RWMutex
	rotationTicker *time.Ticker
	stopCh       chan struct{}
}

// ManagedKey 管理的密钥
type ManagedKey struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Algorithm   string            `json:"algorithm"`
	Key         []byte            `json:"key"`
	PublicKey   []byte            `json:"public_key,omitempty"`
	Salt        []byte            `json:"salt,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
	RotationsAt *time.Time        `json:"rotations_at,omitempty"`
	Version     uint32            `json:"version"`
	Status      KeyStatus         `json:"status"`
	Usage       KeyUsage          `json:"usage"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
	AccessCount int64             `json:"access_count"`
	LastUsed    *time.Time        `json:"last_used,omitempty"`
}

// KeyStatus 密钥状态
type KeyStatus string

const (
	KeyStatusActive   KeyStatus = "active"
	KeyStatusInactive KeyStatus = "inactive"
	KeyStatusExpired  KeyStatus = "expired"
	KeyStatusRevoked  KeyStatus = "revoked"
)

// KeyUsage 密钥用途
type KeyUsage string

const (
	KeyUsageEncryption KeyUsage = "encryption"
	KeyUsageSigning    KeyUsage = "signing"
	KeyUsageDerivation KeyUsage = "derivation"
	KeyUsageMaster     KeyUsage = "master"
)

// KeyStore 密钥存储接口
type KeyStore interface {
	Store(key *ManagedKey) error
	Load(id string) (*ManagedKey, error)
	Delete(id string) error
	List() ([]*ManagedKey, error)
	Close() error
}

// FileKeyStore 文件密钥存储
type FileKeyStore struct {
	basePath  string
	masterKey []byte
	encryptor *HighPerformanceEncryptor
	mu        sync.RWMutex
}

// NewFileKeyStore 创建文件密钥存储
func NewFileKeyStore(basePath string, masterKey []byte) (*FileKeyStore, error) {
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return nil, NewCryptoError("create_keystore_dir", "", err)
	}

	// 创建加密器用于存储密钥
	encryptor, err := NewHighPerformanceEncryptor(AlgorithmChaCha20Poly1305, masterKey, DefaultConfig())
	if err != nil {
		return nil, NewCryptoError("create_encryptor", "", err)
	}

	return &FileKeyStore{
		basePath:  basePath,
		masterKey: masterKey,
		encryptor: encryptor,
	}, nil
}

// Store 存储密钥
func (fks *FileKeyStore) Store(key *ManagedKey) error {
	fks.mu.Lock()
	defer fks.mu.Unlock()

	// 序列化密钥
	keyData, err := json.Marshal(key)
	if err != nil {
		return NewCryptoError("marshal_key", "", err)
	}

	// 加密密钥数据
	encrypted, err := fks.encryptor.Encrypt(keyData)
	if err != nil {
		return NewCryptoError("encrypt_key", "", err)
	}

	// 序列化加密数据
	encryptedData, err := json.Marshal(encrypted)
	if err != nil {
		return NewCryptoError("marshal_encrypted_key", "", err)
	}

	// 写入文件
	keyPath := filepath.Join(fks.basePath, key.ID+".key")
	if err := os.WriteFile(keyPath, encryptedData, 0600); err != nil {
		return NewCryptoError("write_key_file", "", err)
	}

	return nil
}

// Load 加载密钥
func (fks *FileKeyStore) Load(id string) (*ManagedKey, error) {
	fks.mu.RLock()
	defer fks.mu.RUnlock()

	keyPath := filepath.Join(fks.basePath, id+".key")

	// 读取文件
	encryptedData, err := os.ReadFile(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrKeyNotFound
		}
		return nil, NewCryptoError("read_key_file", "", err)
	}

	// 反序列化加密数据
	var encrypted EncryptedData
	if err := json.Unmarshal(encryptedData, &encrypted); err != nil {
		return nil, NewCryptoError("unmarshal_encrypted_key", "", err)
	}

	// 解密密钥数据
	keyData, err := fks.encryptor.Decrypt(&encrypted)
	if err != nil {
		return nil, NewCryptoError("decrypt_key", "", err)
	}

	// 反序列化密钥
	var key ManagedKey
	if err := json.Unmarshal(keyData, &key); err != nil {
		return nil, NewCryptoError("unmarshal_key", "", err)
	}

	return &key, nil
}

// Delete 删除密钥
func (fks *FileKeyStore) Delete(id string) error {
	fks.mu.Lock()
	defer fks.mu.Unlock()

	keyPath := filepath.Join(fks.basePath, id+".key")
	if err := os.Remove(keyPath); err != nil {
		if os.IsNotExist(err) {
			return ErrKeyNotFound
		}
		return NewCryptoError("delete_key_file", "", err)
	}

	return nil
}

// List 列出所有密钥
func (fks *FileKeyStore) List() ([]*ManagedKey, error) {
	fks.mu.RLock()
	defer fks.mu.RUnlock()

	var keys []*ManagedKey

	err := filepath.WalkDir(fks.basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(path) != ".key" {
			return nil
		}

		keyID := filepath.Base(path[:len(path)-4]) // 移除.key扩展名

		key, err := fks.Load(keyID)
		if err != nil {
			// 跳过损坏的密钥文件
			return nil
		}

		keys = append(keys, key)
		return nil
	})

	if err != nil {
		return nil, NewCryptoError("list_keys", "", err)
	}

	return keys, nil
}

// Close 关闭存储
func (fks *FileKeyStore) Close() error {
	if fks.encryptor != nil {
		fks.encryptor.Cleanup()
	}
	return nil
}

// PBKDF2KeyDeriver PBKDF2密钥派生器
type PBKDF2KeyDeriver struct {
	algorithm HashAlgorithm
	iterations int
}

// NewPBKDF2KeyDeriver 创建PBKDF2密钥派生器
func NewPBKDF2KeyDeriver(algorithm HashAlgorithm, iterations int) *PBKDF2KeyDeriver {
	if iterations <= 0 {
		iterations = 100000 // 默认迭代次数
	}

	return &PBKDF2KeyDeriver{
		algorithm:  algorithm,
		iterations: iterations,
	}
}

// DeriveKey 派生密钥
func (kd *PBKDF2KeyDeriver) DeriveKey(password []byte, salt []byte, keyLen int) ([]byte, error) {
	if len(password) == 0 {
		return nil, NewCryptoError("derive_key", string(kd.algorithm), ErrInvalidInput)
	}

	if len(salt) == 0 {
		return nil, NewCryptoError("derive_key", string(kd.algorithm), ErrInvalidInput)
	}

	switch kd.algorithm {
	case HashArgon2id:
		// 使用Argon2id（推荐用于密码派生）
		key := argon2.IDKey(password, salt, uint32(kd.iterations/1000), 64*1024, 4, uint32(keyLen))
		return key, nil

	case HashBlake2b:
		// 使用Blake2b作为PRF的自定义PBKDF2
		return kd.pbkdf2Blake2b(password, salt, kd.iterations, keyLen), nil

	default:
		return nil, NewCryptoError("derive_key", string(kd.algorithm), ErrInvalidAlgorithm)
	}
}

// pbkdf2Blake2b 使用Blake2b的PBKDF2实现
func (kd *PBKDF2KeyDeriver) pbkdf2Blake2b(password, salt []byte, iterations, keyLen int) []byte {
	prf := func(key, data []byte) []byte {
		h, _ := blake2b.New256(key)
		h.Write(data)
		return h.Sum(nil)
	}

	return pbkdf2(prf, password, salt, iterations, keyLen)
}

// pbkdf2 通用PBKDF2实现
func pbkdf2(prf func([]byte, []byte) []byte, password, salt []byte, iterations, keyLen int) []byte {
	hLen := len(prf(password, salt))
	numBlocks := (keyLen + hLen - 1) / hLen

	var result []byte
	for i := 1; i <= numBlocks; i++ {
		block := make([]byte, 4)
		block[0] = byte(i >> 24)
		block[1] = byte(i >> 16)
		block[2] = byte(i >> 8)
		block[3] = byte(i)

		u := prf(password, append(salt, block...))
		sum := make([]byte, len(u))
		copy(sum, u)

		for j := 1; j < iterations; j++ {
			u = prf(password, u)
			XORBytes(sum, sum, u)
		}

		result = append(result, sum...)
	}

	return result[:keyLen]
}

// GenerateSalt 生成盐值
func (kd *PBKDF2KeyDeriver) GenerateSalt() ([]byte, error) {
	return SecureRandomBytes(32)
}

// Algorithm 获取算法
func (kd *PBKDF2KeyDeriver) Algorithm() HashAlgorithm {
	return kd.algorithm
}

// NewKeyManager 创建密钥管理器
func NewKeyManager(config *Config) (*KeyManager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	km := &KeyManager{
		config: config,
		keys:   make(map[string]*ManagedKey),
		stopCh: make(chan struct{}),
	}

	// 初始化密钥派生器
	km.deriver = NewPBKDF2KeyDeriver(config.HashAlgorithm, 100000)

	// 如果启用了密钥存储，初始化存储
	if config.EnableKeystore && config.KeyStorePath != "" {
		masterKey, err := km.generateMasterKey()
		if err != nil {
			return nil, NewCryptoError("generate_master_key", "", err)
		}

		keyStore, err := NewFileKeyStore(config.KeyStorePath, masterKey)
		if err != nil {
			return nil, NewCryptoError("create_keystore", "", err)
		}

		km.keyStore = keyStore
		km.masterKey = masterKey

		// 加载现有密钥
		if err := km.loadExistingKeys(); err != nil {
			return nil, NewCryptoError("load_existing_keys", "", err)
		}
	}

	// 启动密钥轮换
	if config.KeyRotationPeriod > 0 {
		km.startKeyRotation()
	}

	return km, nil
}

// generateMasterKey 生成主密钥
func (km *KeyManager) generateMasterKey() ([]byte, error) {
	// 生成高熵主密钥
	masterKey, err := SecureRandomBytes(64)
	if err != nil {
		return nil, err
	}

	// 验证熵值
	if !HasEntropy(masterKey, 6.0) {
		// 如果熵值不够，混合额外的随机源
		extraEntropy := gatherSystemEntropy()
		for i := 0; i < len(masterKey) && i < len(extraEntropy); i++ {
			masterKey[i] ^= extraEntropy[i]
		}
	}

	return masterKey, nil
}

// loadExistingKeys 加载现有密钥
func (km *KeyManager) loadExistingKeys() error {
	if km.keyStore == nil {
		return nil
	}

	keys, err := km.keyStore.List()
	if err != nil {
		return err
	}

	km.mu.Lock()
	defer km.mu.Unlock()

	for _, key := range keys {
		// 检查密钥是否过期
		if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
			key.Status = KeyStatusExpired
		}

		km.keys[key.ID] = key
	}

	return nil
}

// CreateKey 创建新密钥
func (km *KeyManager) CreateKey(name string, keyType string, usage KeyUsage, options ...KeyOption) (*ManagedKey, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	// 生成密钥ID
	keyID, err := km.generateKeyID()
	if err != nil {
		return nil, NewCryptoError("generate_key_id", "", err)
	}

	// 创建基础密钥对象
	key := &ManagedKey{
		ID:        keyID,
		Name:      name,
		Type:      keyType,
		Usage:     usage,
		CreatedAt: time.Now(),
		Version:   1,
		Status:    KeyStatusActive,
		Metadata:  make(map[string]any),
	}

	// 应用选项
	for _, opt := range options {
		opt(key)
	}

	// 根据用途生成密钥
	switch usage {
	case KeyUsageEncryption:
		key.Key, err = SecureRandomBytes(32)
		key.Algorithm = string(km.config.Algorithm)
	case KeyUsageSigning:
		keyPair, err := GenerateKeyPair(km.config.SignAlgorithm)
		if err != nil {
			return nil, NewCryptoError("generate_signing_key", string(km.config.SignAlgorithm), err)
		}
		key.Key = keyPair.PrivateKey
		key.PublicKey = keyPair.PublicKey
		key.Algorithm = string(km.config.SignAlgorithm)
	case KeyUsageDerivation:
		key.Salt, err = km.deriver.GenerateSalt()
		if err != nil {
			return nil, NewCryptoError("generate_salt", "", err)
		}
		key.Algorithm = string(km.config.HashAlgorithm)
	case KeyUsageMaster:
		key.Key, err = SecureRandomBytes(64)
		key.Algorithm = "master"
	default:
		return nil, NewCryptoError("create_key", "", ErrInvalidInput).WithDetails("usage", usage)
	}

	if err != nil {
		return nil, NewCryptoError("generate_key_material", "", err)
	}

	// 存储密钥
	km.keys[key.ID] = key

	if km.keyStore != nil {
		if err := km.keyStore.Store(key); err != nil {
			delete(km.keys, key.ID)
			return nil, NewCryptoError("store_key", "", err)
		}
	}

	return key, nil
}

// GetKey 获取密钥
func (km *KeyManager) GetKey(id string) (*ManagedKey, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	key, exists := km.keys[id]
	if !exists {
		return nil, ErrKeyNotFound
	}

	// 检查密钥状态
	if key.Status != KeyStatusActive {
		return nil, NewCryptoError("get_key", "", ErrKeyExpired).WithDetails("status", key.Status)
	}

	// 更新访问统计
	key.AccessCount++
	now := time.Now()
	key.LastUsed = &now

	return key, nil
}

// RotateKey 轮换密钥
func (km *KeyManager) RotateKey(id string) (*ManagedKey, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	oldKey, exists := km.keys[id]
	if !exists {
		return nil, ErrKeyNotFound
	}

	// 创建新版本的密钥
	newKey := &ManagedKey{
		ID:        oldKey.ID,
		Name:      oldKey.Name,
		Type:      oldKey.Type,
		Algorithm: oldKey.Algorithm,
		Usage:     oldKey.Usage,
		CreatedAt: time.Now(),
		Version:   oldKey.Version + 1,
		Status:    KeyStatusActive,
		Metadata:  make(map[string]any),
	}

	// 复制元数据
	for k, v := range oldKey.Metadata {
		newKey.Metadata[k] = v
	}

	var err error

	// 生成新的密钥材料
	switch oldKey.Usage {
	case KeyUsageEncryption:
		newKey.Key, err = SecureRandomBytes(32)
	case KeyUsageSigning:
		keyPair, err := GenerateKeyPair(SignatureAlgorithm(oldKey.Algorithm))
		if err != nil {
			return nil, err
		}
		newKey.Key = keyPair.PrivateKey
		newKey.PublicKey = keyPair.PublicKey
	case KeyUsageDerivation:
		newKey.Salt, err = km.deriver.GenerateSalt()
	case KeyUsageMaster:
		newKey.Key, err = SecureRandomBytes(64)
	}

	if err != nil {
		return nil, NewCryptoError("rotate_key", "", err)
	}

	// 标记旧密钥为非活动
	oldKey.Status = KeyStatusInactive

	// 更新存储
	km.keys[newKey.ID] = newKey

	if km.keyStore != nil {
		if err := km.keyStore.Store(newKey); err != nil {
			return nil, NewCryptoError("store_rotated_key", "", err)
		}
	}

	return newKey, nil
}

// DeleteKey 删除密钥
func (km *KeyManager) DeleteKey(id string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	key, exists := km.keys[id]
	if !exists {
		return ErrKeyNotFound
	}

	// 清零密钥材料
	if key.Key != nil {
		SecureZero(key.Key)
	}
	if key.PublicKey != nil {
		SecureZero(key.PublicKey)
	}

	// 从内存删除
	delete(km.keys, id)

	// 从存储删除
	if km.keyStore != nil {
		return km.keyStore.Delete(id)
	}

	return nil
}

// ListKeys 列出密钥
func (km *KeyManager) ListKeys() []*ManagedKey {
	km.mu.RLock()
	defer km.mu.RUnlock()

	var keys []*ManagedKey
	for _, key := range km.keys {
		// 返回副本，不包含敏感数据
		keyCopy := *key
		keyCopy.Key = nil
		keys = append(keys, &keyCopy)
	}

	return keys
}

// generateKeyID 生成密钥ID
func (km *KeyManager) generateKeyID() (string, error) {
	timestamp := time.Now().UnixNano()
	randomBytes, err := RandomBytes(8)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("key_%d_%s", timestamp, Encode(randomBytes, EncodingHex)), nil
}

// startKeyRotation 启动密钥轮换
func (km *KeyManager) startKeyRotation() {
	km.rotationTicker = time.NewTicker(km.config.KeyRotationPeriod)

	go func() {
		for {
			select {
			case <-km.rotationTicker.C:
				km.performScheduledRotation()
			case <-km.stopCh:
				return
			}
		}
	}()
}

// performScheduledRotation 执行预定的密钥轮换
func (km *KeyManager) performScheduledRotation() {
	km.mu.RLock()
	var keysToRotate []string

	now := time.Now()
	for _, key := range km.keys {
		if key.RotationsAt != nil && now.After(*key.RotationsAt) {
			keysToRotate = append(keysToRotate, key.ID)
		}
	}
	km.mu.RUnlock()

	// 轮换密钥
	for _, keyID := range keysToRotate {
		_, err := km.RotateKey(keyID)
		if err != nil {
			// 记录错误但继续轮换其他密钥
			fmt.Printf("Failed to rotate key %s: %v\n", keyID, err)
		}
	}
}

// Stop 停止密钥管理器
func (km *KeyManager) Stop() error {
	close(km.stopCh)

	if km.rotationTicker != nil {
		km.rotationTicker.Stop()
	}

	if km.keyStore != nil {
		return km.keyStore.Close()
	}

	return nil
}

// KeyOption 密钥选项
type KeyOption func(*ManagedKey)

// WithExpiration 设置过期时间
func WithExpiration(expiresAt time.Time) KeyOption {
	return func(key *ManagedKey) {
		key.ExpiresAt = &expiresAt
	}
}

// WithRotationSchedule 设置轮换时间
func WithRotationSchedule(rotationsAt time.Time) KeyOption {
	return func(key *ManagedKey) {
		key.RotationsAt = &rotationsAt
	}
}

// WithMetadata 添加元数据
func WithMetadata(metadata map[string]any) KeyOption {
	return func(key *ManagedKey) {
		for k, v := range metadata {
			key.Metadata[k] = v
		}
	}
}