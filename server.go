package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/clong1995/go-config"
	"github.com/clong1995/go-encipher/gob"
	"github.com/clong1995/go-encipher/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func Listen() {
	mux := http.NewServeMux()

	addr := ":90" + config.Value("MACHINE ID")
	if addr == "" {
		log.Fatalln("ADDR not found")
	}

	for _, handle := range handles {
		register(mux, handle)
	}

	httpserver := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		_ = <-stop
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpserver.Shutdown(ctx); err != nil {
			log.Println(err)
		}
	}()

	fmt.Printf("[http] listening %s\n", addr)
	err := httpserver.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalln(err)
		return
	}

	fmt.Println("[http] server exited!")
}

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
