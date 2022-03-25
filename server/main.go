package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
)

var (
	entering = make(chan chan<- string)
	leaving  = make(chan chan<- string)
	messages = make(chan string)
)

func broadcaster() {
	clients := make(map[chan<- string]bool)
	for {
		select {
		case msg := <-messages:
			for cli := range clients {
				cli <- msg
			}
		case cli := <-entering:
			clients[cli] = true
		case cli := <-leaving:
			delete(clients, cli)
			close(cli)
		}
	}
}

func messageToClientWriter(conn net.Conn, msgToClient <-chan string) {
	for msg := range msgToClient {
		_, err := io.WriteString(conn, msg)
		if err != nil {
			log.Printf("message couldn't be sent to: %s", conn.RemoteAddr().String())
		}
	}
}

func handleIncommingConnection(conn net.Conn, connectionPool chan struct{}) {
	connectionPool <- struct{}{}
	clientRemoteAddr := conn.RemoteAddr().String()
	log.Printf("connection incomming - %s \n", clientRemoteAddr)
	defer conn.Close()
	defer log.Printf("connection processed - %s\n", clientRemoteAddr)
	defer func(connectionPool chan struct{}) { <-connectionPool }(connectionPool)

	msgToClient := make(chan string)
	go messageToClientWriter(conn, msgToClient)

	msgToClient <- fmt.Sprintf("SERVER: You are '%s'\n", clientRemoteAddr)
	messages <- fmt.Sprintf("SERVER: %s has joined\n", clientRemoteAddr)
	entering <- msgToClient

	msgFromClient := bufio.NewScanner(conn)
	for msgFromClient.Scan() {
		messages <- fmt.Sprintf("%s: %s\n", clientRemoteAddr, msgFromClient.Text())
	}
	leaving <- msgToClient
	messages <- fmt.Sprintf("SERVER: %s has left\n", clientRemoteAddr)
}

const network = "tcp"
const port = 8080
const connPoolSize = 3

func main() {
	listener, err := net.Listen(network, fmt.Sprintf(":%d", port))
	connectionPool := make(chan struct{}, connPoolSize)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("TCP CHAT SERVER listening on port: %d", port)

	go broadcaster()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
		}
		go handleIncommingConnection(conn, connectionPool)
	}
}
