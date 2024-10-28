package server

import (
	"github.com/clong1995/go-encipher/gob"
	"github.com/clong1995/go-encipher/json"
	"log"
	"net/http"
	"time"
)

func register(mux *http.ServeMux, handle Handle) {
	mux.HandleFunc(handle.Uri, func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			_ = r.Body.Close()
		}()

		var data body
		err := gob.Decode(r.Body, &data)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp := new(response)
		resp.Timestamp = time.Now().Unix()

		result, err := handle.Process(data.Uid, data.Param)
		if err != nil {
			log.Println(err)
			resp.State = err.Error()
			_ = json.Encoder(resp, w)
		} else {
			if handle.Gob {
				_ = gob.Encoder(&result, w)
			} else {
				resp.State = "OK"
				resp.Data = result
				_ = json.Encoder(resp, w)
			}
		}
	})
}

type response struct {
	State     string `json:"state"`
	Data      any    `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

type body struct {
	Uid   uint64
	Param []byte
}
