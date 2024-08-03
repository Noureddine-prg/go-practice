package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"workshop/http"
)

type Main struct {
	// DB *postgres.DB
	HTTPServer *http.Server
}

func NewMain() *Main {
	return &Main{
		HTTPServer: http.NewServer(),
	}

}

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	m := NewMain()

	if err := m.Run(ctx); err != nil {
		m.Close()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	<-ctx.Done()

	if err := m.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (m *Main) Close() error {
	if m.HTTPServer != nil {
		if err := m.HTTPServer.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (m *Main) Run(ctx context.Context) (err error) {
	if err := m.HTTPServer.Open(); err != nil {
		return err
	}

	log.Printf("running : url=%q debug=http://localhost:6060", m.HTTPServer.URL())

	return nil
}
