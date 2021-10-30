package main

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/cardinaltm/blockchain/internal/blockchain"
	"github.com/cardinaltm/blockchain/internal/network"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"
)

var (
	Filename    string
	Addresses   []string
	User        *blockchain.User
	Serve       string
	Chain       *blockchain.BlockChain
	Block       *blockchain.Block
	Mutex       sync.Mutex
	IsMining    bool
	BreakMining = make(chan bool)
)

const (
	SEPARATOR = "_SEPARATOR_"
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
		serveStr     = ""
		addrStr      = ""
		userNewStr   = ""
		userLoadStr  = ""
		chainNewStr  = ""
		chainLoadStr = ""
	)

	var (
		serveExist     = false
		addrExist      = false
		userNewExist   = false
		userLoadExist  = false
		chainNewExist  = false
		chainLoadExist = false
	)

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch {
		case strings.HasPrefix(arg, "-serve:"):
			serveStr = strings.Replace(arg, "-serve:", "", 1)
			serveExist = true
		case strings.HasPrefix(arg, "-newchain:"):
			chainNewStr = strings.Replace(arg, "-newchain:", "", 1)
			chainLoadExist = true
		case strings.HasPrefix(arg, "-loadchain:"):
			chainLoadStr = strings.Replace(arg, "-loadchain:", "", 1)
			chainLoadExist = true
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

	if !(userNewExist || userLoadExist) || !addrExist || !serveExist || !(chainNewExist || chainLoadExist) {
		panic("failed 2")
	}

	Serve = serveStr

	var addresses []string
	err := json.Unmarshal([]byte(readFile(addrStr)), &addresses)
	if err != nil {
		panic("failed 3")
	}
	var mapaddr = make(map[string]bool)
	for _, addr := range addresses {
		if addr == Serve {
			continue
		}
		if _, ok := mapaddr[addr]; ok {
			continue
		}
		mapaddr[addr] = true
		Addresses = append(Addresses, addr)
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
	if chainNewExist {
		Filename = chainNewStr
		Chain = chainNew(chainNewStr)
	}
	if chainLoadExist {
		Filename = chainLoadStr
		Chain = chainLoad(chainLoadStr)
	}
	if Chain == nil {
		panic("failed: load chain")
	}
	Block = blockchain.NewBlock(User.Address(), Chain.LastHash())
}

func main() {
	network.Listen(Serve, handleServer)
	for {
		fmt.Scanln()
	}
}

func handleServer(conn network.Conn, pack *network.Package) {
	network.Handle(ADD_BLOCK, conn, pack, addBlock)
	network.Handle(ADD_TRNSX, conn, pack, addTransaction)
	network.Handle(GET_BLOCK, conn, pack, getBlock)
	network.Handle(GET_LHASH, conn, pack, getLastHash)
	network.Handle(GET_BLNCE, conn, pack, getBalance)
}

func addBlock(pack *network.Package) string {
	splited := strings.Split(pack.Data, SEPARATOR)
	if len(splited) != 3 {
		return "fail"
	}
	block := blockchain.DeserializeBlock(splited[2])
	if !block.IsValid(Chain) {
		currSize := Chain.Size()
		num, err := strconv.Atoi(splited[1])
		if err != nil {
			return "fail"
		}
		if currSize < uint64(num) {
			go compareChains(splited[0], uint64(num))
			return "ok"
		}
		return "fail"
	}

	Mutex.Lock()
	Chain.AddBlock(block)
	Block = blockchain.NewBlock(User.Address(), Chain.LastHash())
	Mutex.Unlock()

	if IsMining {
		BreakMining <- true
		IsMining = false
	}

	return "ok"
}

func addTransaction(pack *network.Package) string {
	var tx = blockchain.DeserializeTX(pack.Data)
	if tx == nil || len(Block.Transactions) == blockchain.TXS_LIMIT {
		return "fail"
	}
	Mutex.Lock()
	err := Block.AddTransaction(Chain, tx)
	Mutex.Unlock()
	if err != nil {
		return "fail"
	}
	if len(Block.Transactions) == blockchain.TXS_LIMIT {
		go func() {
			Mutex.Lock()
			block := *Block
			IsMining = true
			Mutex.Unlock()
			err := (&block).Accept(Chain, User, BreakMining)
			Mutex.Lock()
			IsMining = false
			if err == nil && bytes.Equal(block.PrevHash, Block.PrevHash) {
				Chain.AddBlock(&block)
				pushBlockToNet(&block)
			}
			Block = blockchain.NewBlock(User.Address(), Chain.LastHash())
			Mutex.Unlock()
		}()
	}
	return "ok"
}

func getBlock(pack *network.Package) string {
	num, err := strconv.Atoi(pack.Data)
	if err != nil {
		return ""
	}
	size := Chain.Size()
	if uint64(num) < size {
		return selectBlock(Chain, num)
	}
	return ""
}

func getLastHash(pack *network.Package) string {
	return blockchain.Base64Encode(Chain.LastHash())
}

func getBalance(pack *network.Package) string {
	return fmt.Sprintf("%d", Chain.Balance(pack.Data))
}

func compareChains(address string, num uint64) {
	filename := "temp_" + hex.EncodeToString(blockchain.GenerateRandomBytes(8))
	file, err := os.Create(filename)
	if err != nil {
		return
	}
	file.Close()
	defer func() {
		os.Remove(filename)
	}()
	res := network.Send(address, &network.Package{
		Option: GET_BLOCK,
		Data:   fmt.Sprintf("%d", 0),
	})
	if res == nil {
		return
	}
	genesis := blockchain.DeserializeBlock(res.Data)
	if genesis == nil {
		return
	}
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return
	}
	defer db.Close()
	_, err = db.Exec(blockchain.CREATE_TABLE)
	chain := &blockchain.BlockChain{
		DB: db,
	}
	chain.AddBlock(genesis)
	for i := uint64(1); i < num; i++ {
		res := network.Send(address, &network.Package{
			Option: GET_BLOCK,
			Data:   fmt.Sprintf("%d", i),
		})
		if res == nil {
			return
		}
		block := blockchain.DeserializeBlock(res.Data)
		if block == nil {
			return
		}
		if !block.IsValid(chain) {
			return
		}
		chain.AddBlock(block)
	}
	Mutex.Lock()
	Chain.DB.Close()
	os.Remove(Filename)
	copyFile(filename, Filename)
	Chain = blockchain.LoadChain(Filename)
	Block = blockchain.NewBlock(User.Address(), Chain.LastHash())
	Mutex.Unlock()
	if IsMining {
		BreakMining <- true
		IsMining = false
	}
}

func pushBlockToNet(block *blockchain.Block) {
	var (
		sblock = blockchain.SerializeBlock(block)
		msg    = Serve + SEPARATOR + fmt.Sprintf("%d", Chain.Size()) + SEPARATOR + sblock
	)

	for _, addr := range Addresses {
		go network.Send(addr, &network.Package{
			Option: ADD_BLOCK,
			Data:   msg,
		})
	}
}

func selectBlock(chain *blockchain.BlockChain, i int) string {
	var sblock string
	row := chain.DB.QueryRow("SELECT Block FROM BlockChain WHERE Id=$1", i+1)
	row.Scan(&sblock)
	return sblock
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func readFile(filename string) string {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return ""
	}
	return string(data)
}

func writeFile(filename string, data string) error {
	return ioutil.WriteFile(filename, []byte(data), 0644)
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

func chainNew(filename string) *blockchain.BlockChain {
	err := blockchain.NewChain(filename, User.Address())
	if err != nil {
		return nil
	}
	return blockchain.LoadChain(filename)
}

func chainLoad(filename string) *blockchain.BlockChain {
	chain := blockchain.LoadChain(filename)
	return chain
}
