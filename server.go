package server

import (
	"github.com/clong1995/go-config"
	"log"
	"net/http"
)

// ListenAndServe 启动服务
func ListenAndServe() {
	mux := http.NewServeMux()
	//执行路由表
	for _, handle := range handles {
		register(mux, handle)
	}
	addr := ":90" + config.Config("PORT")
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
