package server

import (
	"fmt"
	"github.com/clong1995/go-encipher/gob"
	"github.com/clong1995/go-encipher/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

func register(mux *http.ServeMux, handle Handle) {
	mux.HandleFunc(handle.Uri, func(w http.ResponseWriter, r *http.Request) {
		var err error
		defer func() {
			_ = r.Body.Close()
			if err != nil {
				var data string
				if handle.Gob {
					//TODO 这里没有处理错误
				} else {
					data = fmt.Sprintf(`{"state":"%s"}`, err.Error())
				}
				http.Error(w, data, http.StatusBadRequest)
			}
		}()

		userId := r.Header.Get("user-id")
		if userId == "" {
			err = fmt.Errorf("user id is empty")
			log.Println(err)
			return
		}

		uid, err := strconv.ParseUint(userId, 10, 64)
		if err != nil {
			log.Println(err)
			return
		}

		result, err := handle.Process(uid, func(req any) (err error) {
			if handle.Gob {
				if err = gob.Decode(r.Body, req); err != nil {
					log.Println(err)
					return
				}
			} else {
				if err = json.Decode(r.Body, req); err != nil {
					log.Println(err)
					return
				}
			}
			return
		})

		if err != nil {
			log.Println(err)
			return
		}

		if handle.Gob {
			_ = gob.Encode(result, w)
		} else {
			_ = json.Encode(&response{
				State:     "OK",
				Data:      result,
				Timestamp: time.Now().Unix(),
			}, w)
		}
	})
}

type response struct {
	State     string `json:"state"`
	Data      any    `json:"data"`
	Timestamp int64  `json:"timestamp"`
}
