package server

import (
	"github.com/clong1995/go-config"
	"log"
	"net/http"
)

var handles = make([]Handle, 0)

// Listen 启动服务
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
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalln(err)
		return
	}
	return
}
