package geecache

import (
	"fmt"
	pb "geecache/geecachepb"
	"geecache/singleflight"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultHotCacheRatio      = 8
	defaultMaxMinuteRemoteQPS = 10
)

// A Group is a cache namespace and associated data loaded spread over
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	hotCache  cache
	peers     PeerPicker
	// use singleflight.Group tp make sure that
	// each key is only fetched once
	loader *singleflight.Group
	keys   map[string]*KeyStats
}

type KeyStats struct {
	firstGetTime time.Time
	remoteCnt    AtomicInt
}

type AtomicInt int64

func (i *AtomicInt) Add(n int64) {
	atomic.AddInt64((*int64)(i), n)
}

func (i *AtomicInt) Get() int64 {
	return atomic.LoadInt64((*int64)(i))
}

// A Getter loads data for a key
type Getter interface {
	Get(key string) ([]byte, error)
}

// A GetterFunc implements Getter witha function
type GetterFunc func(key string) ([]byte, error)

// Get implements Getter inteface function
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup creates a new instance of Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		hotCache:  cache{cacheBytes: cacheBytes / defaultHotCacheRatio},
		loader:    &singleflight.Group{},
		keys:      make(map[string]*KeyStats),
	}
	groups[name] = g
	return g
}

// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get value for a key from cache
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	if v, ok := g.hotCache.get(key); ok {
		log.Println("[GeeCache] hot cache hit")
		return v, nil
	}
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] main cache hit")
		return v, nil
	}
	return g.load(key)
}

func (g *Group) load(key string) (value ByteView, err error) {
	// each key is only fetched once (either locally or remotely)
	// regardless of the number of concurrent callers.
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})

	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}

	value := ByteView{b: res.Value}

	g.updateKeyStats(key, value)

	return value, nil
}

func (g *Group) updateKeyStats(key string, value ByteView) {
	// mu.Lock()
	// defer mu.Unlock()
	// 更新键的访问统计信息
	if stat, ok := g.keys[key]; ok {
		stat.remoteCnt.Add(1)
		interval := float64(time.Now().Unix()-stat.firstGetTime.Unix()) / 60
		qps := stat.remoteCnt.Get() / int64(math.Max(1, math.Round(interval)))
		// 如果 QPS 超过阈值，将数据添加到热点缓存
		if qps >= defaultMaxMinuteRemoteQPS {
			g.populateHotCache(key, value)
			mu.Lock()
			delete(g.keys, key)
			mu.Unlock()
		}
	} else {
		// 首次访问，初始化统计信息
		g.keys[key] = &KeyStats{
			firstGetTime: time.Now(),
			remoteCnt:    1,
		}
	}
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

func (g *Group) populateHotCache(key string, value ByteView) {
	g.hotCache.add(key, value)
}

// RegisterPeers registers a PeerPicker for choosing remote peer.
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}
