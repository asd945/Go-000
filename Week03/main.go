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

	shouldStop := make(chan error, 2)

	group.Go(func() (err error) {
		defer func() {
			shouldStop <- fmt.Errorf("server err, %v", err)
		}()
		fmt.Println("start http server...")
		err = server.ListenAndServe()
		return
	})
	group.Go(func() (err error) {
		defer func() {
			shouldStop <- fmt.Errorf("debug server err, %v", err)
		}()
		fmt.Println("start debug server...")
		err = debug.ListenAndServe()
		return
	})

	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

		select {
		case err := <-shouldStop:
			fmt.Println("receive err:", err)
		case c := <-signalChan:
			fmt.Println("receive signal: ", c)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
