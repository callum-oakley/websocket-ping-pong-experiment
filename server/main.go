package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

const (
	pingInterval = 2 * time.Second
	readTimeout  = 2 * pingInterval
	writeTimeout = pingInterval
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func protocolPingLoop(conn *websocket.Conn) error {
	for n := 0; ; n++ {
		<-time.After(pingInterval)
		if err := conn.WriteControl(
			websocket.PingMessage,
			[]byte(strconv.Itoa(n)),
			time.Now().Add(writeTimeout),
		); err != nil {
			return fmt.Errorf("protocolPingLoop: %w", err)
		}
	}
}

func applicationPingLoop(conn *websocket.Conn) error {
	for n := 0; ; n++ {
		<-time.After(pingInterval)
		message := strconv.Itoa(n)
		if err := conn.WriteMessage(
			websocket.TextMessage,
			[]byte(message),
		); err != nil {
			return fmt.Errorf("applicationPingLoop: %w", err)
		}
		log.Printf("sent application ping %s", message)
	}
}

func readLoop(conn *websocket.Conn) error {
	for {
		if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			return fmt.Errorf("readLoop: SetReadDeadline: %w", err)
		}
		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("readLoop: ReadMessage: %w", err)
		}
		log.Printf("received application pong %s", message)
	}
}

func pingHandler(appData string) error {
	log.Printf("received protocol ping %s", appData)
	return nil
}

func pongHandler(appData string) error {
	log.Printf("received protocol pong %s", appData)
	return nil
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	conn.SetPingHandler(pingHandler)
	conn.SetPongHandler(pongHandler)

	var g errgroup.Group
	// g.Go(func() error { return protocolPingLoop(conn) })
	g.Go(func() error { return applicationPingLoop(conn) })
	g.Go(func() error { return readLoop(conn) })

	if err := g.Wait(); err != nil {
		log.Println("error:", err)
	}
}

func main() {
	fs := http.FileServer(http.Dir("client"))
	http.Handle("/", fs)
	http.HandleFunc("/server", handleWS)
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Println("error:", err)
	}
}
