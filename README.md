# GeeCache

## Overview

GeeCache is a distributed cache library based on Go language, inspired by groupcache. designed to enhance the performance of applications by providing a scalable and efficient caching mechanism. It incorporates features such as distributed nodes, HTTP communication, prevention of cache breakdown, and efficient data serialization using Protobuf.

## Key Features

- **Distributed Nodes**: Allows registration of nodes and utilizes consistent hashing for distributed caching.
- **HTTP Communication**: Implements an HTTP client for seamless communication between cache nodes.
- **Preventing Cache Breakdown**: Addresses cache breakdown using the `singleflight` package to control multiple requests for the same key.
- **Protobuf for Communication**: Enhances communication efficiency with binary serialization using Protobuf.

## Project Overview

GeeCache is structured as follows:

```
Gee-Cache/
│
├── geecache/
│   ├── consistenthash
│   │   ├── consistenthash.go
│   │   └── consistenthash_test.go
│   ├── geecachepb
│   │   ├── geecachepb.pb.go
│   │   └── geecachepb.proto
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
│   ├── go.mod
│   └── go.sum
├── main.go
├── go.mod
├── go.sum
├── run.sh
├── README_zh.md
└── README.md
```

## Installation

Install GeeCache using the `go get` command:

```shell
go get -u github.com/Mitsui515/GeeCache/geecache
```

## Usage Examples

### Local cache

To use local cache, you need to create a `Group` first, specifying the cache name and the callback function. The callback function is used to get data from the data source and add it to the cache when the cache is missed.

```go
import (
	"fmt"
	"github.com/Mitsui515/GeeCache/geecache"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *geecache.Group {
	return geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			fmt.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

func main() {
	g := createGroup()
	fmt.Println(g.Get("Tom"))
	fmt.Println(g.Get("Tom"))
}
```

Output:

```
[SlowDB] search key Tom
630 <nil>
630 <nil>
```

You can see that the first time you get `Tom`’s score, the callback function is called and the data is queried from the database; the second time, the data is returned directly from the cache, without querying the database again.

### Distributed cache

To use distributed cache, you need to create an `HTTPPool`, which is the core component for node-to-node communication. `HTTPPool` implements the `PeerPicker` interface, which is used to select nodes based on key, and also implements the `PeerGetter` interface, which is used to get cache values from remote nodes.

```go
import (
	"fmt"
	"github.com/Mitsui515/GeeCache/geecache"
	"log"
	"net/http"
)

func startCacheServer(addr string, addrs []string, group *geecache.Group) {
	peers := geecache.NewHTTPPool(addr)
	peers.Set(addrs...)
	group.RegisterPeers(peers)
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

func startAPIServer(apiAddr string, group *geecache.Group) {
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		view, err := group.Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(view.ByteSlice())
	})
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	group := createGroup()
	if api {
		go startAPIServer(apiAddr, group)
	}
	startCacheServer(addrMap[port], addrs, group)
}
```

In the command line, start three cache servers and one API server separately:

```shell
go run main.go -port=8001
go run main.go -port=8002
go run main.go -port=8003
go run main.go -api=1 -port=9999
```

In the browser, visit `http://localhost:9999/api?kay=Tom`, you can see that it returns `Tom`’s score `630`. In the cache server’s log, you can see the node-to-node communication and cache hit situation.