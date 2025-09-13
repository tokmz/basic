package config

import (
	"fmt"
	"strings"
	"testing"
)

func TestNewAESEncryptor(t *testing.T) {
	// 测试正常情况
	encryptor, err := NewAESEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if encryptor == nil {
		t.Fatal("Encryptor should not be nil")
	}
	
	if len(encryptor.key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(encryptor.key))
	}
}

func TestNewAESEncryptorEmptyKey(t *testing.T) {
	_, err := NewAESEncryptor("")
	if err == nil {
		t.Error("Expected error for empty key")
	}
	
	expectedMessage := "encryption key cannot be empty"
	if !strings.Contains(err.Error(), expectedMessage) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedMessage, err.Error())
	}
}

func TestAESEncryptorEncryptDecrypt(t *testing.T) {
	encryptor, err := NewAESEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	testCases := []string{
		"simple string",
		"complex string with symbols: !@#$%^&*()",
		"multiline\nstring\nwith\nnewlines",
		"unicode string: 你好世界 🚀",
		"",  // empty string
		"a", // single character
		strings.Repeat("x", 1000), // long string
	}
	
	for i, original := range testCases {
		t.Run(fmt.Sprintf("TestCase_%d", i), func(t *testing.T) {
			// 加密
			encrypted, err := encryptor.Encrypt(original)
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}
			
			// 空字符串应该返回空字符串
			if original == "" && encrypted != "" {
				t.Error("Empty string should encrypt to empty string")
				return
			}
			
			// 非空字符串应该被加密
			if original != "" {
				if encrypted == "" {
					t.Error("Non-empty string should not encrypt to empty string")
					return
				}
				
				if encrypted == original {
					t.Error("Encrypted string should be different from original")
					return
				}
			}
			
			// 解密
			decrypted, err := encryptor.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}
			
			// 验证解密结果
			if decrypted != original {
				t.Errorf("Decrypted string doesn't match original. Expected '%s', got '%s'", original, decrypted)
			}
		})
	}
}

func TestAESEncryptorDecryptInvalidData(t *testing.T) {
	encryptor, err := NewAESEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	testCases := []struct {
		name  string
		input string
	}{
		{"invalid base64", "not-valid-base64!!!"},
		{"valid base64 but invalid ciphertext", "dGVzdA=="}, // "test" in base64
		{"too short ciphertext", "YWI="}, // "ab" in base64
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := encryptor.Decrypt(tc.input)
			if err == nil {
				t.Error("Expected decryption to fail for invalid input")
			}
		})
	}
}

func TestEncryptedViperManager(t *testing.T) {
	config := DefaultConfig()
	encryptedKeys := []string{"secret_key", "password"}
	
	manager, err := NewEncryptedViperManager(config, encryptedKeys)
	if err != nil {
		t.Fatalf("Failed to create encrypted manager: %v", err)
	}
	
	if manager == nil {
		t.Fatal("Manager should not be nil")
	}
	
	// 验证加密键配置
	if !manager.IsEncryptedKey("secret_key") {
		t.Error("secret_key should be marked as encrypted")
	}
	
	if !manager.IsEncryptedKey("password") {
		t.Error("password should be marked as encrypted")
	}
	
	if manager.IsEncryptedKey("normal_key") {
		t.Error("normal_key should not be marked as encrypted")
	}
}

func TestEncryptedViperManagerSetGetString(t *testing.T) {
	config := DefaultConfig()
	config.EncryptionKey = "test-encryption-key"
	encryptedKeys := []string{"secret_value"}
	
	manager, err := NewEncryptedViperManager(config, encryptedKeys)
	if err != nil {
		t.Fatalf("Failed to create encrypted manager: %v", err)
	}
	
	// 测试加密字段
	originalValue := "this is a secret"
	
	err = manager.SetString("secret_value", originalValue)
	if err != nil {
		t.Fatalf("SetString failed: %v", err)
	}
	
	// 获取值应该自动解密
	retrievedValue := manager.GetString("secret_value")
	if retrievedValue != originalValue {
		t.Errorf("Expected '%s', got '%s'", originalValue, retrievedValue)
	}
	
	// 测试非加密字段
	normalValue := "normal value"
	
	err = manager.SetString("normal_key", normalValue)
	if err != nil {
		t.Fatalf("SetString for normal key failed: %v", err)
	}
	
	retrievedNormalValue := manager.GetString("normal_key")
	if retrievedNormalValue != normalValue {
		t.Errorf("Expected '%s', got '%s'", normalValue, retrievedNormalValue)
	}
}

func TestEncryptedViperManagerExplicitEncryption(t *testing.T) {
	config := DefaultConfig()
	config.EncryptionKey = "test-encryption-key"
	
	manager, err := NewEncryptedViperManager(config, nil)
	if err != nil {
		t.Fatalf("Failed to create encrypted manager: %v", err)
	}
	
	originalValue := "explicit secret"
	
	// 显式加密设置
	err = manager.SetEncrypted("explicit_secret", originalValue)
	if err != nil {
		t.Fatalf("SetEncrypted failed: %v", err)
	}
	
	// 显式解密获取
	decryptedValue, err := manager.GetDecrypted("explicit_secret")
	if err != nil {
		t.Fatalf("GetDecrypted failed: %v", err)
	}
	
	if decryptedValue != originalValue {
		t.Errorf("Expected '%s', got '%s'", originalValue, decryptedValue)
	}
}

func TestEncryptedViperManagerNoEncryptor(t *testing.T) {
	config := DefaultConfig()
	// 没有设置加密密钥
	
	manager, err := NewEncryptedViperManager(config, []string{"secret_key"})
	if err != nil {
		t.Fatalf("Failed to create encrypted manager: %v", err)
	}
	
	// 没有加密器时，SetEncrypted应该返回错误
	err = manager.SetEncrypted("key", "value")
	if err == nil {
		t.Error("SetEncrypted should fail when no encryptor is configured")
	}
}

func TestEncryptedViperManagerKeyManagement(t *testing.T) {
	config := DefaultConfig()
	config.EncryptionKey = "test-encryption-key"
	
	manager, err := NewEncryptedViperManager(config, nil)
	if err != nil {
		t.Fatalf("Failed to create encrypted manager: %v", err)
	}
	
	// 添加加密键
	manager.AddEncryptedKey("new_secret")
	if !manager.IsEncryptedKey("new_secret") {
		t.Error("new_secret should be marked as encrypted after adding")
	}
	
	// 移除加密键
	manager.RemoveEncryptedKey("new_secret")
	if manager.IsEncryptedKey("new_secret") {
		t.Error("new_secret should not be marked as encrypted after removing")
	}
}

func TestEncryptedViperManagerProcessSecretValues(t *testing.T) {
	config := DefaultConfig()
	config.EncryptionKey = "test-encryption-key"
	
	manager, err := NewEncryptedViperManager(config, nil)
	if err != nil {
		t.Fatalf("Failed to create encrypted manager: %v", err)
	}
	
	// 设置包含特殊标记的值
	manager.Set("secret1", "${SECRET:plain_secret}")
	manager.Set("normal_value", "just_normal")
	
	// 处理敏感值
	err = manager.ProcessSecretValues()
	if err != nil {
		t.Fatalf("ProcessSecretValues failed: %v", err)
	}
	
	// ${SECRET:...} 格式的值应该被加密
	processedValue := manager.GetString("secret1")
	if processedValue == "${SECRET:plain_secret}" {
		t.Error("Secret value should have been processed")
	}
	
	// 正常值不应该被改变
	normalValue := manager.GetString("normal_value")
	if normalValue != "just_normal" {
		t.Error("Normal value should not be changed")
	}
}

func TestAESEncryptorDifferentKeys(t *testing.T) {
	encryptor1, _ := NewAESEncryptor("key1")
	encryptor2, _ := NewAESEncryptor("key2")
	
	original := "test message"
	
	// 用第一个加密器加密
	encrypted, err := encryptor1.Encrypt(original)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}
	
	// 用第二个加密器解密应该失败
	_, err = encryptor2.Decrypt(encrypted)
	if err == nil {
		t.Error("Decryption with different key should fail")
	}
	
	// 用正确的加密器解密应该成功
	decrypted, err := encryptor1.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decryption with correct key should succeed: %v", err)
	}
	
	if decrypted != original {
		t.Errorf("Expected '%s', got '%s'", original, decrypted)
	}
}

func BenchmarkAESEncryption(b *testing.B) {
	encryptor, _ := NewAESEncryptor("benchmark-key")
	testString := "This is a test string for benchmarking encryption performance"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encryptor.Encrypt(testString)
		if err != nil {
			b.Fatalf("Encryption failed: %v", err)
		}
	}
}

func BenchmarkAESDecryption(b *testing.B) {
	encryptor, _ := NewAESEncryptor("benchmark-key")
	testString := "This is a test string for benchmarking decryption performance"
	
	encrypted, _ := encryptor.Encrypt(testString)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encryptor.Decrypt(encrypted)
		if err != nil {
			b.Fatalf("Decryption failed: %v", err)
		}
	}
}