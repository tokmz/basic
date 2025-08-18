package logger

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockWriteSyncer 模拟WriteSyncer
type mockWriteSyncer struct {
	buf        *bytes.Buffer
	mu         sync.Mutex
	syncCalled bool
	closed     bool
}

func newMockWriteSyncer() *mockWriteSyncer {
	return &mockWriteSyncer{
		buf: &bytes.Buffer{},
	}
}

func (m *mockWriteSyncer) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, fmt.Errorf("writer is closed")
	}
	return m.buf.Write(p)
}

func (m *mockWriteSyncer) Sync() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.syncCalled = true
	return nil
}

func (m *mockWriteSyncer) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockWriteSyncer) String() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buf.String()
}

func (m *mockWriteSyncer) WasSyncCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.syncCalled
}

func TestNewAsyncBuffer(t *testing.T) {
	writer := newMockWriteSyncer()
	config := BufferConfig{
		Size:      1024,
		FlushTime: 100 * time.Millisecond,
	}

	buffer, err := NewAsyncBuffer(config, writer)
	assert.NoError(t, err)
	assert.NotNil(t, buffer)
	defer buffer.Close()

	// 验证配置
	assert.Equal(t, config, buffer.config)
	assert.Equal(t, writer, buffer.writeSyncer)
	assert.NotNil(t, buffer.buffer)
	assert.NotNil(t, buffer.flushCh)
	assert.NotNil(t, buffer.closeCh)
}

func TestAsyncBuffer_Write(t *testing.T) {
	writer := newMockWriteSyncer()
	config := BufferConfig{
		Size:      100,
		FlushTime: 1 * time.Second, // 长间隔，避免自动刷新
	}

	buffer, err := NewAsyncBuffer(config, writer)
	assert.NoError(t, err)
	defer buffer.Close()

	// 写入数据
	data := []byte("test data")
	n, err := buffer.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)

	// 数据应该在缓冲区中，还未写入writer
	assert.Empty(t, writer.String())

	// 手动刷新
	buffer.Flush()
	time.Sleep(10 * time.Millisecond) // 等待异步刷新

	// 验证数据被写入
	assert.Equal(t, string(data), writer.String())
}

func TestAsyncBuffer_WriteAutoFlush(t *testing.T) {
	writer := newMockWriteSyncer()
	config := BufferConfig{
		Size:      10, // 小缓冲区，容易触发自动刷新
		FlushTime: 1 * time.Second,
	}

	buffer, err := NewAsyncBuffer(config, writer)
	assert.NoError(t, err)
	defer buffer.Close()

	// 写入足够的数据触发自动刷新
	for i := 0; i < 15; i++ {
		data := []byte(fmt.Sprintf("data-%d\n", i))
		_, err := buffer.Write(data)
		assert.NoError(t, err)
	}

	// 等待自动刷新
	time.Sleep(50 * time.Millisecond)

	// 验证数据被写入
	assert.NotEmpty(t, writer.String())
}

func TestAsyncBuffer_FlushInterval(t *testing.T) {
	writer := newMockWriteSyncer()
	config := BufferConfig{
		Size:      1024,
		FlushTime: 50 * time.Millisecond, // 短间隔
	}

	buffer, err := NewAsyncBuffer(config, writer)
	assert.NoError(t, err)
	defer buffer.Close()

	// 写入数据
	data := []byte("test interval flush")
	_, err = buffer.Write(data)
	assert.NoError(t, err)

	// 等待定时刷新
	time.Sleep(100 * time.Millisecond)

	// 验证数据被写入
	assert.Equal(t, string(data), writer.String())
}

func TestAsyncBuffer_Sync(t *testing.T) {
	writer := newMockWriteSyncer()
	config := BufferConfig{
		Size:      1024,
		FlushTime: 1 * time.Second,
	}

	buffer, err := NewAsyncBuffer(config, writer)
	assert.NoError(t, err)
	defer buffer.Close()

	// 写入数据
	data := []byte("test sync data")
	_, err = buffer.Write(data)
	assert.NoError(t, err)

	// 同步刷新
	err = buffer.Sync()
	assert.NoError(t, err)

	// 等待异步刷新完成
	time.Sleep(50 * time.Millisecond)

	// 验证数据被写入和同步
	assert.Equal(t, string(data), writer.String())
	assert.True(t, writer.WasSyncCalled())
}

func TestAsyncBuffer_Close(t *testing.T) {
	writer := newMockWriteSyncer()
	config := BufferConfig{
		Size:      1024,
		FlushTime: 1 * time.Second,
	}

	buffer, err := NewAsyncBuffer(config, writer)
	assert.NoError(t, err)

	// 写入数据
	data := []byte("test close data")
	_, err = buffer.Write(data)
	assert.NoError(t, err)

	// 关闭缓冲区
	buffer.Close()

	// 验证数据被刷新
	assert.Equal(t, string(data), writer.String())

	// 验证缓冲区已关闭
	assert.True(t, buffer.isClosed())

	// 再次写入应该返回0字节（不报错，但不写入）
	n, err := buffer.Write([]byte("should not write"))
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestAsyncBuffer_ConcurrentWrites(t *testing.T) {
	writer := newMockWriteSyncer()
	config := BufferConfig{
		Size:      1024,
		FlushTime: 100 * time.Millisecond,
	}

	buffer, err := NewAsyncBuffer(config, writer)
	assert.NoError(t, err)
	defer buffer.Close()

	var wg sync.WaitGroup
	numGoroutines := 10
	numWrites := 100

	// 并发写入
	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range numWrites {
				data := fmt.Appendf(nil, "goroutine-%d-write-%d\n", id, j)
				_, err = buffer.Write(data)
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// 等待所有数据被刷新
	err = buffer.Sync()
	assert.NoError(t, err)

	// 等待异步刷新完成
	time.Sleep(100 * time.Millisecond)

	// 验证所有数据都被写入
	writtenData := writer.String()
	assert.NotEmpty(t, writtenData)

	// 验证写入的数据量是否合理（简化验证）
	lines := strings.Split(strings.TrimSpace(writtenData), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = []string{} // 空字符串情况
	}
	assert.GreaterOrEqual(t, len(lines), numGoroutines) // 至少每个goroutine写入了一些数据
}

func TestNewBufferedWriteSyncer(t *testing.T) {
	writer := newMockWriteSyncer()
	config := BufferConfig{
		Size:      1024,
		FlushTime: 100 * time.Millisecond,
	}

	bws, err := NewBufferedWriteSyncer(writer, config)
	assert.NoError(t, err)
	assert.NotNil(t, bws)
	defer bws.Close()
}

func TestBufferedWriteSyncer_Write(t *testing.T) {
	writer := newMockWriteSyncer()
	config := BufferConfig{
		Size:      1024,
		FlushTime: 1 * time.Second,
	}

	bws, err := NewBufferedWriteSyncer(writer, config)
	assert.NoError(t, err)
	defer bws.Close()

	data := []byte("test buffered write")
	n, err := bws.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
}

func TestBufferedWriteSyncer_Sync(t *testing.T) {
	writer := newMockWriteSyncer()
	config := BufferConfig{
		Size:      1024,
		FlushTime: 1 * time.Second,
	}

	bws, err := NewBufferedWriteSyncer(writer, config)
	assert.NoError(t, err)
	defer bws.Close()

	data := []byte("test sync")
	_, err = bws.Write(data)
	assert.NoError(t, err)

	err = bws.Sync()
	assert.NoError(t, err)

	// 等待异步刷新
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, string(data), writer.String())
	assert.True(t, writer.WasSyncCalled())
}

func TestBufferedWriteSyncer_Close(t *testing.T) {
	writer := newMockWriteSyncer()
	config := BufferConfig{
		Size:      1024,
		FlushTime: 1 * time.Second,
	}

	bws, err := NewBufferedWriteSyncer(writer, config)
	assert.NoError(t, err)

	data := []byte("test close")
	_, err = bws.Write(data)
	assert.NoError(t, err)

	bws.Close()

	// 验证数据被刷新
	assert.Equal(t, string(data), writer.String())
}
