package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"

	"golang.org/x/crypto/ssh"
)

func runServer(c net.Conn) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(fmt.Sprintf("failed to generate rsa host key: %s", err))
	}

	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		panic(fmt.Sprintf("failed to create signer for host key: %s", err))
	}

	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	config.AddHostKey(signer)

	conn, channels, requests, err := ssh.NewServerConn(c, config)
	if err != nil {
		panic(fmt.Sprintf("failed to set up server-side connection: %s", err))
	}
	defer conn.Close()

	go ssh.DiscardRequests(requests)

	for newChannel := range channels {
		if err := newChannel.Reject(ssh.UnknownChannelType, ""); err != nil {
			panic(fmt.Sprintf("failed to reject channel: %s", err))
		}
	}
}

func runClient(address string) {
	client, err := ssh.Dial("tcp", address, &ssh.ClientConfig{
		User:            "root",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		panic(fmt.Sprintf("failed to set up client-side connection: %s", err))
	}

	for {
		_, _, err := client.OpenChannel("test", nil)
		if err == nil {
			panic(fmt.Sprintf("should have rejected channel"))
		}
		if _, ok := err.(*ssh.OpenChannelError); !ok {
			panic(fmt.Sprintf("should have rejected channel: %s", err))
		}
	}
}

func main() {
	listener, err := net.Listen("tcp", "localhost:2222")
	if err != nil {
		panic(fmt.Sprintf("failed to listen on tcp port: %s", err))
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				panic(fmt.Sprintf("failed to accept on tcp port: %s", err))
			}
			go runServer(conn)
		}
	}()

	for i := 0; i < 10; i++ {
		go runClient("localhost:2222")
	}

	http.ListenAndServe("localhost:6060", nil)
}
