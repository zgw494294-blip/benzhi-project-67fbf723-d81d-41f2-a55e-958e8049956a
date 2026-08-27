package main

import "strings"

func validAddr(address string) bool {
	return strings.HasPrefix(address, "127.0.0.1:") && len(strings.Split(address, ":")) == 2
}
