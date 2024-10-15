package server

import (
	"encoding/gob"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func register(mux *http.ServeMux, handle Handle) {
	mux.HandleFunc(handle.Uri, func(w http.ResponseWriter, r *http.Request) {
		//var uid string
		var err error
		//var reqBytes []byte
		var resBytes []byte

		defer func() {
			_ = r.Body.Close()
		}()

		dec := gob.NewDecoder(r.Body)
		var data body
		err = dec.Decode(&data)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		//执行处理函数
		result, err := handle.Process(data.Uid, data.Param, data.File)
		if err != nil {
			log.Println(err)
			resBytes, _ = json.Marshal(response{
				err.Error(),
				nil,
				0,
			})
		} else {
			resBytes, _ = json.Marshal(response{
				"OK",
				result,
				time.Now().Unix(),
			})
		}
		//状态
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
	File  []byte
}
