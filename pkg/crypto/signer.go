package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"sync"
	"time"
)

// AntiReverseSigner 抗逆向签名器
type AntiReverseSigner struct {
	algorithm    SignatureAlgorithm
	privateKey   []byte
	publicKey    []byte
	config       *Config
	replayCache  *ReplayCache
	obfuscationLayers []ObfuscationLayer
	mu           sync.RWMutex
}

// ObfuscationLayer 混淆层定义
type ObfuscationLayer struct {
	Name      string
	Transform func([]byte) []byte
	Inverse   func([]byte) []byte
}

// ReplayCache 重放攻击缓存
type ReplayCache struct {
	signatures map[string]time.Time
	maxAge     time.Duration
	mu         sync.RWMutex
}

// NewReplayCache 创建重放缓存
func NewReplayCache(maxAge time.Duration) *ReplayCache {
	cache := &ReplayCache{
		signatures: make(map[string]time.Time),
		maxAge:     maxAge,
	}

	// 启动清理goroutine
	go cache.cleanup()
	return cache
}

// cleanup 定期清理过期签名
func (rc *ReplayCache) cleanup() {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	for range ticker.C {
		rc.mu.Lock()
		now := time.Now()
		for sig, timestamp := range rc.signatures {
			if now.Sub(timestamp) > rc.maxAge {
				delete(rc.signatures, sig)
			}
		}
		rc.mu.Unlock()
	}
}

// Check 检查签名是否已存在
func (rc *ReplayCache) Check(signature string) bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	timestamp, exists := rc.signatures[signature]
	if !exists {
		return false
	}

	return time.Since(timestamp) <= rc.maxAge
}

// Add 添加签名到缓存
func (rc *ReplayCache) Add(signature string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.signatures[signature] = time.Now()
}

// NewAntiReverseSigner 创建抗逆向签名器
func NewAntiReverseSigner(algorithm SignatureAlgorithm, privateKey []byte, config *Config) (*AntiReverseSigner, error) {
	if config == nil {
		config = DefaultConfig()
	}

	signer := &AntiReverseSigner{
		algorithm:   algorithm,
		config:      config,
		replayCache: NewReplayCache(config.MaxSignatureAge),
	}

	// 设置密钥对
	if err := signer.setKeyPair(privateKey); err != nil {
		return nil, NewCryptoError("set_keypair", string(algorithm), err)
	}

	// 初始化混淆层
	signer.initObfuscationLayers()

	return signer, nil
}

// setKeyPair 设置密钥对
func (s *AntiReverseSigner) setKeyPair(privateKey []byte) error {
	switch s.algorithm {
	case SignatureEd25519:
		if len(privateKey) != ed25519.PrivateKeySize {
			return NewCryptoError("set_keypair", "ed25519", ErrInvalidKeySize).WithDetails("expected", ed25519.PrivateKeySize).WithDetails("actual", len(privateKey))
		}

		s.privateKey = make([]byte, len(privateKey))
		copy(s.privateKey, privateKey)

		// 从私钥派生公钥
		publicKey := make([]byte, ed25519.PublicKeySize)
		copy(publicKey, privateKey[32:]) // Ed25519私钥后32字节是公钥
		s.publicKey = publicKey

	default:
		return NewCryptoError("set_keypair", string(s.algorithm), ErrInvalidAlgorithm)
	}

	return nil
}

// initObfuscationLayers 初始化混淆层
func (s *AntiReverseSigner) initObfuscationLayers() {
	if !s.config.EnableObfuscation {
		return
	}

	// 第一层：简单XOR混淆
	xorLayer := ObfuscationLayer{
		Name: "xor_obfuscation",
		Transform: func(data []byte) []byte {
			result := make([]byte, len(data))
			for i, b := range data {
				// 使用位置相关的固定密钥
				key := byte(0x5A ^ (i % 256))
				result[i] = b ^ key
			}
			return result
		},
		Inverse: func(data []byte) []byte {
			// XOR是对称操作
			result := make([]byte, len(data))
			for i, b := range data {
				key := byte(0x5A ^ (i % 256))
				result[i] = b ^ key
			}
			return result
		},
	}

	// 第二层：位移混淆
	bitShiftLayer := ObfuscationLayer{
		Name: "bit_shift_obfuscation",
		Transform: func(data []byte) []byte {
			result := make([]byte, len(data))
			for i, b := range data {
				// 根据位置进行不同的位移
				shift := uint(i%7 + 1)
				result[i] = (b << shift) | (b >> (8 - shift))
			}
			return result
		},
		Inverse: func(data []byte) []byte {
			result := make([]byte, len(data))
			for i, b := range data {
				shift := uint(i%7 + 1)
				result[i] = (b >> shift) | (b << (8 - shift))
			}
			return result
		},
	}

	// 第三层：简单异或混淆（替换有问题的多项式）
	xorMixLayer := ObfuscationLayer{
		Name: "xor_mix_obfuscation",
		Transform: func(data []byte) []byte {
			result := make([]byte, len(data))
			for i, dataByte := range data {
				// 使用位置相关的多重异或
				key1 := byte(0x3A + i%13)
				key2 := byte(0x7F + i%17)
				result[i] = dataByte ^ key1 ^ key2
			}
			return result
		},
		Inverse: func(data []byte) []byte {
			// XOR是对称操作
			result := make([]byte, len(data))
			for i, dataByte := range data {
				key1 := byte(0x3A + i%13)
				key2 := byte(0x7F + i%17)
				result[i] = dataByte ^ key1 ^ key2
			}
			return result
		},
	}

	s.obfuscationLayers = []ObfuscationLayer{
		xorLayer,
		bitShiftLayer,
		xorMixLayer,
	}
}

// Sign 签名数据（抗逆向）
func (s *AntiReverseSigner) Sign(data []byte, options *SignOptions) (*Signature, error) {
	if len(data) == 0 {
		return nil, NewCryptoError("sign", string(s.algorithm), ErrInvalidInput)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	perf := NewPerformance("sign_" + string(s.algorithm))
	defer perf.Stop()

	// 预先生成时间戳和nonce，确保载荷构建时可用
	var timestamp int64
	var nonce []byte

	if options != nil && options.Timestamp != nil {
		timestamp = *options.Timestamp
	} else {
		timestamp = time.Now().Unix()
	}

	if options != nil && len(options.Nonce) > 0 {
		nonce = options.Nonce
	} else {
		var err error
		nonce, err = RandomBytes(16)
		if err != nil {
			return nil, NewCryptoError("generate_nonce", string(s.algorithm), err)
		}
	}

	// 创建完整的签名选项，包含生成的timestamp和nonce
	completeOptions := &SignOptions{
		Timestamp: &timestamp,
		Nonce:     nonce,
	}

	// 复制其他选项
	if options != nil {
		completeOptions.Context = options.Context
		completeOptions.TTL = options.TTL
		completeOptions.Metadata = options.Metadata
		completeOptions.AntiReplay = options.AntiReplay
	}

	// 构建签名载荷
	payload, err := s.buildSignaturePayload(data, completeOptions)
	if err != nil {
		return nil, NewCryptoError("build_payload", string(s.algorithm), err)
	}

	// 应用混淆层
	obfuscatedPayload := s.applyObfuscation(payload)

	// 执行实际签名
	var signature []byte
	switch s.algorithm {
	case SignatureEd25519:
		signature = ed25519.Sign(s.privateKey, obfuscatedPayload)
	default:
		return nil, NewCryptoError("sign", string(s.algorithm), ErrInvalidAlgorithm)
	}

	// 构建签名对象
	sig := &Signature{
		Algorithm: s.algorithm,
		Value:     signature,
		PublicKey: s.publicKey,
		Timestamp: timestamp,
		Nonce:     nonce,
		Version:   1,
		Metadata:  make(map[string]any),
	}

	// 添加抗逆向元数据
	sig.Metadata["entropy"] = Fingerprint(obfuscatedPayload)
	sig.Metadata["obfuscation_layers"] = len(s.obfuscationLayers)

	if options != nil && options.Metadata != nil {
		for k, v := range options.Metadata {
			sig.Metadata[k] = v
		}
	}

	return sig, nil
}

// Verify 验证签名（抗逆向）
func (s *AntiReverseSigner) Verify(data []byte, signature *Signature, options *VerifyOptions) error {
	if len(data) == 0 || signature == nil {
		return NewCryptoError("verify", string(s.algorithm), ErrInvalidInput)
	}

	if signature.Algorithm != s.algorithm {
		return NewCryptoError("verify", string(signature.Algorithm), ErrInvalidAlgorithm)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	perf := NewPerformance("verify_" + string(s.algorithm))
	defer perf.Stop()

	// 检查签名年龄
	if options != nil && options.MaxAge > 0 {
		age := time.Since(time.Unix(signature.Timestamp, 0))
		if age > options.MaxAge {
			return NewCryptoError("verify", string(s.algorithm), ErrSignatureExpired).WithDetails("age", age).WithDetails("max_age", options.MaxAge)
		}
	}

	// 检查重放攻击
	if options != nil && options.AntiReplay {
		sigFingerprint := s.calculateSignatureFingerprint(signature)
		if s.replayCache.Check(sigFingerprint) {
			return NewCryptoError("verify", string(s.algorithm), ErrSignatureReplay).WithDetails("fingerprint", sigFingerprint)
		}
		s.replayCache.Add(sigFingerprint)
	}

	// 重构签名选项
	signOptions := &SignOptions{
		Timestamp: &signature.Timestamp,
		Nonce:     signature.Nonce,
	}

	// 从验证选项中获取Context
	if options != nil {
		signOptions.Context = options.Context
	}

	// 构建验证载荷
	payload, err := s.buildSignaturePayload(data, signOptions)
	if err != nil {
		return NewCryptoError("build_verify_payload", string(s.algorithm), err)
	}

	// 应用混淆层
	obfuscatedPayload := s.applyObfuscation(payload)

	// 确定用于验证的公钥
	publicKey := s.publicKey
	if options != nil && len(options.PublicKey) > 0 {
		publicKey = options.PublicKey
	}

	// 执行签名验证
	var valid bool
	switch s.algorithm {
	case SignatureEd25519:
		if len(publicKey) != ed25519.PublicKeySize {
			return NewCryptoError("verify", "ed25519", ErrInvalidKey).WithDetails("key_size", len(publicKey))
		}
		valid = ed25519.Verify(publicKey, obfuscatedPayload, signature.Value)
	default:
		return NewCryptoError("verify", string(s.algorithm), ErrInvalidAlgorithm)
	}

	if !valid {
		return NewCryptoError("verify", string(s.algorithm), ErrInvalidSignature)
	}

	// 验证抗逆向元数据
	if err := s.verifyAntiReverseMetadata(signature, obfuscatedPayload); err != nil {
		return err
	}

	return nil
}

// buildSignaturePayload 构建签名载荷
func (s *AntiReverseSigner) buildSignaturePayload(data []byte, options *SignOptions) ([]byte, error) {
	// 计算数据哈希
	hash := sha256.Sum256(data)
	payload := hash[:]

	// 添加时间戳
	var timestamp int64
	if options != nil && options.Timestamp != nil {
		timestamp = *options.Timestamp
	} else {
		timestamp = time.Now().Unix()
	}

	timestampBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(timestampBytes, uint64(timestamp))
	payload = append(payload, timestampBytes...)

	// 添加随机数
	if options != nil && len(options.Nonce) > 0 {
		payload = append(payload, options.Nonce...)
	}

	// 添加上下文
	if options != nil && len(options.Context) > 0 {
		contextHash := sha256.Sum256(options.Context)
		payload = append(payload, contextHash[:]...)
	}

	// 添加版本标识
	payload = append(payload, 0x01) // version 1

	return payload, nil
}

// applyObfuscation 应用混淆层
func (s *AntiReverseSigner) applyObfuscation(data []byte) []byte {
	if !s.config.EnableObfuscation || len(s.obfuscationLayers) == 0 {
		return data
	}

	result := make([]byte, len(data))
	copy(result, data)

	// 依次应用每一层混淆
	for _, layer := range s.obfuscationLayers {
		result = layer.Transform(result)
	}

	return result
}

// removeObfuscation 移除混淆层
func (s *AntiReverseSigner) removeObfuscation(data []byte) []byte {
	if !s.config.EnableObfuscation || len(s.obfuscationLayers) == 0 {
		return data
	}

	result := make([]byte, len(data))
	copy(result, data)

	// 逆序移除混淆层
	for i := len(s.obfuscationLayers) - 1; i >= 0; i-- {
		result = s.obfuscationLayers[i].Inverse(result)
	}

	return result
}

// calculateSignatureFingerprint 计算签名指纹
func (s *AntiReverseSigner) calculateSignatureFingerprint(signature *Signature) string {
	hash := sha256.New()
	hash.Write(signature.Value)
	hash.Write(signature.Nonce)
	binary.Write(hash, binary.LittleEndian, signature.Timestamp)
	return Fingerprint(hash.Sum(nil))
}

// verifyAntiReverseMetadata 验证抗逆向元数据
func (s *AntiReverseSigner) verifyAntiReverseMetadata(signature *Signature, payload []byte) error {
	if !s.config.EnableObfuscation {
		return nil
	}

	// 验证载荷熵值
	if entropy, exists := signature.Metadata["entropy"]; exists {
		expectedEntropy := Fingerprint(payload)
		if !TimingSafeCompare(entropy.(string), expectedEntropy) {
			return NewSecurityError("tampering_detected", "payload entropy mismatch")
		}
	}

	// 验证混淆层数量
	if layerCount, exists := signature.Metadata["obfuscation_layers"]; exists {
		if layerCount.(int) != len(s.obfuscationLayers) {
			return NewSecurityError("tampering_detected", "obfuscation layer count mismatch")
		}
	}

	return nil
}

// Algorithm 获取签名算法
func (s *AntiReverseSigner) Algorithm() SignatureAlgorithm {
	return s.algorithm
}

// GetPublicKey 获取公钥
func (s *AntiReverseSigner) GetPublicKey() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]byte, len(s.publicKey))
	copy(result, s.publicKey)
	return result
}

// Cleanup 清理敏感数据
func (s *AntiReverseSigner) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.privateKey != nil {
		SecureZero(s.privateKey)
		s.privateKey = nil
	}

	if s.publicKey != nil {
		SecureZero(s.publicKey)
		s.publicKey = nil
	}
}

// GenerateKeyPair 生成新密钥对
func GenerateKeyPair(algorithm SignatureAlgorithm) (*KeyPair, error) {
	switch algorithm {
	case SignatureEd25519:
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, NewCryptoError("generate_keypair", "ed25519", err)
		}

		return &KeyPair{
			PrivateKey: privateKey,
			PublicKey:  publicKey,
			Algorithm:  SignatureEd25519,
			CreatedAt:  time.Now(),
			Metadata:   make(map[string]any),
		}, nil

	default:
		return nil, NewCryptoError("generate_keypair", string(algorithm), ErrInvalidAlgorithm)
	}
}

// ValidateKeyPair 验证密钥对
func ValidateKeyPair(keyPair *KeyPair) error {
	if keyPair == nil {
		return NewCryptoError("validate_keypair", "", ErrInvalidInput)
	}

	switch keyPair.Algorithm {
	case SignatureEd25519:
		if len(keyPair.PrivateKey) != ed25519.PrivateKeySize {
			return NewCryptoError("validate_keypair", "ed25519", ErrInvalidKeySize).WithDetails("expected", ed25519.PrivateKeySize).WithDetails("actual", len(keyPair.PrivateKey))
		}

		if len(keyPair.PublicKey) != ed25519.PublicKeySize {
			return NewCryptoError("validate_keypair", "ed25519", ErrInvalidKeySize).WithDetails("expected", ed25519.PublicKeySize).WithDetails("actual", len(keyPair.PublicKey))
		}

		// 验证公私钥匹配
		derivedPublic := make([]byte, ed25519.PublicKeySize)
		copy(derivedPublic, keyPair.PrivateKey[32:])

		if subtle.ConstantTimeCompare(derivedPublic, keyPair.PublicKey) != 1 {
			return NewCryptoError("validate_keypair", "ed25519", ErrInvalidKey).WithDetails("reason", "public key mismatch")
		}

	default:
		return NewCryptoError("validate_keypair", string(keyPair.Algorithm), ErrInvalidAlgorithm)
	}

	return nil
}