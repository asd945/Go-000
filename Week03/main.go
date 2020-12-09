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
	stop := make(chan struct{}, 1)
	group, _ := errgroup.WithContext(context.Background())

	server := &http.Server{Addr: ":8080", Handler: &Foo{}}
	debug := &http.Server{Addr: ":8081", Handler: &Foo{}}

	group.Go(func() error {
		fmt.Println("start http server...")
		return server.ListenAndServe()
	})
	group.Go(func() error {
		fmt.Println("start debug server...")
		return debug.ListenAndServe()
	})

	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
		select {
		case <-stop:
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
		stop <- struct{}{}
	}


}

type Foo struct{}

func (f *Foo) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprint(writer, "bar")
}
