package main

import (
	"net"
	"time"
)

func wait(address string) {
	for i := 0; i < 50; i++ {
		connection, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
