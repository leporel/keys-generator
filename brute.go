package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"go.uber.org/ratelimit"
	"math/big"
	"net/http"
	"time"
)

type BTCData struct {
	FinalBalance  int `json:"final_balance"`
	NTx           int `json:"n_tx"`
	TotalReceived int `json:"total_received"`
}

type ETHData struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

type ETHAccount struct {
	Account string `json:"account"`
	Balance string `json:"balance"`
}

// https://api.etherscan.io/api?module=account&action=txlist&address=0xddbd2b932c763ba5b1b7ae3b362eac3e8d40121a&startblock=0&endblock=99999999&page=1&offset=1&sort=asc&apikey=YourApiKeyToken
type ETHTX struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  []struct {
		BlockNumber       string `json:"blockNumber"`
		TimeStamp         string `json:"timeStamp"`
		Hash              string `json:"hash"`
		Nonce             string `json:"nonce"`
		BlockHash         string `json:"blockHash"`
		TransactionIndex  string `json:"transactionIndex"`
		From              string `json:"from"`
		To                string `json:"to"`
		Value             string `json:"value"`
		Gas               string `json:"gas"`
		GasPrice          string `json:"gasPrice"`
		IsError           string `json:"isError"`
		TxreceiptStatus   string `json:"txreceipt_status"`
		Input             string `json:"input"`
		ContractAddress   string `json:"contractAddress"`
		CumulativeGasUsed string `json:"cumulativeGasUsed"`
		GasUsed           string `json:"gasUsed"`
		Confirmations     string `json:"confirmations"`
	} `json:"result"`
}

type Checker interface {
	// https://www.blockchain.com/ru/api/blockchain_api
	CheckBTC(addresses []string) (map[string]BTCData, bool, error) // 128 list, ? per minute
	// https://etherscan.io/apis
	CheckETH(addresses []string) ([]ETHAccount, bool, error) // 20 list, 300 per minute (5 	sec/IP)
}

type Chkr struct {
	rl     ratelimit.Limiter
	btcApi string
	ethApi string
}

func (c *Chkr) CheckBTC(addresses []string) (map[string]BTCData, bool, error) {
	_ = c.rl.Take()
	return c.checkBtcBalanceWallet(addresses)
}

func (c *Chkr) CheckETH(addresses []string) ([]ETHAccount, bool, error) {
	_ = c.rl.Take()
	return c.checkEthBalanceWallet(addresses)
}

func NewChecker(rateLimit int, ethApiKey string) Checker {
	rl := ratelimit.New(rateLimit, ratelimit.Per(60*time.Second))

	return &Chkr{
		rl:     rl,
		btcApi: "https://blockchain.info/balance?cors=true&active=",
		ethApi: fmt.Sprintf("https://api.etherscan.io/api?module=account&action=balancemulti&apikey=%s&address=", ethApiKey),
	}
}

func getRand(Range *big.Int) *big.Int {
	n, err := rand.Int(rand.Reader, Range)
	if err != nil {
		panic(err)
	}
	return n
}

func (c *Chkr) checkBtcBalanceWallet(compressed []string) (map[string]BTCData, bool, error) {
	if len(compressed) > 128 {
		return nil, false, fmt.Errorf("maxsimum adress list 128")
	}

	list := ""
	for _, s := range compressed {
		list = fmt.Sprintf("%s|%s", list, s)
	}

	resp, err := http.Get(c.btcApi + list)

	if err != nil {
		return nil, false, err
	}

	if resp.StatusCode != 200 {
		return nil, false, fmt.Errorf("blockchain.info/balance resp status %v", resp.Status)
	}

	result := make(map[string]BTCData, 0)

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, false, err
	}

	var found bool
	for _, data := range result {
		if data.FinalBalance > 0 || data.NTx > 0 {
			found = true
		}
	}

	return result, found, nil
}

func (c *Chkr) checkEthBalanceWallet(compressed []string) ([]ETHAccount, bool, error) {
	if len(compressed) > 20 {
		return nil, false, fmt.Errorf("maxsimum adress list 20")
	}

	list := compressed[0]
	for i := 1; i < len(compressed); i++ {
		list = fmt.Sprintf("%s,%s", list, compressed[i])
	}

	resp, err := http.Get(c.ethApi + list)

	if err != nil {
		return nil, false, err
	}

	if resp.StatusCode != 200 {
		return nil, false, fmt.Errorf("api.etherscan.io resp status %v", resp.Status)
	}

	var result ETHData

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, false, err
	}

	if result.Status != "1" {
		return nil, false, fmt.Errorf("api.etherscan.io status %v", result.Status)
	}

	var accounts []ETHAccount

	err = json.Unmarshal(result.Result, &accounts)
	if err != nil {
		return nil, false, err
	}

	var found bool
	for _, data := range accounts {
		if data.Balance != "0" {
			found = true
		}
	}

	return accounts, found, nil
}
