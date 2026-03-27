package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/clong1995/go-ansi-color"
	"github.com/clong1995/go-config"
	"github.com/clong1995/go-db-kv"
	"github.com/clong1995/go-encipher/gob"
	"github.com/clong1995/go-encipher/json"
)

var prefix = "server"

func init() {
	machineId, exists := config.Value[int]("MACHINE ID")
	if !exists || machineId == 0 {
		log.Fatalln("MACHINE not found")
	}

	addr := fmt.Sprintf(":90%d", machineId)

	http.HandleFunc("/", handler)

	httpserver = &http.Server{
		Addr:         addr,
		Handler:      nil,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		err := httpserver.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			pcolor.PrintFatal(prefix, err.Error())
			return
		}
		pcolor.PrintSucc(prefix, "server exited!")
	}()
	pcolor.PrintSucc(prefix, "listening %s", addr)
}

var httpserver *http.Server
var hs = newHandles()
var reg = regexp.MustCompile(`"t":\d+,"a":"[^"]+",?`)

func Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpserver.Shutdown(ctx); err != nil {
		pcolor.PrintError(prefix, err.Error())
	}
	kv.Close()
}

func handler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	handle, ok := hs.get(r.URL.Path)
	if !ok {
		err := fmt.Errorf("handler not found:%s", r.URL.Path)
		log.Println(err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	userId := r.Header.Get("user-id")
	if userId == "" {
		err := errors.New("user id is empty")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	uid, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if handle.Gob {
		w.Header().Set("Content-Type", "application/octet-stream")
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	if handle.Cache != "" {
		var all []byte

		if all, err = io.ReadAll(r.Body); err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// result []byte, err error
		process := func() ([]byte, error) {
			res, processErr := handle.Process(uid, func(req any) error {
				buf := bytes.NewBuffer(all)
				if handle.Gob {
					if paramErr := gob.Decode(buf, req); paramErr != nil {
						log.Printf("%+v\n", paramErr)
						return paramErr
					}
				} else {
					if paramErr := json.Decode(buf, req); paramErr != nil {
						log.Printf("%+v\n", paramErr)
						return paramErr
					}
				}
				return nil
			})
			if processErr != nil {
				log.Printf("%+v\n", processErr)
				return nil, processErr
			}

			buf := new(bytes.Buffer)
			if handle.Gob {
				if res != nil {
					if paramErr := gob.Encode(res, buf); paramErr != nil {
						log.Printf("%+v\n", paramErr)
						return nil, paramErr
					}
				}
			} else {
				if paramErr := json.Encode(&response{
					State:     "OK",
					Data:      res,
					Timestamp: time.Now().Unix(),
				}, buf); paramErr != nil {
					log.Printf("%+v\n", paramErr)
					return nil, paramErr
				}
			}

			return buf.Bytes(), nil
		}

		param := reg.ReplaceAll(all, []byte(""))
		machineId, _ := config.Value[int]("MACHINE ID")
		key := kv.HashKey(fmt.Sprintf(
			"%s%s%s",
			machineId,
			handle.Uri,
			param,
		))

		var cacheType string
		var ttl int64 = 15000
		arr := strings.Split(handle.Cache, ":")
		cacheType = arr[0]
		if len(arr) == 2 {
			i, atoiErr := strconv.Atoi(arr[1])
			if atoiErr != nil {
				log.Println(atoiErr)
				http.Error(w, atoiErr.Error(), http.StatusInternalServerError)
				return
			}
			ttl = int64(i)
		} else if len(arr) != 1 {
			err = errors.New("cache err")
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var storage []byte

		switch cacheType {
		case "perm":
			if storage, err = kv.Storage[[]byte](key, process); err != nil {
				log.Printf("%+v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case "ttl":
			if storage, err = kv.Storage[[]byte](key, process, ttl); err != nil {
				log.Printf("%+v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case "ttl-dsc":
			if storage, err = kv.Storage[[]byte](key, process, ttl, 1); err != nil {
				log.Printf("%+v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		default:
			err = fmt.Errorf("unknown cache type: %s", cacheType)
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, _ = w.Write(storage)
	} else {
		var result any
		result, err = handle.Process(uid, func(req any) error {
			if handle.Gob {
				if decodeErr := gob.Decode(r.Body, req); decodeErr != nil {
					log.Printf("%+v\n", decodeErr)
					return decodeErr
				}
				return nil
			}
			if decodeErr := json.Decode(r.Body, req); decodeErr != nil {
				log.Printf("%+v\n", decodeErr)
				return decodeErr
			}
			return nil
		})

		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if handle.Gob {
			if result != nil {
				if err = gob.Encode(result, w); err != nil {
					log.Printf("%+v\n", err)
				}
			}
		} else {
			if err = json.Encode(&response{
				State:     "OK",
				Data:      result,
				Timestamp: time.Now().Unix(),
			}, w); err != nil {
				log.Printf("%+v\n", err)
			}
		}
	}
}

type response struct {
	State     string `json:"state"`
	Data      any    `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

type handles struct {
	mu   sync.RWMutex
	data map[string]Handle
}

func (h *handles) set(key string, value Handle) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.data[key] = value
}

func (h *handles) get(key string) (Handle, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	value, ok := h.data[key]
	return value, ok
}

func newHandles() *handles {
	return &handles{
		data: make(map[string]Handle),
	}
}
