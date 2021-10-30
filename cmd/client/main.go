package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/cardinaltm/blockchain/internal/blockchain"
	"github.com/cardinaltm/blockchain/internal/network"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

var (
	Addresses []string
	User      *blockchain.User
)

const (
	ADD_BLOCK = iota + 1
	ADD_TRNSX
	GET_BLOCK
	GET_LHASH
	GET_BLNCE
)

func init() {
	if len(os.Args) < 2 {
		panic("failed 1")
	}

	var (
		addrStr     = ""
		userNewStr  = ""
		userLoadStr = ""
	)

	var (
		addrExist     = false
		userNewExist  = false
		userLoadExist = false
	)

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch {
		case strings.HasPrefix(arg, "-loadaddr:"):
			addrStr = strings.Replace(arg, "-loadaddr:", "", 1)
			addrExist = true
		case strings.HasPrefix(arg, "-newuser:"):
			userNewStr = strings.Replace(arg, "-newuser:", "", 1)
			userNewExist = true
		case strings.HasPrefix(arg, "-loaduser:"):
			userLoadStr = strings.Replace(arg, "-loaduser:", "", 1)
			userLoadExist = true
		}
	}

	if !(userNewExist || userLoadExist) || !addrExist {
		panic("failed 2")
	}

	err := json.Unmarshal([]byte(readFile(addrStr)), &Addresses)
	if err != nil {
		panic("failed 3")
	}

	if len(Addresses) == 0 {
		panic("failed 4")
	}
	if userNewExist {
		User = userNew(userNewStr)
	}
	if userLoadExist {
		User = userLoad(userLoadStr)
	}
	if User == nil {
		panic("failed 5")
	}
}

func main() {
	handleClient()
}

func readFile(filename string) string {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return ""
	}
	return string(data)
}

func userNew(filename string) *blockchain.User {
	user := blockchain.NewUser()
	if user == nil {
		return nil
	}

	err := writeFile(filename, user.Purse())
	if err != nil {
		return nil
	}
	return user
}

func userLoad(filename string) *blockchain.User {
	priv := readFile(filename)
	if priv == "" {
		return nil
	}
	user := blockchain.LoadUser(priv)
	if user == nil {
		return nil
	}
	return user
}

func writeFile(filename string, data string) error {
	return ioutil.WriteFile(filename, []byte(data), 0644)
}

func handleClient() {
	var (
		message string
		splited []string
	)
	for {
		message = inputString("> ")
		splited = strings.Split(message, " ")
		switch splited[0] {
		case "/exit":
			os.Exit(0)
		case "/user":
			if len(splited) < 2 {
				fmt.Println("len(user) < 2")
				continue
			}
			switch splited[1] {
			case "address":
				userAddress()
			case "purse":
				userPurse()
			case "balance":
				userBalance()
			}
		case "/chain":
			if len(splited) < 2 {
				fmt.Println("len(user) < 2")
				continue
			}
			switch splited[1] {
			case "print":
				chainPrint()
			case "tx":
				chainTX(splited[1:])
			case "balance":
				chainBalance(splited[1:])
			}
		default:
			fmt.Println("undefined command\n")
		}
	}
}

func inputString(begin string) string {
	fmt.Print(begin)
	msg, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.Replace(msg, "\n", "", 1)
}

func userAddress() {
	fmt.Println("Address: ", User.Address(), "\n")
}

func userPurse() {
	fmt.Println("Purse: ", User.Purse(), "\n")
}

func userBalance() {
	printBalance(User.Address())
}

func chainPrint() {
	for i := 0; ; i++ {
		res := network.Send(Addresses[0], &network.Package{
			Option: GET_BLOCK,
			Data:   fmt.Sprintf("%d", i),
		})

		if res == nil || res.Data == "" {
			break
		}
		fmt.Printf("[%d] => %s\n", i+1, res.Data)
	}
	fmt.Println()
}

func chainTX(splited []string) {
	if len(splited) != 3 {
		fmt.Println("len(splited) != 3")
		return
	}
	num, err := strconv.Atoi(splited[2])
	if err != nil {
		fmt.Println("strconv error")
		return
	}

	for _, addr := range Addresses {
		res := network.Send(addr, &network.Package{
			Option: GET_LHASH,
		})
		if err == nil {
			continue
		}
		tx := blockchain.NewTransaction(User, blockchain.Base64Decode(res.Data), splited[1], uint64(num))
		if tx == nil {
			fmt.Println("tx is null")
			return
		}
		res = network.Send(addr, &network.Package{
			Option: ADD_TRNSX,
			Data:   blockchain.SerializeTX(tx),
		})
		if res == nil {
			continue
		}
		if res.Data == "ok" {
			fmt.Printf("ok: (%s)\n", addr)
		} else {
			fmt.Printf("fail: (%s)\n", addr)
		}
	}
	fmt.Println()
}

func chainBalance(splited []string) {
	if len(splited) != 2 {
		fmt.Println("len(splited) != 2")
		return
	}
	printBalance(splited[1])
}

func printBalance(address string) {
	for _, addr := range Addresses {
		res := network.Send(addr, &network.Package{
			Option: GET_BLNCE,
			Data:   address,
		})
		if res == nil {
			continue
		}
		fmt.Printf("Balance (%s): %s coins\n", addr, res.Data)
	}
	fmt.Println()
}
