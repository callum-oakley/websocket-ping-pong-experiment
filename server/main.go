package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

const (
	pingInterval = 5 * time.Second
	readTimeout  = 10 * time.Second
	writeTimeout = 10 * time.Second
)

var upgrader = websocket.Upgrader{}

type connection struct {
	ws    *websocket.Conn
	timer *time.Timer
}

func (c *connection) pingLoop(ctx context.Context) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := c.ws.WriteMessage(
		websocket.TextMessage,
		[]byte(fmt.Sprintf(`{ "read_timeout": %v }`, readTimeout.Seconds())),
	); err != nil {
		return fmt.Errorf("pingLoop: WriteMessage %w", err)
	}
	for n := 0; ; n++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			c.ws.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.ws.WriteMessage(
				websocket.TextMessage,
				[]byte(fmt.Sprintf(`{ "ping": %v }`, n)),
			); err != nil {
				return fmt.Errorf("pingLoop: WriteMessage %w", err)
			}
			log.Printf("sent ping %v", n)
			<-time.After(pingInterval)
		}
	}
}

func (c *connection) readLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			c.ws.SetReadDeadline(time.Now().Add(readTimeout))
			_, message, err := c.ws.ReadMessage()
			if err != nil {
				return fmt.Errorf("readLoop: ReadMessage: %w", err)
			}
			log.Printf("received pong %s", message)
			c.timer.Reset(readTimeout)
		}
	}
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	c := &connection{
		ws: conn,
		timer: time.AfterFunc(readTimeout, func() {
			log.Println("read timeout")
			cancel()
		}),
	}

	var g errgroup.Group
	g.Go(func() error { return c.pingLoop(ctx) })
	g.Go(func() error { return c.readLoop(ctx) })

	if err := g.Wait(); err != nil {
		log.Println("error:", err)
	}
	log.Println("closing connection")
}

func main() {
	fs := http.FileServer(http.Dir("client"))
	http.Handle("/", fs)
	http.HandleFunc("/server", handleWS)
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Println("error:", err)
	}
}
