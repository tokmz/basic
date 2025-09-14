package crypto

import (
	"fmt"
	"sync"
	"time"
)

// SecureCryptoSuite 安全加密套件
type SecureCryptoSuite struct {
	config      *Config
	encryptor   *HighPerformanceEncryptor
	signer      *AntiReverseSigner
	keyManager  *KeyManager
	deriver     KeyDeriver
	obfuscator  *DataObfuscator
	initialized bool
	mu          sync.RWMutex
}

// NewSecureCryptoSuite 创建安全加密套件
func NewSecureCryptoSuite(config *Config) (*SecureCryptoSuite, error) {
	if config == nil {
		config = DefaultConfig()
	}

	suite := &SecureCryptoSuite{
		config: config,
	}

	// 初始化各个组件
	if err := suite.initialize(); err != nil {
		return nil, NewCryptoError("initialize_suite", "", err)
	}

	return suite, nil
}

// initialize 初始化套件组件
func (scs *SecureCryptoSuite) initialize() error {
	var err error

	// 初始化密钥管理器
	scs.keyManager, err = NewKeyManager(scs.config)
	if err != nil {
		return NewCryptoError("init_key_manager", "", err)
	}

	// 创建或获取主密钥
	masterKey, err := scs.getOrCreateMasterKey()
	if err != nil {
		return NewCryptoError("get_master_key", "", err)
	}

	// 初始化加密器
	scs.encryptor, err = NewHighPerformanceEncryptor(scs.config.Algorithm, masterKey, scs.config)
	if err != nil {
		return NewCryptoError("init_encryptor", "", err)
	}

	// 创建或获取签名密钥
	signingKey, err := scs.getOrCreateSigningKey()
	if err != nil {
		return NewCryptoError("get_signing_key", "", err)
	}

	// 初始化签名器
	scs.signer, err = NewAntiReverseSigner(scs.config.SignAlgorithm, signingKey, scs.config)
	if err != nil {
		return NewCryptoError("init_signer", "", err)
	}

	// 初始化密钥派生器
	scs.deriver = NewPBKDF2KeyDeriver(scs.config.HashAlgorithm, 100000)

	// 初始化混淆器
	if scs.config.EnableObfuscation {
		scs.obfuscator = NewDataObfuscator(masterKey, scs.config)
	}

	scs.initialized = true
	return nil
}

// getOrCreateMasterKey 获取或创建主密钥
func (scs *SecureCryptoSuite) getOrCreateMasterKey() ([]byte, error) {
	// 尝试从密钥管理器获取现有主密钥
	keys := scs.keyManager.ListKeys()
	for _, key := range keys {
		if key.Usage == KeyUsageMaster && key.Status == KeyStatusActive {
			fullKey, err := scs.keyManager.GetKey(key.ID)
			if err == nil && fullKey.Key != nil {
				return fullKey.Key, nil
			}
		}
	}

	// 创建新的主密钥
	masterKey, err := scs.keyManager.CreateKey(
		"master-encryption-key",
		"symmetric",
		KeyUsageMaster,
		WithMetadata(map[string]any{
			"purpose": "primary_encryption",
			"created_by": "secure_crypto_suite",
		}),
	)

	if err != nil {
		return nil, err
	}

	return masterKey.Key, nil
}

// getOrCreateSigningKey 获取或创建签名密钥
func (scs *SecureCryptoSuite) getOrCreateSigningKey() ([]byte, error) {
	// 尝试从密钥管理器获取现有签名密钥
	keys := scs.keyManager.ListKeys()
	for _, key := range keys {
		if key.Usage == KeyUsageSigning && key.Status == KeyStatusActive {
			fullKey, err := scs.keyManager.GetKey(key.ID)
			if err == nil && fullKey.Key != nil {
				return fullKey.Key, nil
			}
		}
	}

	// 创建新的签名密钥
	signingKey, err := scs.keyManager.CreateKey(
		"anti-reverse-signing-key",
		"asymmetric",
		KeyUsageSigning,
		WithMetadata(map[string]any{
			"purpose": "digital_signature",
			"algorithm": string(scs.config.SignAlgorithm),
			"created_by": "secure_crypto_suite",
		}),
	)

	if err != nil {
		return nil, err
	}

	return signingKey.Key, nil
}

// Encrypt 加密数据
func (scs *SecureCryptoSuite) Encrypt(plaintext []byte) (*EncryptedData, error) {
	if !scs.initialized {
		return nil, NewCryptoError("encrypt", "", ErrInvalidConfiguration).WithDetails("reason", "suite not initialized")
	}

	scs.mu.RLock()
	defer scs.mu.RUnlock()

	perf := NewPerformance("secure_encrypt")
	defer perf.Stop()

	// 应用预加密混淆
	processedData := plaintext
	if scs.obfuscator != nil {
		var err error
		processedData, err = scs.obfuscator.Obfuscate(plaintext)
		if err != nil {
			return nil, NewCryptoError("pre_encrypt_obfuscation", "", err)
		}
	}

	// 执行加密
	encrypted, err := scs.encryptor.Encrypt(processedData)
	if err != nil {
		return nil, NewCryptoError("encrypt", string(scs.config.Algorithm), err)
	}

	// 添加套件元数据
	if encrypted.Metadata == nil {
		encrypted.Metadata = make(map[string]any)
	}
	encrypted.Metadata["suite_version"] = "1.0"
	encrypted.Metadata["obfuscated"] = scs.obfuscator != nil
	encrypted.Metadata["fingerprint"] = Fingerprint(plaintext)

	return encrypted, nil
}

// Decrypt 解密数据
func (scs *SecureCryptoSuite) Decrypt(encrypted *EncryptedData) ([]byte, error) {
	if !scs.initialized {
		return nil, NewCryptoError("decrypt", "", ErrInvalidConfiguration).WithDetails("reason", "suite not initialized")
	}

	scs.mu.RLock()
	defer scs.mu.RUnlock()

	perf := NewPerformance("secure_decrypt")
	defer perf.Stop()

	// 执行解密
	decryptedData, err := scs.encryptor.Decrypt(encrypted)
	if err != nil {
		return nil, NewCryptoError("decrypt", string(encrypted.Algorithm), err)
	}

	// 应用逆向混淆
	if scs.obfuscator != nil && encrypted.Metadata != nil {
		if obfuscated, exists := encrypted.Metadata["obfuscated"]; exists && obfuscated.(bool) {
			decryptedData, err = scs.obfuscator.Deobfuscate(decryptedData)
			if err != nil {
				return nil, NewCryptoError("post_decrypt_deobfuscation", "", err)
			}
		}
	}

	// 验证数据完整性
	if encrypted.Metadata != nil {
		if expectedFingerprint, exists := encrypted.Metadata["fingerprint"]; exists {
			actualFingerprint := Fingerprint(decryptedData)
			if !TimingSafeCompare(expectedFingerprint.(string), actualFingerprint) {
				return nil, NewSecurityError("integrity_check_failed", "data fingerprint mismatch")
			}
		}
	}

	return decryptedData, nil
}

// Sign 签名数据
func (scs *SecureCryptoSuite) Sign(data []byte, options *SignOptions) (*Signature, error) {
	if !scs.initialized {
		return nil, NewCryptoError("sign", "", ErrInvalidConfiguration).WithDetails("reason", "suite not initialized")
	}

	scs.mu.RLock()
	defer scs.mu.RUnlock()

	perf := NewPerformance("secure_sign")
	defer perf.Stop()

	// 创建选项的副本，避免修改调用者的数据
	var signOptions *SignOptions
	if options != nil {
		signOptions = &SignOptions{
			Timestamp:  options.Timestamp,
			Nonce:      options.Nonce,
			TTL:        options.TTL,
			AntiReplay: options.AntiReplay,
			Metadata:   make(map[string]any),
		}

		// 复制元数据
		if options.Metadata != nil {
			for k, v := range options.Metadata {
				signOptions.Metadata[k] = v
			}
		}

		// 处理Context：创建新的切片，不修改原始数据
		suiteContext := fmt.Sprintf("SecureCryptoSuite-v1.0-%s", string(scs.config.SignAlgorithm))
		if options.Context == nil {
			signOptions.Context = []byte(suiteContext)
		} else {
			// 创建新切片，不修改原始Context
			prefixBytes := []byte(suiteContext + "_")
			signOptions.Context = make([]byte, 0, len(prefixBytes)+len(options.Context))
			signOptions.Context = append(signOptions.Context, prefixBytes...)
			signOptions.Context = append(signOptions.Context, options.Context...)
		}
	} else {
		signOptions = &SignOptions{
			AntiReplay: true,
			TTL:        scs.config.MaxSignatureAge,
		}
		suiteContext := fmt.Sprintf("SecureCryptoSuite-v1.0-%s", string(scs.config.SignAlgorithm))
		signOptions.Context = []byte(suiteContext)
	}

	// 默认启用防重放攻击（除非在原选项中明确禁用）
	if options == nil || options.AntiReplay {
		signOptions.AntiReplay = true
	}

	// 执行签名
	signature, err := scs.signer.Sign(data, signOptions)
	if err != nil {
		return nil, NewCryptoError("sign", string(scs.config.SignAlgorithm), err)
	}

	// 添加套件元数据
	if signature.Metadata == nil {
		signature.Metadata = make(map[string]any)
	}
	signature.Metadata["suite_version"] = "1.0"
	signature.Metadata["data_fingerprint"] = Fingerprint(data)

	return signature, nil
}

// Verify 验证签名
func (scs *SecureCryptoSuite) Verify(data []byte, signature *Signature, options *VerifyOptions) error {
	if !scs.initialized {
		return NewCryptoError("verify", "", ErrInvalidConfiguration).WithDetails("reason", "suite not initialized")
	}

	scs.mu.RLock()
	defer scs.mu.RUnlock()

	perf := NewPerformance("secure_verify")
	defer perf.Stop()

	// 创建选项的副本，避免修改调用者的数据
	var verifyOptions *VerifyOptions
	if options != nil {
		verifyOptions = &VerifyOptions{
			MaxAge:      options.MaxAge,
			AntiReplay:  options.AntiReplay,
			PublicKey:   options.PublicKey,
			TrustedKeys: options.TrustedKeys,
		}

		// 处理Context：创建新的切片，不修改原始数据
		suiteContext := fmt.Sprintf("SecureCryptoSuite-v1.0-%s", string(signature.Algorithm))
		if options.Context == nil {
			verifyOptions.Context = []byte(suiteContext)
		} else {
			// 创建新切片，不修改原始Context
			prefixBytes := []byte(suiteContext + "_")
			verifyOptions.Context = make([]byte, 0, len(prefixBytes)+len(options.Context))
			verifyOptions.Context = append(verifyOptions.Context, prefixBytes...)
			verifyOptions.Context = append(verifyOptions.Context, options.Context...)
		}
	} else {
		verifyOptions = &VerifyOptions{
			AntiReplay: true,
			MaxAge:     scs.config.MaxSignatureAge,
		}
		suiteContext := fmt.Sprintf("SecureCryptoSuite-v1.0-%s", string(signature.Algorithm))
		verifyOptions.Context = []byte(suiteContext)
	}

	// 默认启用防重放检查（除非在原选项中明确禁用）
	if options == nil || options.AntiReplay {
		verifyOptions.AntiReplay = true
	}

	// 验证套件元数据
	if signature.Metadata != nil {
		if fingerprint, exists := signature.Metadata["data_fingerprint"]; exists {
			actualFingerprint := Fingerprint(data)
			if !TimingSafeCompare(fingerprint.(string), actualFingerprint) {
				return NewSecurityError("data_tampering", "data fingerprint mismatch in signature")
			}
		}
	}

	// 执行验证
	return scs.signer.Verify(data, signature, verifyOptions)
}

// DeriveKey 派生密钥
func (scs *SecureCryptoSuite) DeriveKey(password []byte, salt []byte, keyLen int) ([]byte, error) {
	if !scs.initialized {
		return nil, NewCryptoError("derive_key", "", ErrInvalidConfiguration).WithDetails("reason", "suite not initialized")
	}

	return scs.deriver.DeriveKey(password, salt, keyLen)
}

// GenerateSalt 生成盐值
func (scs *SecureCryptoSuite) GenerateSalt() ([]byte, error) {
	if !scs.initialized {
		return nil, NewCryptoError("generate_salt", "", ErrInvalidConfiguration).WithDetails("reason", "suite not initialized")
	}

	return scs.deriver.GenerateSalt()
}

// Algorithm 获取加密算法
func (scs *SecureCryptoSuite) Algorithm() CryptoAlgorithm {
	return scs.config.Algorithm
}

// GenerateKeyPair 生成密钥对
func (scs *SecureCryptoSuite) GenerateKeyPair() (*KeyPair, error) {
	if !scs.initialized {
		return nil, NewCryptoError("generate_keypair", "", ErrInvalidConfiguration).WithDetails("reason", "suite not initialized")
	}

	return GenerateKeyPair(scs.config.SignAlgorithm)
}

// SetMasterKey 设置主密钥
func (scs *SecureCryptoSuite) SetMasterKey(key []byte) error {
	if !scs.initialized {
		return NewCryptoError("set_master_key", "", ErrInvalidConfiguration).WithDetails("reason", "suite not initialized")
	}

	scs.mu.Lock()
	defer scs.mu.Unlock()

	// 更新加密器密钥
	if err := scs.encryptor.UpdateKey(key); err != nil {
		return NewCryptoError("update_encryptor_key", "", err)
	}

	// 更新混淆器密钥
	if scs.obfuscator != nil {
		scs.obfuscator.UpdateKey(key)
	}

	return nil
}

// RotateKeys 轮换密钥
func (scs *SecureCryptoSuite) RotateKeys() error {
	if !scs.initialized {
		return NewCryptoError("rotate_keys", "", ErrInvalidConfiguration).WithDetails("reason", "suite not initialized")
	}

	scs.mu.Lock()
	defer scs.mu.Unlock()

	perf := NewPerformance("rotate_all_keys")
	defer perf.Stop()

	// 轮换主密钥
	keys := scs.keyManager.ListKeys()
	for _, key := range keys {
		if key.Usage == KeyUsageMaster && key.Status == KeyStatusActive {
			newKey, err := scs.keyManager.RotateKey(key.ID)
			if err != nil {
				return NewCryptoError("rotate_master_key", "", err)
			}

			// 更新加密器
			if err := scs.encryptor.UpdateKey(newKey.Key); err != nil {
				return NewCryptoError("update_encryptor_after_rotation", "", err)
			}

			// 更新混淆器
			if scs.obfuscator != nil {
				scs.obfuscator.UpdateKey(newKey.Key)
			}
		}

		if key.Usage == KeyUsageSigning && key.Status == KeyStatusActive {
			newKey, err := scs.keyManager.RotateKey(key.ID)
			if err != nil {
				return NewCryptoError("rotate_signing_key", "", err)
			}

			// 重新初始化签名器
			scs.signer, err = NewAntiReverseSigner(scs.config.SignAlgorithm, newKey.Key, scs.config)
			if err != nil {
				return NewCryptoError("reinit_signer_after_rotation", "", err)
			}
		}
	}

	return nil
}

// GetStats 获取套件统计信息
func (scs *SecureCryptoSuite) GetStats() map[string]any {
	if !scs.initialized {
		return map[string]any{"initialized": false}
	}

	scs.mu.RLock()
	defer scs.mu.RUnlock()

	stats := map[string]any{
		"initialized": true,
		"config": map[string]any{
			"algorithm":         string(scs.config.Algorithm),
			"sign_algorithm":    string(scs.config.SignAlgorithm),
			"hash_algorithm":    string(scs.config.HashAlgorithm),
			"obfuscation":       scs.config.EnableObfuscation,
			"hardware_accel":    scs.config.EnableHardware,
		},
		"components": map[string]any{
			"encryptor":   scs.encryptor != nil,
			"signer":      scs.signer != nil,
			"key_manager": scs.keyManager != nil,
			"obfuscator":  scs.obfuscator != nil,
		},
	}

	// 密钥管理统计
	if scs.keyManager != nil {
		keys := scs.keyManager.ListKeys()
		keyStats := map[string]int{
			"total":      len(keys),
			"active":     0,
			"inactive":   0,
			"expired":    0,
			"encryption": 0,
			"signing":    0,
			"derivation": 0,
			"master":     0,
		}

		for _, key := range keys {
			switch key.Status {
			case KeyStatusActive:
				keyStats["active"]++
			case KeyStatusInactive:
				keyStats["inactive"]++
			case KeyStatusExpired:
				keyStats["expired"]++
			}

			switch key.Usage {
			case KeyUsageEncryption:
				keyStats["encryption"]++
			case KeyUsageSigning:
				keyStats["signing"]++
			case KeyUsageDerivation:
				keyStats["derivation"]++
			case KeyUsageMaster:
				keyStats["master"]++
			}
		}

		stats["keys"] = keyStats
	}

	return stats
}

// Shutdown 关闭套件
func (scs *SecureCryptoSuite) Shutdown() error {
	scs.mu.Lock()
	defer scs.mu.Unlock()

	var errors []error

	// 清理加密器
	if scs.encryptor != nil {
		scs.encryptor.Cleanup()
		scs.encryptor = nil
	}

	// 清理签名器
	if scs.signer != nil {
		scs.signer.Cleanup()
		scs.signer = nil
	}

	// 停止密钥管理器
	if scs.keyManager != nil {
		if err := scs.keyManager.Stop(); err != nil {
			errors = append(errors, NewCryptoError("stop_key_manager", "", err))
		}
		scs.keyManager = nil
	}

	// 清理混淆器
	if scs.obfuscator != nil {
		scs.obfuscator.Cleanup()
		scs.obfuscator = nil
	}

	scs.initialized = false

	// 返回第一个错误（如果有）
	if len(errors) > 0 {
		return errors[0]
	}

	return nil
}

// IsInitialized 检查是否已初始化
func (scs *SecureCryptoSuite) IsInitialized() bool {
	scs.mu.RLock()
	defer scs.mu.RUnlock()
	return scs.initialized
}

// EncryptWithContext 带上下文的加密
func (scs *SecureCryptoSuite) EncryptWithContext(ctx *CryptoContext, plaintext []byte) (*EncryptedData, error) {
	if ctx != nil && ctx.Context != nil {
		// 检查上下文超时
		select {
		case <-ctx.Context.Done():
			return nil, NewCryptoError("encrypt_with_context", "", ErrContextCanceled)
		default:
		}

		// 设置超时
		if ctx.Timeout > 0 {
			timer := time.NewTimer(ctx.Timeout)
			defer timer.Stop()

			done := make(chan struct{})
			var result *EncryptedData
			var err error

			go func() {
				result, err = scs.Encrypt(plaintext)
				done <- struct{}{}
			}()

			select {
			case <-done:
				return result, err
			case <-timer.C:
				return nil, NewCryptoError("encrypt_with_context", "", ErrTimeout)
			case <-ctx.Context.Done():
				return nil, NewCryptoError("encrypt_with_context", "", ErrContextCanceled)
			}
		}
	}

	return scs.Encrypt(plaintext)
}

// DecryptWithContext 带上下文的解密
func (scs *SecureCryptoSuite) DecryptWithContext(ctx *CryptoContext, encrypted *EncryptedData) ([]byte, error) {
	if ctx != nil && ctx.Context != nil {
		// 检查上下文超时
		select {
		case <-ctx.Context.Done():
			return nil, NewCryptoError("decrypt_with_context", "", ErrContextCanceled)
		default:
		}

		// 设置超时
		if ctx.Timeout > 0 {
			timer := time.NewTimer(ctx.Timeout)
			defer timer.Stop()

			done := make(chan struct{})
			var result []byte
			var err error

			go func() {
				result, err = scs.Decrypt(encrypted)
				done <- struct{}{}
			}()

			select {
			case <-done:
				return result, err
			case <-timer.C:
				return nil, NewCryptoError("decrypt_with_context", "", ErrTimeout)
			case <-ctx.Context.Done():
				return nil, NewCryptoError("decrypt_with_context", "", ErrContextCanceled)
			}
		}
	}

	return scs.Decrypt(encrypted)
}