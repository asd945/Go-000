package main

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	baseCtx, cancel  := context.WithCancel(context.Background())
	group, ctx := errgroup.WithContext(baseCtx)
	server := &http.Server{Addr: ":8080", Handler: &Foo{}}
	debug := &http.Server{Addr: ":8081", Handler: &Foo{}}

	group.Go(func() (err error) {
		go func() {
			<- ctx.Done()
			_ = server.Shutdown(context.Background())
		}()
		fmt.Println("start http server...")
		err = server.ListenAndServe()
		return
	})
	group.Go(func() (err error) {
		go func() {
			<- ctx.Done()
			_ = debug.Shutdown(context.Background())
		}()
		fmt.Println("start debug server...")
		err = debug.ListenAndServe()
		return
	})

	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
		select {
		case c := <-signalChan:
			fmt.Println("receive signal: ", c)
			cancel()
		}
	}()

	if err := group.Wait(); err != nil {
		fmt.Println("all shutdown, err is:", err)
	}

}

type Foo struct{}

func (f *Foo) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprint(writer, "bar")
}
