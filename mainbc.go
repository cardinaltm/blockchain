package main

import (
	"fmt"
	"github.com/cardinaltm/blockchain/internal/blockchain"
)

const (
	DBNAME = "blockchain.db"
)

func main() {
	miner := blockchain.NewUser()
	blockchain.NewChain(DBNAME, miner.Address())
	chain := blockchain.LoadChain(DBNAME)

	for i := 0; i < 3; i++ {
		block := blockchain.NewBlock(miner.Address(), chain.LastHash())
		block.AddTransaction(chain, blockchain.NewTransaction(miner, chain.LastHash(), "aaa", 3))
		block.AddTransaction(chain, blockchain.NewTransaction(miner, chain.LastHash(), "bbb", 2))
		block.Accept(chain, miner, make(chan bool))
		chain.AddBlock(block)
	}

	var sblock string
	rows, err := chain.DB.Query("SELECT Block FROM BlockChain")
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		rows.Scan(&sblock)
		fmt.Println(sblock)
	}
}
