package server

import (
	"io"
	"log"
)

type Handle struct {
	Uri     string
	Desc    string
	Gob     bool
	Process func(uid uint64, reader io.Reader) (any, error)
}

var handles = make([]Handle, 0)

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
