package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"seed-germination-workbench/internal/selfcheck"
	"seed-germination-workbench/internal/store"
	"seed-germination-workbench/internal/web"
	"seed-germination-workbench/internal/workflow"
	"syscall"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:19081", "监听地址")
	self := flag.Bool("selfcheck", false, "运行自检")
	flag.Parse()
	if env := os.Getenv("PORT"); env != "" && !flag.CommandLine.Parsed() {
		*addr = "127.0.0.1:" + env
	}
	if env := os.Getenv("PORT"); env != "" && *addr == "127.0.0.1:19081" {
		*addr = "127.0.0.1:" + env
	}
	if !validAddr(*addr) {
		fmt.Fprintln(os.Stderr, "地址必须为回环 host:port")
		os.Exit(2)
	}
	dir := filepath.Join(".data")
	if *self {
		d, e := os.MkdirTemp("", "seed-selfcheck-")
		if e != nil {
			panic(e)
		}
		defer os.RemoveAll(d)
		dir = d
	}
	st, e := store.New(dir)
	if e != nil {
		panic(e)
	}
	svc := workflow.New(st)
	srv := web.New(svc, *addr)
	go srv.ListenAndServe()
	if *self {
		wait(*addr)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		e = selfcheck.Runner{Base: "http://" + *addr}.Run(ctx)
		srv.Shutdown()
		if e != nil {
			fmt.Fprintln(os.Stderr, "自检失败:", e)
			os.Exit(1)
		}
		fmt.Println("自检通过")
		return
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	srv.Shutdown()
}
