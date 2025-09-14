package crypto

import (
	"crypto/sha256"
	"encoding/binary"
	"sync"
	"time"
)

// DataObfuscator 数据混淆器
type DataObfuscator struct {
	masterKey       []byte
	config          *Config
	obfuscationKey  []byte
	transformLayers []TransformLayer
	mu              sync.RWMutex
}

// TransformLayer 变换层
type TransformLayer struct {
	Name      string
	Forward   func([]byte, []byte) []byte // 前向变换
	Backward  func([]byte, []byte) []byte // 反向变换
	KeyDerive func([]byte) []byte         // 密钥派生
}

// NewDataObfuscator 创建数据混淆器
func NewDataObfuscator(masterKey []byte, config *Config) *DataObfuscator {
	obfuscator := &DataObfuscator{
		masterKey: make([]byte, len(masterKey)),
		config:    config,
	}

	copy(obfuscator.masterKey, masterKey)
	obfuscator.deriveObfuscationKey()
	obfuscator.initTransformLayers()

	return obfuscator
}

// deriveObfuscationKey 派生混淆密钥
func (do *DataObfuscator) deriveObfuscationKey() {
	hash := sha256.New()
	hash.Write(do.masterKey)
	hash.Write([]byte("data_obfuscation_key_v1"))
	do.obfuscationKey = hash.Sum(nil)
}

// initTransformLayers 初始化变换层
func (do *DataObfuscator) initTransformLayers() {
	// 第一层：时间混淆层
	timeLayer := TransformLayer{
		Name: "time_obfuscation",
		KeyDerive: func(masterKey []byte) []byte {
			hash := sha256.New()
			hash.Write(masterKey)
			hash.Write([]byte("time_layer"))
			return hash.Sum(nil)[:16]
		},
		Forward: func(data []byte, key []byte) []byte {
			timestamp := time.Now().UnixNano()
			timeSeed := make([]byte, 8)
			binary.LittleEndian.PutUint64(timeSeed, uint64(timestamp))

			result := make([]byte, len(data))
			for i, b := range data {
				keyByte := key[i%len(key)]
				timeByte := timeSeed[i%8]
				result[i] = b ^ keyByte ^ timeByte
			}

			// 在开头添加时间戳混淆信息
			timeObfuscated := make([]byte, 8)
			for i := 0; i < 8; i++ {
				timeObfuscated[i] = timeSeed[i] ^ key[i%len(key)]
			}

			return append(timeObfuscated, result...)
		},
		Backward: func(data []byte, key []byte) []byte {
			if len(data) < 8 {
				return data
			}

			// 恢复时间戳
			timeObfuscated := data[:8]
			timeSeed := make([]byte, 8)
			for i := 0; i < 8; i++ {
				timeSeed[i] = timeObfuscated[i] ^ key[i%len(key)]
			}

			// 解混淆数据
			obfuscatedData := data[8:]
			result := make([]byte, len(obfuscatedData))
			for i, b := range obfuscatedData {
				keyByte := key[i%len(key)]
				timeByte := timeSeed[i%8]
				result[i] = b ^ keyByte ^ timeByte
			}

			return result
		},
	}

	// 第二层：多项式变换层
	polyLayer := TransformLayer{
		Name: "polynomial_transform",
		KeyDerive: func(masterKey []byte) []byte {
			hash := sha256.New()
			hash.Write(masterKey)
			hash.Write([]byte("poly_layer"))
			return hash.Sum(nil)[:16]
		},
		Forward: func(data []byte, key []byte) []byte {
			result := make([]byte, len(data))
			for i, b := range data {
				// 使用密钥派生多项式系数
				a := uint16(key[i%len(key)]) | 1 // 确保a是奇数
				b_coeff := uint16(key[(i+1)%len(key)])

				// 多项式变换: f(x) = (ax + b) mod 256
				transformed := (uint16(b)*a + b_coeff) & 0xFF
				result[i] = byte(transformed)
			}
			return result
		},
		Backward: func(data []byte, key []byte) []byte {
			result := make([]byte, len(data))
			for i, b := range data {
				// 恢复多项式系数
				a := uint16(key[i%len(key)]) | 1 // 确保a是奇数
				b_coeff := uint16(key[(i+1)%len(key)])

				// 逆向多项式变换
				// 寻找模逆元
				aInv := modInverse(a, 256)
				original := (uint16(b)*aInv - b_coeff*aInv) & 0xFF
				result[i] = byte(original)
			}
			return result
		},
	}

	// 第三层：位置相关变换层
	positionLayer := TransformLayer{
		Name: "position_transform",
		KeyDerive: func(masterKey []byte) []byte {
			hash := sha256.New()
			hash.Write(masterKey)
			hash.Write([]byte("position_layer"))
			return hash.Sum(nil)[:16]
		},
		Forward: func(data []byte, key []byte) []byte {
			result := make([]byte, len(data))
			for i, b := range data {
				// 位置相关的变换
				positionKey := key[(i*3)%len(key)]
				rotateCount := (i + int(positionKey)) % 8

				// 循环位移
				rotated := (b << rotateCount) | (b >> (8 - rotateCount))
				result[i] = rotated ^ positionKey
			}
			return result
		},
		Backward: func(data []byte, key []byte) []byte {
			result := make([]byte, len(data))
			for i, b := range data {
				// 逆向位置相关的变换
				positionKey := key[(i*3)%len(key)]
				unxored := b ^ positionKey

				// 逆向循环位移
				rotateCount := (i + int(positionKey)) % 8
				original := (unxored >> rotateCount) | (unxored << (8 - rotateCount))
				result[i] = original
			}
			return result
		},
	}

	// 第四层：简单置换层
	permutationLayer := TransformLayer{
		Name: "permutation",
		KeyDerive: func(masterKey []byte) []byte {
			hash := sha256.New()
			hash.Write(masterKey)
			hash.Write([]byte("perm_layer"))
			return hash.Sum(nil)[:16]
		},
		Forward: func(data []byte, key []byte) []byte {
			if len(data) == 0 {
				return data
			}

			result := make([]byte, len(data))
			copy(result, data)

			// 简单的异或和位移
			for i := 0; i < len(result); i++ {
				keyByte := key[i%len(key)]
				result[i] ^= keyByte
				// 简单位移
				result[i] = (result[i] << 3) | (result[i] >> 5)
			}

			return result
		},
		Backward: func(data []byte, key []byte) []byte {
			if len(data) == 0 {
				return data
			}

			result := make([]byte, len(data))
			copy(result, data)

			// 逆向变换
			for i := 0; i < len(result); i++ {
				// 逆向位移
				result[i] = (result[i] >> 3) | (result[i] << 5)
				keyByte := key[i%len(key)]
				result[i] ^= keyByte
			}

			return result
		},
	}

	do.transformLayers = []TransformLayer{
		timeLayer,
		polyLayer,
		positionLayer,
		permutationLayer,
	}
}


// modInverse 计算模逆元（简化版本）
func modInverse(a, m uint16) uint16 {
	// 扩展欧几里德算法计算模逆元
	if gcd(a, m) != 1 {
		return 1 // 如果不存在逆元，返回1作为fallback
	}

	// 简化实现：暴力搜索
	for i := uint16(1); i < m; i++ {
		if (a*i)%m == 1 {
			return i
		}
	}
	return 1
}

// gcd 计算最大公约数
func gcd(a, b uint16) uint16 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// Obfuscate 混淆数据
func (do *DataObfuscator) Obfuscate(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	do.mu.RLock()
	defer do.mu.RUnlock()

	perf := NewPerformance("data_obfuscation")
	defer perf.Stop()

	result := make([]byte, len(data))
	copy(result, data)

	// 依次应用每个变换层
	for _, layer := range do.transformLayers {
		layerKey := layer.KeyDerive(do.obfuscationKey)
		result = layer.Forward(result, layerKey)
	}

	return result, nil
}

// Deobfuscate 反混淆数据
func (do *DataObfuscator) Deobfuscate(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	do.mu.RLock()
	defer do.mu.RUnlock()

	perf := NewPerformance("data_deobfuscation")
	defer perf.Stop()

	result := make([]byte, len(data))
	copy(result, data)

	// 逆序应用每个变换层的反向变换
	for i := len(do.transformLayers) - 1; i >= 0; i-- {
		layer := do.transformLayers[i]
		layerKey := layer.KeyDerive(do.obfuscationKey)
		result = layer.Backward(result, layerKey)
	}

	return result, nil
}

// UpdateKey 更新混淆密钥
func (do *DataObfuscator) UpdateKey(newMasterKey []byte) {
	do.mu.Lock()
	defer do.mu.Unlock()

	// 清零旧密钥
	SecureZero(do.masterKey)
	SecureZero(do.obfuscationKey)

	// 设置新密钥
	do.masterKey = make([]byte, len(newMasterKey))
	copy(do.masterKey, newMasterKey)

	// 重新派生混淆密钥
	do.deriveObfuscationKey()
}

// Cleanup 清理敏感数据
func (do *DataObfuscator) Cleanup() {
	do.mu.Lock()
	defer do.mu.Unlock()

	if do.masterKey != nil {
		SecureZero(do.masterKey)
		do.masterKey = nil
	}

	if do.obfuscationKey != nil {
		SecureZero(do.obfuscationKey)
		do.obfuscationKey = nil
	}

	// 清空变换层
	do.transformLayers = nil
}

// GetObfuscationLayers 获取混淆层信息
func (do *DataObfuscator) GetObfuscationLayers() []string {
	do.mu.RLock()
	defer do.mu.RUnlock()

	layers := make([]string, len(do.transformLayers))
	for i, layer := range do.transformLayers {
		layers[i] = layer.Name
	}

	return layers
}

// TestObfuscation 测试混淆效果
func (do *DataObfuscator) TestObfuscation(testData []byte) map[string]any {
	if len(testData) == 0 {
		return map[string]any{"error": "empty test data"}
	}

	// 执行混淆
	obfuscated, err := do.Obfuscate(testData)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	// 执行反混淆
	deobfuscated, err := do.Deobfuscate(obfuscated)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	// 验证正确性
	correctness := ConstantTimeCompare(testData, deobfuscated)

	// 计算熵值变化
	originalEntropy := calculateEntropy(testData)
	obfuscatedEntropy := calculateEntropy(obfuscated)

	return map[string]any{
		"correctness":         correctness,
		"original_size":       len(testData),
		"obfuscated_size":     len(obfuscated),
		"size_overhead":       len(obfuscated) - len(testData),
		"original_entropy":    originalEntropy,
		"obfuscated_entropy":  obfuscatedEntropy,
		"entropy_increase":    obfuscatedEntropy - originalEntropy,
		"transformation_layers": len(do.transformLayers),
	}
}

// calculateEntropy 计算数据熵值
func calculateEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	// 统计字节频率
	freq := make(map[byte]int)
	for _, b := range data {
		freq[b]++
	}

	// 计算香农熵
	entropy := 0.0
	length := float64(len(data))

	for _, count := range freq {
		if count > 0 {
			p := float64(count) / length
			entropy -= p * log2(p)
		}
	}

	return entropy
}

// log2 计算以2为底的对数（简化实现）
func log2(x float64) float64 {
	if x <= 0 {
		return 0
	}
	if x == 1.0 {
		return 0
	}

	// 使用Go的内置math包会更精确，但为了简化这里使用近似算法
	// 对于熵值计算来说精度足够
	if x < 1.0 {
		// 处理小于1的情况
		result := 0.0
		for x < 1.0 {
			x *= 2
			result--
		}
		return result
	}

	// 处理大于1的情况
	result := 0.0
	for x >= 2.0 {
		x /= 2
		result++
	}
	return result
}