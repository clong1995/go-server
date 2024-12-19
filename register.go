package server

import (
	"errors"
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
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
		}()

		userId := r.Header.Get("user-id")
		if userId == "" {
			err = errors.New("user id is empty")
			log.Println(err)
			return
		}

		uid, err := strconv.ParseInt(userId, 10, 64)
		if err != nil {
			log.Fatalln(err)
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
			if result == nil {
				return
			}
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
