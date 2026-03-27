package server

import (
	"log"
)

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
