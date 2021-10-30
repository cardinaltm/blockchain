package main

import (
	"fmt"
	"github.com/cardinaltm/blockchain/internal/network"
	"strings"
	"time"
)

const (
	TO_UPPER = iota + 1
	TO_LOWER
)

const (
	ADDRESS = ":8080"
)

func main() {
	go network.Listen(ADDRESS, handleServer)
	time.Sleep(500 * time.Millisecond)

	res := network.Send(ADDRESS, &network.Package{
		Option: TO_UPPER,
		Data:   "Hello World!",
	})
	fmt.Println(res.Data)

	res = network.Send(ADDRESS, &network.Package{
		Option: TO_LOWER,
		Data:   "Hello World!",
	})
	fmt.Println(res.Data)
}

func handleServer(conn network.Conn, pack *network.Package) {
	network.Handle(TO_UPPER, conn, pack, handleToUpper)
	network.Handle(TO_LOWER, conn, pack, handleToLower)
}

func handleToUpper(pack *network.Package) string {
	return strings.ToUpper(pack.Data)
}

func handleToLower(pack *network.Package) string {
	return strings.ToLower(pack.Data)
}
