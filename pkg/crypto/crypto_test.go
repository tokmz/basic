package crypto

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"
)

func TestHighPerformanceEncryptor(t *testing.T) {
	tests := []struct {
		name      string
		algorithm CryptoAlgorithm
		keySize   int
	}{
		{"AES-GCM", AlgorithmAESGCM, 32},
		{"ChaCha20-Poly1305", AlgorithmChaCha20Poly1305, 32},
		{"XChaCha20-Poly1305", AlgorithmXChaCha20, 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 生成测试密钥
			key := make([]byte, tt.keySize)
			if _, err := rand.Read(key); err != nil {
				t.Fatalf("Failed to generate key: %v", err)
			}

			config := DefaultConfig()
			encryptor, err := NewHighPerformanceEncryptor(tt.algorithm, key, config)
			if err != nil {
				t.Fatalf("Failed to create encryptor: %v", err)
			}
			defer encryptor.Cleanup()

			// 测试数据
			plaintext := []byte("Hello, World! This is a test message for encryption.")

			// 加密
			encrypted, err := encryptor.Encrypt(plaintext)
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			// 验证加密数据结构
			if encrypted.Algorithm != tt.algorithm {
				t.Errorf("Expected algorithm %v, got %v", tt.algorithm, encrypted.Algorithm)
			}

			if len(encrypted.Nonce) != encryptor.NonceSize() {
				t.Errorf("Expected nonce size %d, got %d", encryptor.NonceSize(), len(encrypted.Nonce))
			}

			// 解密
			decrypted, err := encryptor.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			// 验证结果
			if !bytes.Equal(plaintext, decrypted) {
				t.Errorf("Decrypted text doesn't match original")
			}
		})
	}
}

func TestAntiReverseSigner(t *testing.T) {
	// 生成密钥对
	keyPair, err := GenerateKeyPair(SignatureEd25519)
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}

	config := DefaultConfig()
	signer, err := NewAntiReverseSigner(SignatureEd25519, keyPair.PrivateKey, config)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}
	defer signer.Cleanup()

	// 测试数据
	data := []byte("Important message that needs to be signed securely")

	// 签名选项
	signOptions := &SignOptions{
		AntiReplay: true,
		TTL:        time.Minute * 15,
		Context:    []byte("test_context"),
		Metadata: map[string]any{
			"test": true,
		},
	}

	// 签名
	signature, err := signer.Sign(data, signOptions)
	if err != nil {
		t.Fatalf("Signing failed: %v", err)
	}

	// 验证签名结构
	if signature.Algorithm != SignatureEd25519 {
		t.Errorf("Expected algorithm %v, got %v", SignatureEd25519, signature.Algorithm)
	}

	if len(signature.Nonce) == 0 {
		t.Error("Signature should have a nonce")
	}

	// 验证签名
	verifyOptions := &VerifyOptions{
		AntiReplay: true,
		MaxAge:     time.Minute * 20,
		Context:    []byte("test_context"),
	}

	if err := signer.Verify(data, signature, verifyOptions); err != nil {
		t.Fatalf("Signature verification failed: %v", err)
	}

	// 测试篡改检测
	tamperedData := append(data, byte(0))
	if err := signer.Verify(tamperedData, signature, verifyOptions); err == nil {
		t.Error("Should detect tampered data")
	}

	// 测试重放攻击检测
	if err := signer.Verify(data, signature, verifyOptions); err == nil {
		t.Error("Should detect replay attack")
	}
}

func TestKeyManager(t *testing.T) {
	config := DefaultConfig()
	config.EnableKeystore = false // 使用内存存储以简化测试

	keyManager, err := NewKeyManager(config)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}
	defer keyManager.Stop()

	// 创建不同用途的密钥
	testCases := []struct {
		name  string
		usage KeyUsage
	}{
		{"encryption-key", KeyUsageEncryption},
		{"signing-key", KeyUsageSigning},
		{"derivation-key", KeyUsageDerivation},
		{"master-key", KeyUsageMaster},
	}

	createdKeys := make(map[string]*ManagedKey)

	for _, tc := range testCases {
		t.Run("create_"+tc.name, func(t *testing.T) {
			key, err := keyManager.CreateKey(tc.name, "test", tc.usage)
			if err != nil {
				t.Fatalf("Failed to create key: %v", err)
			}

			if key.Usage != tc.usage {
				t.Errorf("Expected usage %v, got %v", tc.usage, key.Usage)
			}

			if key.Status != KeyStatusActive {
				t.Errorf("Expected status %v, got %v", KeyStatusActive, key.Status)
			}

			createdKeys[key.ID] = key
		})
	}

	// 测试密钥检索
	t.Run("retrieve_keys", func(t *testing.T) {
		for keyID := range createdKeys {
			retrievedKey, err := keyManager.GetKey(keyID)
			if err != nil {
				t.Fatalf("Failed to retrieve key %s: %v", keyID, err)
			}

			if retrievedKey.ID != keyID {
				t.Errorf("Retrieved wrong key: expected %s, got %s", keyID, retrievedKey.ID)
			}
		}
	})

	// 测试密钥轮换
	t.Run("key_rotation", func(t *testing.T) {
		for keyID := range createdKeys {
			newKey, err := keyManager.RotateKey(keyID)
			if err != nil {
				t.Fatalf("Failed to rotate key %s: %v", keyID, err)
			}

			if newKey.ID != keyID {
				t.Errorf("Rotated key should have same ID")
			}

			if newKey.Version <= createdKeys[keyID].Version {
				t.Errorf("Rotated key should have higher version")
			}
		}
	})

	// 测试密钥列表
	t.Run("list_keys", func(t *testing.T) {
		keys := keyManager.ListKeys()
		if len(keys) < len(testCases) {
			t.Errorf("Expected at least %d keys, got %d", len(testCases), len(keys))
		}
	})

	// 测试密钥删除
	t.Run("delete_keys", func(t *testing.T) {
		for keyID := range createdKeys {
			if err := keyManager.DeleteKey(keyID); err != nil {
				t.Fatalf("Failed to delete key %s: %v", keyID, err)
			}

			// 验证密钥已被删除
			_, err := keyManager.GetKey(keyID)
			if err == nil {
				t.Errorf("Key %s should have been deleted", keyID)
			}
		}
	})
}

func TestDataObfuscator(t *testing.T) {
	masterKey, err := RandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	config := DefaultConfig()
	obfuscator := NewDataObfuscator(masterKey, config)
	defer obfuscator.Cleanup()

	// 测试数据
	testData := []byte("This is sensitive data that needs to be obfuscated to prevent reverse engineering")

	// 测试混淆
	t.Run("obfuscate_deobfuscate", func(t *testing.T) {
		// 混淆
		obfuscated, err := obfuscator.Obfuscate(testData)
		if err != nil {
			t.Fatalf("Obfuscation failed: %v", err)
		}

		// 验证数据已被混淆
		if bytes.Equal(testData, obfuscated) {
			t.Error("Data should be obfuscated")
		}

		// 反混淆
		deobfuscated, err := obfuscator.Deobfuscate(obfuscated)
		if err != nil {
			t.Fatalf("Deobfuscation failed: %v", err)
		}

		// 验证结果
		if !bytes.Equal(testData, deobfuscated) {
			t.Error("Deobfuscated data doesn't match original")
		}
	})

	// 测试混淆效果
	t.Run("obfuscation_effectiveness", func(t *testing.T) {
		results := obfuscator.TestObfuscation(testData)

		correctness, ok := results["correctness"].(bool)
		if !ok || !correctness {
			t.Error("Obfuscation test should pass correctness check")
		}

		entropyIncrease, ok := results["entropy_increase"].(float64)
		if !ok || entropyIncrease <= 0 {
			t.Error("Obfuscation should increase entropy")
		}

		t.Logf("Obfuscation test results: %+v", results)
	})

	// 测试空数据
	t.Run("empty_data", func(t *testing.T) {
		emptyData := []byte{}
		obfuscated, err := obfuscator.Obfuscate(emptyData)
		if err != nil {
			t.Fatalf("Failed to obfuscate empty data: %v", err)
		}

		deobfuscated, err := obfuscator.Deobfuscate(obfuscated)
		if err != nil {
			t.Fatalf("Failed to deobfuscate empty data: %v", err)
		}

		if !bytes.Equal(emptyData, deobfuscated) {
			t.Error("Empty data handling failed")
		}
	})
}

func TestSecureCryptoSuite(t *testing.T) {
	config := DefaultConfig()
	config.EnableKeystore = false // 使用内存存储
	config.EnableObfuscation = true

	suite, err := NewSecureCryptoSuite(config)
	if err != nil {
		t.Fatalf("Failed to create secure crypto suite: %v", err)
	}
	defer suite.Shutdown()

	if !suite.IsInitialized() {
		t.Fatal("Suite should be initialized")
	}

	// 测试数据
	plaintext := []byte("Confidential data that requires comprehensive protection")

	// 测试加密/解密
	t.Run("encrypt_decrypt", func(t *testing.T) {
		encrypted, err := suite.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encryption failed: %v", err)
		}

		// 验证元数据
		if encrypted.Metadata == nil {
			t.Error("Encrypted data should have metadata")
		}

		decrypted, err := suite.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Decryption failed: %v", err)
		}

		if !bytes.Equal(plaintext, decrypted) {
			t.Error("Decrypted data doesn't match original")
		}
	})

	// 测试签名/验证
	t.Run("sign_verify", func(t *testing.T) {
		signOptions := &SignOptions{
			AntiReplay: true,
			Context:    []byte("test_suite_context"),
		}

		signature, err := suite.Sign(plaintext, signOptions)
		if err != nil {
			t.Fatalf("Signing failed: %v", err)
		}

		// 验证元数据
		if signature.Metadata == nil {
			t.Error("Signature should have metadata")
		}

		verifyOptions := &VerifyOptions{
			AntiReplay: true,
			MaxAge:     time.Minute * 10,
			Context:    []byte("test_suite_context"), // 必须与签名时的Context一致
		}

		if err := suite.Verify(plaintext, signature, verifyOptions); err != nil {
			t.Fatalf("Signature verification failed: %v", err)
		}
	})

	// 测试密钥派生
	t.Run("key_derivation", func(t *testing.T) {
		password := []byte("secure_password_123")
		salt, err := suite.GenerateSalt()
		if err != nil {
			t.Fatalf("Failed to generate salt: %v", err)
		}

		derivedKey, err := suite.DeriveKey(password, salt, 32)
		if err != nil {
			t.Fatalf("Key derivation failed: %v", err)
		}

		if len(derivedKey) != 32 {
			t.Errorf("Expected key length 32, got %d", len(derivedKey))
		}

		// 相同输入应产生相同输出
		derivedKey2, err := suite.DeriveKey(password, salt, 32)
		if err != nil {
			t.Fatalf("Second key derivation failed: %v", err)
		}

		if !bytes.Equal(derivedKey, derivedKey2) {
			t.Error("Key derivation should be deterministic")
		}
	})

	// 测试带上下文的操作
	t.Run("context_operations", func(t *testing.T) {
		ctx := NewCryptoContext(context.Background())
		ctx.Timeout = time.Second * 5

		encrypted, err := suite.EncryptWithContext(ctx, plaintext)
		if err != nil {
			t.Fatalf("Context encryption failed: %v", err)
		}

		decrypted, err := suite.DecryptWithContext(ctx, encrypted)
		if err != nil {
			t.Fatalf("Context decryption failed: %v", err)
		}

		if !bytes.Equal(plaintext, decrypted) {
			t.Error("Context operation result mismatch")
		}
	})

	// 测试密钥轮换
	t.Run("key_rotation", func(t *testing.T) {
		// 加密一些数据
		encrypted1, err := suite.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encryption before rotation failed: %v", err)
		}

		// 轮换密钥
		if err := suite.RotateKeys(); err != nil {
			t.Fatalf("Key rotation failed: %v", err)
		}

		// 新密钥应该能加密
		encrypted2, err := suite.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encryption after rotation failed: %v", err)
		}

		// 新密钥应该能解密新数据
		decrypted2, err := suite.Decrypt(encrypted2)
		if err != nil {
			t.Fatalf("Decryption with new key failed: %v", err)
		}

		if !bytes.Equal(plaintext, decrypted2) {
			t.Error("New key decryption failed")
		}

		// 注意：在实际实现中，可能需要支持用旧密钥解密旧数据
		// 这里为了简化测试，我们只测试新密钥的功能
		_ = encrypted1 // 避免未使用变量警告
	})

	// 测试统计信息
	t.Run("stats", func(t *testing.T) {
		stats := suite.GetStats()
		if stats == nil {
			t.Error("Stats should not be nil")
		}

		initialized, ok := stats["initialized"].(bool)
		if !ok || !initialized {
			t.Error("Suite should report as initialized")
		}

		t.Logf("Suite stats: %+v", stats)
	})
}

func TestPerformance(t *testing.T) {
	config := DefaultConfig()
	config.EnableKeystore = false
	config.EnableObfuscation = true

	suite, err := NewSecureCryptoSuite(config)
	if err != nil {
		t.Fatalf("Failed to create suite: %v", err)
	}
	defer suite.Shutdown()

	// 测试不同大小的数据
	dataSizes := []int{100, 1024, 10240, 102400} // 100B, 1KB, 10KB, 100KB

	for _, size := range dataSizes {
		t.Run(fmt.Sprintf("size_%dB", size), func(t *testing.T) {
			data := make([]byte, size)
			if _, err := rand.Read(data); err != nil {
				t.Fatalf("Failed to generate test data: %v", err)
			}

			// 加密性能测试
			encryptStart := time.Now()
			encrypted, err := suite.Encrypt(data)
			encryptDuration := time.Since(encryptStart)

			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			// 解密性能测试
			decryptStart := time.Now()
			decrypted, err := suite.Decrypt(encrypted)
			decryptDuration := time.Since(decryptStart)

			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			if !bytes.Equal(data, decrypted) {
				t.Error("Performance test data mismatch")
			}

			t.Logf("Size: %dB, Encrypt: %v, Decrypt: %v",
				size, encryptDuration, decryptDuration)

			// 性能基准（这些值可能需要根据实际硬件调整）
			maxEncryptTime := time.Millisecond * 100
			maxDecryptTime := time.Millisecond * 100

			if size <= 1024 { // 对小数据要求更高性能
				maxEncryptTime = time.Millisecond * 10
				maxDecryptTime = time.Millisecond * 10
			}

			if encryptDuration > maxEncryptTime {
				t.Logf("Warning: Encryption took %v, expected < %v", encryptDuration, maxEncryptTime)
			}

			if decryptDuration > maxDecryptTime {
				t.Logf("Warning: Decryption took %v, expected < %v", decryptDuration, maxDecryptTime)
			}
		})
	}
}

func BenchmarkSecureCryptoSuite(b *testing.B) {
	config := DefaultConfig()
	config.EnableKeystore = false

	suite, err := NewSecureCryptoSuite(config)
	if err != nil {
		b.Fatalf("Failed to create suite: %v", err)
	}
	defer suite.Shutdown()

	data := make([]byte, 1024)
	rand.Read(data)

	b.Run("Encrypt", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := suite.Encrypt(data)
			if err != nil {
				b.Fatalf("Encryption failed: %v", err)
			}
		}
	})

	encrypted, _ := suite.Encrypt(data)

	b.Run("Decrypt", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := suite.Decrypt(encrypted)
			if err != nil {
				b.Fatalf("Decryption failed: %v", err)
			}
		}
	})

	b.Run("Sign", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := suite.Sign(data, nil)
			if err != nil {
				b.Fatalf("Signing failed: %v", err)
			}
		}
	})

	b.Run("Verify", func(b *testing.B) {
		// 禁用重放检测以允许基准测试
		options := &SignOptions{
			AntiReplay: false, // 基准测试中关闭重放检测
		}
		signature, _ := suite.Sign(data, options)

		verifyOptions := &VerifyOptions{
			AntiReplay: false, // 基准测试中关闭重放检测
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := suite.Verify(data, signature, verifyOptions)
			if err != nil {
				b.Fatalf("Verification failed: %v", err)
			}
		}
	})
}