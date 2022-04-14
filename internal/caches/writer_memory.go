package caches

import (
	"errors"
	"github.com/cespare/xxhash"
	"sync"
	"time"
)

type MemoryWriter struct {
	storage *MemoryStorage

	key        string
	expiredAt  int64
	headerSize int64
	bodySize   int64
	status     int
	isDirty    bool
	maxSize    int64

	hash    uint64
	item    *MemoryItem
	endFunc func()
	once    sync.Once
}

func NewMemoryWriter(memoryStorage *MemoryStorage, key string, expiredAt int64, status int, isDirty bool, maxSize int64, endFunc func()) *MemoryWriter {
	w := &MemoryWriter{
		storage:   memoryStorage,
		key:       key,
		expiredAt: expiredAt,
		item: &MemoryItem{
			ExpiresAt:  expiredAt,
			ModifiedAt: time.Now().Unix(),
			Status:     status,
		},
		status:  status,
		isDirty: isDirty,
		maxSize: maxSize,
		endFunc: endFunc,
	}
	w.hash = w.calculateHash(key)

	return w
}

// WriteHeader 写入数据
func (this *MemoryWriter) WriteHeader(data []byte) (n int, err error) {
	this.headerSize += int64(len(data))
	this.item.HeaderValue = append(this.item.HeaderValue, data...)
	return len(data), nil
}

// Write 写入数据
func (this *MemoryWriter) Write(data []byte) (n int, err error) {
	this.bodySize += int64(len(data))
	this.item.BodyValue = append(this.item.BodyValue, data...)

	// 检查尺寸
	if this.maxSize > 0 && this.bodySize > this.maxSize {
		err = ErrEntityTooLarge
		this.storage.IgnoreKey(this.key)
		return len(data), err
	}

	return len(data), nil
}

// WriteAt 在指定位置写入数据
func (this *MemoryWriter) WriteAt(offset int64, b []byte) error {
	_ = b
	_ = offset
	return errors.New("not supported")
}

// HeaderSize 数据尺寸
func (this *MemoryWriter) HeaderSize() int64 {
	return this.headerSize
}

// BodySize 主体内容尺寸
func (this *MemoryWriter) BodySize() int64 {
	return this.bodySize
}

// Close 关闭
func (this *MemoryWriter) Close() error {
	// 需要在Locker之外
	defer this.once.Do(func() {
		this.endFunc()
	})

	if this.item == nil {
		return nil
	}

	this.storage.locker.Lock()
	this.item.IsDone = true
	this.storage.valuesMap[this.hash] = this.item
	if this.isDirty {
		if this.storage.parentStorage != nil {
			select {
			case this.storage.dirtyChan <- this.key:
			default:

			}
		}
	}
	this.storage.locker.Unlock()

	return nil
}

// Discard 丢弃
func (this *MemoryWriter) Discard() error {
	// 需要在Locker之外
	defer this.once.Do(func() {
		this.endFunc()
	})

	this.storage.locker.Lock()
	delete(this.storage.valuesMap, this.hash)
	this.storage.locker.Unlock()
	return nil
}

// Key 获取Key
func (this *MemoryWriter) Key() string {
	return this.key
}

// ExpiredAt 过期时间
func (this *MemoryWriter) ExpiredAt() int64 {
	return this.expiredAt
}

// ItemType 内容类型
func (this *MemoryWriter) ItemType() ItemType {
	return ItemTypeMemory
}

// 计算Key Hash
func (this *MemoryWriter) calculateHash(key string) uint64 {
	return xxhash.Sum64String(key)
}
