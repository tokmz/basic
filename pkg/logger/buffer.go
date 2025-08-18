package logger

import (
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap/zapcore"
)

// 字节切片对象池，减少内存分配
var byteSlicePool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 1024)
	},
}

// getByteSlice 从对象池获取字节切片
func getByteSlice(size int) []byte {
	slice := byteSlicePool.Get().([]byte)
	if cap(slice) < size {
		// 如果容量不够，创建新的切片
		return make([]byte, size)
	}
	return slice[:size]
}

// putByteSlice 归还字节切片到对象池
func putByteSlice(slice []byte) {
	if cap(slice) > 4096 {
		// 避免池中对象过大
		return
	}
	byteSlicePool.Put(slice[:0])
}

// AsyncBuffer 异步缓冲器
type AsyncBuffer struct {
	config      BufferConfig
	writeSyncer zapcore.WriteSyncer
	buffer      [][]byte
	bufferMu    sync.Mutex
	flushCh     chan struct{}
	closeCh     chan struct{}
	closed      int32
	wg          sync.WaitGroup
}

// NewAsyncBuffer 创建新的异步缓冲器
func NewAsyncBuffer(config BufferConfig, writeSyncer zapcore.WriteSyncer) (*AsyncBuffer, error) {
	buffer := &AsyncBuffer{
		config:      config,
		writeSyncer: writeSyncer,
		buffer:      make([][]byte, 0, config.Size),
		flushCh:     make(chan struct{}, 1),
		closeCh:     make(chan struct{}),
	}

	// 启动后台刷新goroutine
	buffer.wg.Add(1)
	go buffer.flushLoop()

	return buffer, nil
}

// Write 写入数据到缓冲区
func (ab *AsyncBuffer) Write(p []byte) (n int, err error) {
	if ab.isClosed() {
		return 0, nil
	}

	// 优化：使用对象池减少内存分配
	data := getByteSlice(len(p))
	copy(data, p)

	ab.bufferMu.Lock()
	ab.buffer = append(ab.buffer, data)
	needFlush := len(ab.buffer) >= ab.config.Size
	ab.bufferMu.Unlock()

	// 如果缓冲区满了，触发刷新
	if needFlush {
		ab.triggerFlush()
	}

	return len(p), nil
}

// Sync 同步刷新
func (ab *AsyncBuffer) Sync() error {
	ab.Flush()
	return ab.writeSyncer.Sync()
}

// Flush 刷新缓冲区
func (ab *AsyncBuffer) Flush() {
	if ab.isClosed() {
		return
	}
	ab.triggerFlush()
}

// Close 关闭缓冲器
func (ab *AsyncBuffer) Close() {
	if !atomic.CompareAndSwapInt32(&ab.closed, 0, 1) {
		return
	}

	close(ab.closeCh)
	ab.wg.Wait()

	// 最后刷新一次
	ab.flushBuffer()
}

// triggerFlush 触发刷新
func (ab *AsyncBuffer) triggerFlush() {
	select {
	case ab.flushCh <- struct{}{}:
	default:
		// 如果通道已满，说明已经有刷新请求在等待
	}
}

// flushLoop 刷新循环
func (ab *AsyncBuffer) flushLoop() {
	defer ab.wg.Done()

	ticker := time.NewTicker(ab.config.FlushTime)
	defer ticker.Stop()

	for {
		select {
		case <-ab.closeCh:
			return
		case <-ab.flushCh:
			ab.flushBuffer()
		case <-ticker.C:
			ab.flushBuffer()
		}
	}
}

// flushBuffer 刷新缓冲区数据
func (ab *AsyncBuffer) flushBuffer() {
	ab.bufferMu.Lock()
	if len(ab.buffer) == 0 {
		ab.bufferMu.Unlock()
		return
	}

	// 交换缓冲区
	toFlush := ab.buffer
	ab.buffer = make([][]byte, 0, ab.config.Size)
	ab.bufferMu.Unlock()

	// 批量写入
	for _, data := range toFlush {
		if _, err := ab.writeSyncer.Write(data); err != nil {
			// 这里可以添加错误处理逻辑
			// 比如记录到错误日志或重试
		}
		// 优化：归还字节切片到对象池
		putByteSlice(data)
	}

	// 同步写入
	if err := ab.writeSyncer.Sync(); err != nil {
		// 错误处理
	}
}

// isClosed 检查是否已关闭
func (ab *AsyncBuffer) isClosed() bool {
	return atomic.LoadInt32(&ab.closed) == 1
}

// BufferedWriteSyncer 带缓冲的写入同步器
type BufferedWriteSyncer struct {
	zapcore.WriteSyncer
	buffer *AsyncBuffer
}

// NewBufferedWriteSyncer 创建带缓冲的写入同步器
func NewBufferedWriteSyncer(ws zapcore.WriteSyncer, config BufferConfig) (*BufferedWriteSyncer, error) {
	buffer, err := NewAsyncBuffer(config, ws)
	if err != nil {
		return nil, err
	}

	return &BufferedWriteSyncer{
		WriteSyncer: buffer,
		buffer:      buffer,
	}, nil
}

// Close 关闭缓冲写入器
func (bws *BufferedWriteSyncer) Close() {
	bws.buffer.Close()
}
