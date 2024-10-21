package server

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func register(mux *http.ServeMux, handle Handle) {
	mux.HandleFunc(handle.Uri, func(w http.ResponseWriter, r *http.Request) {
		var err error
		var resBytes []byte

		defer func() {
			_ = r.Body.Close()
		}()

		var data body
		if err = gob.NewDecoder(r.Body).Decode(&data); err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		result, err := handle.Process(data.Uid, data.Param)
		if err != nil {
			log.Println(err)
			resBytes, _ = json.Marshal(response{
				err.Error(),
				nil,
				0,
			})
		} else {
			if handle.Gob {
				var buffer bytes.Buffer
				if err = gob.NewEncoder(&buffer).Encode(result); err != nil {
					log.Println(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				resBytes = buffer.Bytes()
			} else {
				resBytes, _ = json.Marshal(response{
					"OK",
					result,
					time.Now().Unix(),
				})
			}
		}

		w.WriteHeader(http.StatusOK)

		//写出结果
		if _, err = w.Write(resBytes); err != nil {
			log.Println(err)
			return
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

//
