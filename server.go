package server

import (
	"context"
	"fmt"
	"github.com/clong1995/go-config"
	"log"
	"net/http"
	"time"
)

var server *http.Server
var handles = make([]Handle, 0)

func Listen() {
	mux := http.NewServeMux()
	//执行路由表
	for _, handle := range handles {
		register(mux, handle)
	}

	addr := ":90" + config.Value("MACHINE ID")
	if addr == "" {
		log.Fatalln("ADDR not found")
	}

	//启动服务
	log.Printf("[http] listening %s\n", addr)
	server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Fatalln(err)
			return
		}
	}()
}

func Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("优雅关闭失败: %v\n", err)
	} else {
		fmt.Println("优雅关闭成功")
	}
}
