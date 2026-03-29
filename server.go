package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	pcolor "github.com/clong1995/go-ansi-color"
	"github.com/clong1995/go-config"
	kv "github.com/clong1995/go-db-kv"
	"github.com/clong1995/go-encipher/gob"
	"github.com/clong1995/go-encipher/json"
)

var (
	prefix     = "server"
	httpserver *http.Server
	hs         = newHandles()
	reg        = regexp.MustCompile(`"t":\d+,"a":"[^"]+",?`)
	machineID  int
)

func start() {
	var exists bool
	machineID, exists = config.Value[int]("MACHINE ID")
	if !exists || machineID == 0 {
		pcolor.PrintFatal(prefix, "MACHINE not found")
	}

	addr := fmt.Sprintf(":90%d", machineID)

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

func Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpserver.Shutdown(ctx); err != nil {
		pcolor.PrintError(prefix, err)
	}
	kv.Close()
}

func handler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	handle, ok := hs.get(r.URL.Path)
	if !ok {
		err := fmt.Errorf("handler not found:%s", r.URL.Path)
		pcolor.PrintError(prefix, err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	userId := r.Header.Get("user-id")
	if userId == "" {
		err := fmt.Errorf("user id is empty")
		pcolor.PrintError(prefix, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	uid, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		pcolor.PrintError(prefix, err)
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
			pcolor.PrintError(prefix, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// result []byte, err error
		process := func() ([]byte, error) {
			res, processErr := handle.Process(uid, func(req any) error {
				buf := bytes.NewBuffer(all)
				if handle.Gob {
					if paramErr := gob.Decode(buf, req); paramErr != nil {
						pcolor.PrintErr(prefix, "%+v\n", paramErr)
						return paramErr
					}
				} else {
					if paramErr := json.Decode(buf, req); paramErr != nil {
						pcolor.PrintErr(prefix, "%+v\n", paramErr)
						return paramErr
					}
				}
				return nil
			})
			if processErr != nil {
				pcolor.PrintErr(prefix, "%+v\n", processErr)
				return nil, processErr
			}

			buf := new(bytes.Buffer)
			if handle.Gob {
				if res != nil {
					if paramErr := gob.Encode(res, buf); paramErr != nil {
						pcolor.PrintErr(prefix, "%+v\n", paramErr)
						return nil, paramErr
					}
				}
			} else {
				if paramErr := json.Encode(&response{
					State:     "OK",
					Data:      res,
					Timestamp: time.Now().Unix(),
				}, buf); paramErr != nil {
					pcolor.PrintErr(prefix, "%+v\n", paramErr)
					return nil, paramErr
				}
			}

			return buf.Bytes(), nil
		}

		param := reg.ReplaceAll(all, []byte(""))
		key := kv.HashKey(fmt.Sprintf(
			"%d%s%s",
			machineID,
			handle.Uri,
			param,
		))

		var cacheType string
		var ttl int64 = 15000
		arr := strings.Split(handle.Cache, ":")
		cacheType = arr[0]
		if len(arr) == 2 {
			var i int
			if i, err = strconv.Atoi(arr[1]); err != nil {
				pcolor.PrintError(prefix, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			ttl = int64(i)
		} else if len(arr) != 1 {
			err = fmt.Errorf("cache err")
			pcolor.PrintError(prefix, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var storage []byte

		switch cacheType {
		case "perm":
			if storage, err = kv.Storage[[]byte](key, process); err != nil {
				pcolor.PrintErr(prefix, "%+v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case "ttl":
			if storage, err = kv.Storage[[]byte](key, process, ttl); err != nil {
				pcolor.PrintErr(prefix, "%+v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case "ttl-dsc":
			if storage, err = kv.Storage[[]byte](key, process, ttl, 1); err != nil {
				pcolor.PrintErr(prefix, "%+v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		default:
			err = fmt.Errorf("unknown cache type: %s", cacheType)
			pcolor.PrintError(prefix, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, _ = w.Write(storage)
	} else {
		var result any
		result, err = handle.Process(uid, func(req any) error {
			if handle.Gob {
				if decodeErr := gob.Decode(r.Body, req); decodeErr != nil {
					pcolor.PrintErr(prefix, "%+v\n", decodeErr)
					return decodeErr
				}
				return nil
			}
			if decodeErr := json.Decode(r.Body, req); decodeErr != nil {
				pcolor.PrintErr(prefix, "%+v\n", decodeErr)
				return decodeErr
			}
			return nil
		})

		if err != nil {
			pcolor.PrintError(prefix, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if handle.Gob {
			if result != nil {
				if err = gob.Encode(result, w); err != nil {
					pcolor.PrintErr(prefix, "%+v\n", err)
				}
			}
		} else {
			if err = json.Encode(&response{
				State:     "OK",
				Data:      result,
				Timestamp: time.Now().Unix(),
			}, w); err != nil {
				pcolor.PrintErr(prefix, "%+v\n", err)
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
