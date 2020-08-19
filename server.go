package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var clientSenders = make(map[*websocket.Conn]bool) // connected clients
var clientReceivers = make(map[*websocket.Conn]bool)
var broadcast = make(chan []byte) // broadcast channel

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func hello(w http.ResponseWriter, req *http.Request) {

	fmt.Fprintf(w, "hello\n")
}

func headers(w http.ResponseWriter, req *http.Request) {

	for name, headers := range req.Header {
		for _, h := range headers {
			fmt.Fprintf(w, "%v: %v\n", name, h)
		}
	}
}

func handleRecievers(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {

		log.Fatal("in receeiver", err)
	}
	// Make sure we close the connection when the function returns
	// defer ws.Close()

	// Register our new client
	clientReceivers[ws] = true
	log.Printf("receiver conneceted")
	for client := range clientReceivers {
		log.Println("sending confirmed: ")
		err := client.WriteJSON("{data: test}")
		log.Println("confirmed success")
		if err != nil {
			log.Printf("error sending message: %v", err)
			client.Close()
			delete(clientReceivers, client)
		}
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Make sure we close the connection when the function returns
	defer ws.Close()

	// Register our new client
	clientSenders[ws] = true
	log.Printf("connection received")
	for {
		_, p, err := ws.ReadMessage()
		if err != nil {
			log.Printf("error: %v", err)
			delete(clientSenders, ws)
			break
		}
		// Send the newly received message to the broadcast channel
		broadcast <- p
	}
}

func handleMessages() {
	for {
		// Grab the next message from the broadcast channel
		p := <-broadcast
		// Send it out to every client that is currently connected
		for client := range clientReceivers {
			if err := client.WriteMessage(1, p); err != nil {
				log.Printf("error sending message: %v", err)
				client.Close()
				delete(clientReceivers, client)
			}
		}
	}
}

func main() {

	http.HandleFunc("/hello", hello)
	http.HandleFunc("/headers", headers)
	// Configure websocket route
	http.HandleFunc("/wsout", handleConnections)
	http.HandleFunc("/wsin", handleRecievers)

	// Start listening for incoming  messages
	go handleMessages()

	log.Println("http server started on :8090")

	host, _ := os.Hostname()
	addrs, _ := net.LookupIP(host)
	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			fmt.Println("IPv4: ", ipv4)
		}
	}

	err := http.ListenAndServe(":8090", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
