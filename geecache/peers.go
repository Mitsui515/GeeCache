package geecache

import pb "geecache/geecachepb"

//在这里，抽象出 2 个接口，PeerPicker 的 PickPeer() 方法用于根据传入的 key 选择相应节点 PeerGetter。
//接口 PeerGetter 的 Get() 方法用于从对应 group 查找缓存值。PeerGetter 就对应于上述流程中相应远程节点的客户端。

// PeerPicker is the interface that must be implemented to locate
// the peer that owns a specific key.
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter is the interface that must be implemented by a peer.
type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error
}
