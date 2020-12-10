package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

func main() {
	group, _ := errgroup.WithContext(context.Background())

	server := &http.Server{Addr: ":8080", Handler: &Foo{}}
	debug := &http.Server{Addr: ":8081", Handler: &Foo{}}

	shouldStop := make(chan struct{}, 2)

	group.Go(func() error {
		defer func() {
			shouldStop <- struct{}{}
		}()
		fmt.Println("start http server...")
		return server.ListenAndServe()
	})
	group.Go(func() error {
		defer func() {
			shouldStop <- struct{}{}
		}()
		fmt.Println("start debug server...")
		return debug.ListenAndServe()
	})

	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
		select {
		case <-shouldStop:
			fmt.Println("receive err")
		case c := <-signalChan:
			fmt.Println("receive signal: ", c)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		_ = debug.Shutdown(ctx)
	}()

	if err := group.Wait(); err != nil {
		fmt.Println("all shutdown")
 	}


}

type Foo struct{}

func (f *Foo) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprint(writer, "bar")
}
