package main

import (
	"io"
	"log"
	"net"
	"os"
)

func main() {
	target := os.Getenv("TARGET_ADDR")
	if target == "" {
		target = "172.29.143.230:8080"
	}
	l, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
	defer l.Close()
	log.Printf("Proxying 127.0.0.1:8080 -> %s", target)

	for {
		client, err := l.Accept()
		if err != nil {
			continue
		}
		go func(c net.Conn) {
			defer c.Close()
			remote, err := net.Dial("tcp", target)
			if err != nil {
				return
			}
			defer remote.Close()
			done := make(chan struct{}, 2)
			go func() {
				io.Copy(remote, c)
				done <- struct{}{}
			}()
			go func() {
				io.Copy(c, remote)
				done <- struct{}{}
			}()
			<-done
		}(client)
	}
}
