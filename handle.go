package server

import (
	"log"
	"sync"
)

var hs = newHandles()

type Handle struct {
	Uri     string
	Desc    string
	Gob     bool
	Cache   string //perm:永久存储。ttl:自动过期和续期（只要被访问就继续续期）。ttl-dsc:自动过期(当新产生的时会有过期时间，被访问不会续期)
	Process func(uid int64, param Param) (any, error)
}

type Param func(req any) error

func (h Handle) Register() {
	if h.Uri == "" || h.Desc == "" || h.Process == nil {
		log.Fatalln("uri or desc or process is empty")
	}

	_, ok := hs.get(h.Uri)
	if ok {
		log.Fatalf("%s is redeclared \n", h.Uri)
	}

	hs.set(h.Uri, h)
}

type handles struct {
	mu   sync.RWMutex
	data map[string]Handle
}

func (h *handles) set(key string, value Handle) {
	h.mu.Lock()         // 获取写锁
	defer h.mu.Unlock() // 确保锁被释放
	h.data[key] = value
}

func (h *handles) get(key string) (value Handle, ok bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	value, ok = h.data[key]
	return
}

func newHandles() *handles {
	return &handles{
		data: make(map[string]Handle),
	}
}
