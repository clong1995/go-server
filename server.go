package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/clong1995/go-config"
	"github.com/clong1995/go-db-kv"
	"github.com/clong1995/go-encipher/gob"
	"github.com/clong1995/go-encipher/json"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

func init() {
	mId := config.Value("MACHINE ID")
	if mId == "" {
		log.Fatalln("MACHINE not found")
	}

	addr := fmt.Sprintf(":90%s", mId)

	http.HandleFunc("/", handler)

	httpserver = &http.Server{
		Addr:    addr,
		Handler: nil,
	}

	/*stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		_ = <-stop
	}()*/

	go func() {
		err := httpserver.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalln(err)
			return
		}
		fmt.Println("[http] server exited!")
	}()
	fmt.Printf("[http] listening %s\n", addr)
}

var httpserver *http.Server

var hs = newHandles()

var re = regexp.MustCompile(`"t":\d+,"a":"[^"]+",?`)

func Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpserver.Shutdown(ctx); err != nil {
		log.Println(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	var err error
	defer func() {
		_ = r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}()

	handle, ok := hs.get(r.URL.Path)
	if !ok {
		err = errors.New("not found")
		log.Println(err)
		return
	}

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

	if handle.Cache != "" {
		var all []byte
		if all, err = io.ReadAll(r.Body); err != nil {
			log.Println(err)
			return
		}
		process := func() (result []byte, err error) {
			res, err := handle.Process(uid, func(req any) (err error) {
				buf := bytes.NewBuffer(all)
				if handle.Gob {
					if err = gob.Decode(buf, req); err != nil {
						log.Println(err)
						return
					}
				} else {
					if err = json.Decode(buf, req); err != nil {
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
			buf := new(bytes.Buffer)
			if handle.Gob {
				if res == nil {
					return
				}
				_ = gob.Encode(res, buf)
			} else {
				var now int64
				if handle.Cache == "" {
					now = time.Now().Unix()
				}
				_ = json.Encode(&response{
					State:     "OK",
					Data:      res,
					Timestamp: now,
				}, buf)
			}
			result = buf.Bytes()
			return
		}
		param := re.ReplaceAllString(string(all), "")
		key := kv.HashKey(fmt.Sprintf(
			"%s%s%s",
			config.Value("MACHINE ID"),
			handle.Uri,
			param,
		))

		var cacheType string
		ttl := 15000
		arr := strings.Split(handle.Cache, ":")
		if len(arr) == 1 {
			cacheType = arr[0]
		} else if len(arr) == 2 {
			cacheType = arr[0]
			if ttl, err = strconv.Atoi(arr[1]); err != nil {
				log.Println(err)
				return
			}
		} else {
			err = errors.New("cache type error")
			log.Println(err)
			return
		}

		var storage []byte
		switch cacheType {
		case "perm":
			if storage, err = kv.Storage[[]byte](key, process); err != nil {
				log.Println(err)
				return
			}
		case "ttl":
			if storage, err = kv.StorageTtl[[]byte](key, process, ttl); err != nil {
				log.Println(err)
				return
			}
		case "ttl-dsc":
			if storage, err = kv.StorageTtlDiscord[[]byte](key, process, ttl); err != nil {
				log.Println(err)
				return
			}
		}
		_, _ = w.Write(storage)
	} else {
		var result any
		if result, err = handle.Process(uid, func(req any) (err error) {
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
		}); err != nil {
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
	h.mu.Lock()         // 获取写锁
	defer h.mu.Unlock() // 确保锁被释放
	h.data[key] = value
}

func (h *handles) get(key string) (value Handle, ok bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	value, ok = h.data[key]
	return
}

func newHandles() *handles {
	return &handles{
		data: make(map[string]Handle),
	}
}
