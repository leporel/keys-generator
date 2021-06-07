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
	CheckBTC(addresses []string) (map[string]BTCData, bool, error) // 128 list, 58 per minute
	// https://etherscan.io/apis
	CheckETH(addresses []string) ([]ETHAccount, bool, error) // 20 list, 300 per minute (5 sec/IP)
	AvaibleProxy() int
}

type Chkr struct {
	btcApi string
	ethApi string

	RBAPI     RoundRobin
	RBClients RoundRobin
}

func (c *Chkr) CheckBTC(addresses []string) (map[string]BTCData, bool, error) {
	return c.checkBtcBalanceWallet(addresses)
}

func (c *Chkr) CheckETH(addresses []string) ([]ETHAccount, bool, error) {
	return c.checkEthBalanceWallet(addresses)
}

func (c *Chkr) AvaibleProxy() int {
	return c.RBClients.Len()
}

func NewChecker(rateLimit int, ethApiKey []string) Checker {
	var apiKeys []rbElement
	for _, s := range ethApiKey {
		apiKeys = append(apiKeys, &apiKey{
			s,
		})
	}
	rbAPI := roundRobinNew(apiKeys)

	defaultClient := &client{
		name:   "local client",
		client: http.DefaultClient,
		rl:     ratelimit.New(rateLimit, ratelimit.Per(60*time.Second)),
	}
	clients := []rbElement{defaultClient}

	proxyList, err := readLines("./proxy.txt")
	if err != nil {
		panic(err)
	}
	for _, s := range proxyList {
		proxyCli, err := NewProxyClient(s)
		if err != nil {
			panic(err)
		}
		cli := &client{
			name:   s,
			client: proxyCli,
			rl:     ratelimit.New(rateLimit, ratelimit.Per(60*time.Second)),
		}
		clients = append(clients, cli)
	}

	rbCli := roundRobinNew(clients)

	return &Chkr{
		btcApi:    "https://blockchain.info/balance?cors=true&active=",
		ethApi:    "https://api.etherscan.io/api?module=account&action=balancemulti&apikey=%s&address=%s",
		RBAPI:     rbAPI,
		RBClients: rbCli,
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

	maxTry := 5
	var rsErr error
	for i := 0; i < maxTry; i++ {
		elemC, err := c.RBClients.Next()
		if err != nil {
			return nil, false, fmt.Errorf("not avaible proxys")
		}
		cli := elemC.Get().(*client)
		_ = cli.rl.Take()

		resp, err := cli.client.Get(c.btcApi + list)
		if err != nil {
			if c.RBClients.Delete(elemC) {
				fmt.Printf("\nproxy not working [%s] removed from list \n", cli.name)
			}
			continue
		}

		if resp.StatusCode != 200 {
			rsErr = fmt.Errorf("blockchain.info/balance resp status %v", resp.Status)
			continue
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

	if rsErr != nil {
		return nil, false, rsErr
	}

	return nil, false, fmt.Errorf("did nothing")
}

func (c *Chkr) checkEthBalanceWallet(compressed []string) ([]ETHAccount, bool, error) {
	if len(compressed) > 20 {
		return nil, false, fmt.Errorf("maxsimum adress list 20")
	}

	list := compressed[0]
	for i := 1; i < len(compressed); i++ {
		list = fmt.Sprintf("%s,%s", list, compressed[i])
	}

	maxTry := 5
	var rsErr error
	for i := 0; i < maxTry; i++ {

		elemK, err := c.RBAPI.Next()
		if err != nil {
			return nil, false, fmt.Errorf("ether scan api keys gone")
		}

		apiKey := elemK.Get().(string)

		elemC, err := c.RBClients.Next()
		if err != nil {
			return nil, false, fmt.Errorf("not avaible proxys")
		}
		cli := elemC.Get().(*client)
		_ = cli.rl.Take()

		resp, err := cli.client.Get(fmt.Sprintf(c.ethApi, apiKey, list))
		if err != nil {
			if c.RBClients.Delete(elemC) {
				fmt.Printf("\nproxy not working [%s] removed from list \n", cli.name)
			}
			continue
		}

		if resp.StatusCode != 200 {
			rsErr = fmt.Errorf("api.etherscan.io resp status %v", resp.Status)
			continue
		}

		var result ETHData

		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return nil, false, err
		}

		if result.Status != "1" {
			rsErr = fmt.Errorf("api.etherscan.io status %v", result.Status)
			continue
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
	if rsErr != nil {
		return nil, false, rsErr
	}

	return nil, false, fmt.Errorf("did nothing")
}

type rbElement interface {
	Get() interface{}
}

type apiKey struct {
	key string
}

func (a *apiKey) Get() interface{} {
	return a.key
}

type client struct {
	name   string
	client *http.Client
	rl     ratelimit.Limiter
}

func (c *client) Get() interface{} {
	return c
}
