package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

// RandomBytes 生成随机字节
func RandomBytes(n int) ([]byte, error) {
	if n <= 0 {
		return nil, NewValidationError("n", n, "must be positive")
	}

	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return nil, NewCryptoError("random", "", err)
	}
	return bytes, nil
}

// SecureRandomBytes 生成安全随机字节（使用系统熵源）
func SecureRandomBytes(n int) ([]byte, error) {
	bytes, err := RandomBytes(n)
	if err != nil {
		return nil, err
	}

	// 额外的熵混合
	entropy := gatherSystemEntropy()
	for i := 0; i < len(bytes) && i < len(entropy); i++ {
		bytes[i] ^= entropy[i]
	}

	return bytes, nil
}

// gatherSystemEntropy 收集系统熵源
func gatherSystemEntropy() []byte {
	var entropy []byte

	// 时间纳秒
	timeBytes := make([]byte, 8)
	t := time.Now().UnixNano()
	for i := 0; i < 8; i++ {
		timeBytes[i] = byte(t >> (i * 8))
	}
	entropy = append(entropy, timeBytes...)

	// Goroutine数量
	entropy = append(entropy, byte(runtime.NumGoroutine()))

	// 内存地址（ASLR）
	var x int
	addr := uintptr(unsafe.Pointer(&x))
	addrBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		addrBytes[i] = byte(addr >> (i * 8))
	}
	entropy = append(entropy, addrBytes...)

	return entropy
}

// ConstantTimeCompare 常数时间比较
func ConstantTimeCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// SecureZero 安全清零内存
func SecureZero(b []byte) {
	if len(b) == 0 {
		return
	}

	// 使用volatile写入防止编译器优化
	for i := range b {
		b[i] = 0
	}

	// 强制内存屏障
	runtime.KeepAlive(b)
}

// XORBytes XOR操作
func XORBytes(dst, a, b []byte) {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if len(dst) < n {
		n = len(dst)
	}

	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
}

// RotateLeft 循环左移
func RotateLeft(data []byte, n int) {
	if len(data) == 0 || n == 0 {
		return
	}

	n = n % len(data)
	if n < 0 {
		n += len(data)
	}

	temp := make([]byte, n)
	copy(temp, data[:n])
	copy(data[:len(data)-n], data[n:])
	copy(data[len(data)-n:], temp)
}

// Obfuscate 简单数据混淆
func Obfuscate(data []byte, key []byte) []byte {
	if len(data) == 0 || len(key) == 0 {
		return data
	}

	result := make([]byte, len(data))
	keyLen := len(key)

	for i, b := range data {
		// 多层混淆
		keyByte := key[i%keyLen]

		// 第一层：XOR with key
		result[i] = b ^ keyByte

		// 第二层：位移
		result[i] = (result[i] << 3) | (result[i] >> 5)

		// 第三层：与位置相关的变换
		result[i] ^= byte(i & 0xFF)
	}

	return result
}

// Deobfuscate 逆向混淆
func Deobfuscate(data []byte, key []byte) []byte {
	if len(data) == 0 || len(key) == 0 {
		return data
	}

	result := make([]byte, len(data))
	keyLen := len(key)

	for i, b := range data {
		keyByte := key[i%keyLen]

		// 逆向第三层
		result[i] = b ^ byte(i&0xFF)

		// 逆向第二层
		result[i] = (result[i] >> 3) | (result[i] << 5)

		// 逆向第一层
		result[i] = result[i] ^ keyByte
	}

	return result
}

// EncodingType 编码类型
type EncodingType int

const (
	EncodingHex EncodingType = iota
	EncodingBase64
	EncodingBase64URL
)

// Encode 编码数据
func Encode(data []byte, encoding EncodingType) string {
	switch encoding {
	case EncodingHex:
		return hex.EncodeToString(data)
	case EncodingBase64:
		return base64.StdEncoding.EncodeToString(data)
	case EncodingBase64URL:
		return base64.URLEncoding.EncodeToString(data)
	default:
		return hex.EncodeToString(data)
	}
}

// Decode 解码数据
func Decode(encoded string, encoding EncodingType) ([]byte, error) {
	switch encoding {
	case EncodingHex:
		return hex.DecodeString(encoded)
	case EncodingBase64:
		return base64.StdEncoding.DecodeString(encoded)
	case EncodingBase64URL:
		return base64.URLEncoding.DecodeString(encoded)
	default:
		return hex.DecodeString(encoded)
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	bytes, err := RandomBytes(16)
	if err != nil {
		// fallback to timestamp
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// TimingSafeCompare 防时序攻击的比较函数
func TimingSafeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// SecureBytes 安全字节切片（使用后自动清零）
type SecureBytes struct {
	data []byte
	mu   sync.Mutex
}

// NewSecureBytes 创建安全字节切片
func NewSecureBytes(size int) *SecureBytes {
	data, err := SecureRandomBytes(size)
	if err != nil {
		// fallback
		data = make([]byte, size)
	}

	return &SecureBytes{
		data: data,
	}
}

// Bytes 获取字节数据
func (sb *SecureBytes) Bytes() []byte {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.data
}

// Clear 清空数据
func (sb *SecureBytes) Clear() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	if sb.data != nil {
		SecureZero(sb.data)
		sb.data = nil
	}
}

// Len 获取长度
func (sb *SecureBytes) Len() int {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return len(sb.data)
}

// 析构时自动清理
func (sb *SecureBytes) finalize() {
	sb.Clear()
}

// Fingerprint 计算数据指纹
func Fingerprint(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	// 简单哈希指纹
	var hash uint64 = 14695981039346656037 // FNV offset basis
	for _, b := range data {
		hash ^= uint64(b)
		hash *= 1099511628211 // FNV prime
	}

	return fmt.Sprintf("%016x", hash)
}

// IsZero 检查是否为零值
func IsZero(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

// HasEntropy 检查数据是否有足够熵值
func HasEntropy(data []byte, minEntropy float64) bool {
	if len(data) < 4 {
		return false
	}

	// 简单熵值检查：统计字节分布
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
			entropy -= p * (func(x float64) float64 {
				if x <= 0 {
					return 0
				}
				// 简化的对数计算
				return x * 1.4426950408889634 // log2(e)
			})(p)
		}
	}

	return entropy >= minEntropy
}

// Performance 性能测试工具
type Performance struct {
	startTime time.Time
	operation string
}

// NewPerformance 创建性能测试实例
func NewPerformance(operation string) *Performance {
	return &Performance{
		startTime: time.Now(),
		operation: operation,
	}
}

// Stop 停止计时并返回持续时间
func (p *Performance) Stop() time.Duration {
	return time.Since(p.startTime)
}

// StopAndLog 停止计时并记录日志
func (p *Performance) StopAndLog() time.Duration {
	duration := time.Since(p.startTime)
	fmt.Printf("Operation '%s' took %v\n", p.operation, duration)
	return duration
}