package lru

import (
	"container/list"
	"log"
	"math/rand"
	"time"
)

/*
Cache 定义了一个结构体，用来实现lru缓存淘汰算法
maxBytes：最大存储容量
nBytes：已占用的容量
ll：直接使用 Go 语言标准库实现的双向链表list.List，双向链表常用于维护缓存中各个数据的访问顺序，以便在淘汰数据时能够方便地找到最近最少使用的数据。
cache：map,键是字符串，值是双向链表中对应节点的指针
OnEvicted：是某条记录被移除时的回调函数，可以为 nil
defaultTTL：记录在缓存中的默认过期时间

增加了键的移除操作后，需要考虑的⼀个问题是：移除了某个键，每个节点中的热点缓存中如果有该键，
也需要删除，因此删除操作需要通知所有节点（etcd实现）。
另外对于缓存来说，超时淘汰有时候也是必要的，因此在缓存中对于每个value，都设计了过期时间。
*/
type Cache struct {
	maxBytes   int64
	nbytes     int64
	ll         *list.List
	cache      map[string]*list.Element
	OnEvicted  func(key string, value Value) // optional and executed when an entry is purged.
	defaultTTL time.Duration
}

// entry 键值对是双向链表节点的数据类型，在链表中仍保存每个值对应的 key 的好处在于，淘汰队首节点时，需要用 key 从字典中删除对应的映射。
type entry struct {
	key    string
	value  Value
	expire time.Time
}

// Value uses Len to count how many bytes it takes
type Value interface {
	Len() int
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value), defaultTTL time.Duration) *Cache {
	return &Cache{
		maxBytes:   maxBytes,
		ll:         list.New(),
		cache:      make(map[string]*list.Element),
		OnEvicted:  onEvicted,
		defaultTTL: defaultTTL,
	}
}

// Get 函数用于根据键获取缓存中的值。
// 如果键存在，则将对应的节点移动到链表的最前面（表示最近使用），并返回对应的值和 true；
// 如果键不存在或者键已经过期，则返回零值和 false。
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		kv := ele.Value.(*entry)
		if kv.expire.Before(time.Now()) {
			c.RemoveElement(ele)
			log.Printf("The Cache key—%s has expired", key)
			return nil, false
		}
		c.ll.MoveToFront(ele)
		return kv.value, true
	}
	return
}

// RemoveOldest 函数找到最久未使用且已过期的缓存项，然后将其从缓存中移除。
func (c *Cache) RemoveOldest() {
	for ele := c.ll.Back(); ele != nil; ele = ele.Prev() {
		kv := ele.Value.(*entry)
		if kv.expire.Before(time.Now()) {
			c.RemoveElement(ele)
			break
		}
	}
}

// Add adds a value to the cache
func (c *Cache) Add(key string, value Value, ttl time.Duration) {
	expireTime := time.Now().Add(ttl + time.Duration(rand.Intn(60))*time.Second)
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
		// 更新过期时间时，判断是否应该保留原本的过期时间
		if kv.expire.Before(expireTime) {
			kv.expire = expireTime
		}
	} else {
		ele := c.ll.PushFront(&entry{key, value, expireTime})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
	// 如果 maxBytes 的值为 0，表示没有限制缓存的总大小，即不限制缓存的内存使用量。
	//在这种情况下，不需要执行缓存大小控制的相关逻辑，可以直接添加或更新缓存项。
	//因此，不需要在 Add 方法中执行删除最旧的缓存项 (RemoveOldest) 的操作。
}

// Len 方法返回当前缓存中的记录数量
func (c *Cache) Len() int {
	return c.ll.Len()
}

// RemoveElement 函数用于删除某个节点
func (c *Cache) RemoveElement(ele *list.Element) {
	c.ll.Remove(ele)
	kv := ele.Value.(*entry)
	delete(c.cache, kv.key)                                //删除key-节点这对映射
	c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len()) //重新计算已用容量
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value) //调用对应的回调函数
	}
}
