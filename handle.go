package server

import (
	"log"
)

var handles = make([]Handle, 0)

type Handle struct {
	Uri     string
	Desc    string
	Gob     bool
	Process func(uid int64, param Param) (any, error)
}

type Param func(req any) error

func (h Handle) Register() {
	if h.Uri == "" || h.Desc == "" || h.Process == nil {
		log.Fatalln("uri or desc or process is empty")
	}

	for _, handle := range handles {
		if handle.Uri == h.Uri {
			log.Fatalf("'%s' is redeclared", h.Uri)
		}
	}

	handles = append(handles, h)
}
