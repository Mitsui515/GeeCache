# GeeCache

GeeCache is a simple cache library inspired by groupcache.

## Project Overview

GeeCache is structured as follows:

```
plaintextCopy codeGee-Cache/
│
├── geecache/
│   ├── consistenthash
│   │   ├── consistenthash.go
│   │   └── consistenthash_test.go
│   ├── lru
│   │   ├── lru.go
│   │   └── lru_test.go
│   ├── singleflight
│   │   ├── singleflight.go
│   │   └── singleflight_test.go
│   ├── byteview.go
│   ├── cache.go
│   ├── geecache.go
│   ├── geecache_test.go
│   ├── http.go
│   ├── peer.go
│   └── go.mod
├── main.go
├── go.mod
├── go.sum
├── run.sh
├── README_zh.md
└── README.md
```