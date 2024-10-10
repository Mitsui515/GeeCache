package lru

import (
	"reflect"
	"testing"
)

type String string

func (d String) Len() int {
	return len(d)
}

func TestGet(t *testing.T) {
	lru := New(int64(0), nil, 60)
	//在这个特定的上下文中，int64(0) 作为参数传递给 New 函数，用于指定 LRU 缓存的最大存储容量。
	//在这里，将其设置为 0 表示缓存的最大容量为零，即没有存储空间，因此不会保存任何键值对。
	//这可以用于创建一个非常小的缓存或用于特定的测试场景，其中不需要实际存储数据。
	lru.Add("key1", String("1234"), 60)
	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "1234" {
		t.Fatalf("cache hit key1-1234 failed")
	}
	if _, ok := lru.Get("key2"); ok {
		t.Fatalf("cache miss key2 failed")
	}
}

func TestRemoveOldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := "value1", "value2", "v3"
	cap := len(k1 + k2 + v1 + v2)
	lru := New(int64(cap), nil, 30)
	lru.Add(k1, String(v1), 30)
	lru.Add(k2, String(v2), 30)
	lru.Add(k3, String(v3), 30)

	if _, ok := lru.Get("key1"); ok || lru.Len() != 2 {
		t.Fatalf("RemoveOldest key1 failed")
	}
}

func TestOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}
	lru := New(int64(10), callback, 30)
	lru.Add("key1", String("123456"), 30)
	lru.Add("k2", String("k2"), 30)
	lru.Add("k3", String("k3"), 30)
	lru.Add("k4", String("k4"), 30)

	expect := []string{"key1", "k2"}

	t.Logf("Evicted keys: %v", keys)

	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %s", expect)
	}
}

func TestAdd(t *testing.T) {
	lru := New(int64(0), nil, 60)
	lru.Add("key", String("1"), 60)
	lru.Add("key", String("111"), 60)

	if lru.nbytes != int64(len("key")+len("111")) {
		t.Fatal("expected 6 but got", lru.nbytes)
	}
}
