package main

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type bruteFunc func(id int, start string, checker Checker, status chan PrinterData, writer func(string))

func main() {
	coin := os.Args[1]

	keysPerPage := 128
	wait := false

	switch coin {
	case "btc":
		printBitcoinKeys(os.Args[2], keysPerPage)
	case "btc-search":
		printBtcWifSearch(os.Args[2], keysPerPage)
	case "eth":
		printEthereumKeys(os.Args[2], keysPerPage)
	case "eth-search":
		printEthPrivateKeySearch(os.Args[2], keysPerPage)
	case "btc-brute":
		bruteKeys(os.Args[2], 50, nil, "", "./btc_output.txt", btcWorker)
	case "bsc-brute":
		var apiKeys []string
		var start = ""
		if len(os.Args) > 3 {
			apiKeys = strings.Split(os.Args[3], ",")
		}
		rate := 290
		if len(apiKeys) == 0 {
			log.Fatal("api key not provided")
		}
		if len(os.Args) > 4 {
			start = os.Args[4]
		}
		bruteKeys(os.Args[2], rate, apiKeys, start, "./bsc_output.txt", bscWorker)

		wait = true
	case "eth-brute":
		var apiKeys []string
		var start = ""
		if len(os.Args) > 3 {
			apiKeys = strings.Split(os.Args[3], ",")
		}
		rate := 270
		if len(apiKeys) == 0 {
			apiKeys = []string{"YourApiKeyToken"}
			rate = 10
		}
		if len(os.Args) > 4 {
			start = os.Args[4]
		}
		bruteKeys(os.Args[2], rate, apiKeys, start, "./eth_output.txt", ethWorker)

		wait = true
	default:
		log.Fatal("Invalid coin type")
	}

	if wait {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		os.Exit(1)
	}
}

func printBitcoinKeys(pageNumber string, keysPerPage int) {
	bitcoinKeys := generateBitcoinKeys(pageNumber, keysPerPage)

	length := len(bitcoinKeys)

	for i, key := range bitcoinKeys {
		fmt.Printf("%v", key)

		if i != length-1 {
			fmt.Print("\n")
		}
	}
}

func printBtcWifSearch(wif string, keysPerPage int) {
	pageNumber := findBtcWifPage(wif, keysPerPage)

	fmt.Printf("%v", pageNumber)
}

func printEthereumKeys(pageNumber string, keysPerPage int) {
	ethereumKeys := generateEthereumKeys(pageNumber, keysPerPage)

	length := len(ethereumKeys)

	for i, key := range ethereumKeys {
		fmt.Printf("%v", key)

		if i != length-1 {
			fmt.Print("\n")
		}
	}
}

func printEthPrivateKeySearch(privateKey string, keysPerPage int) {
	pageNumber := findEthPrivateKeyPage(privateKey, keysPerPage)

	fmt.Printf("%v", pageNumber)
}

func bruteKeys(workers string, limit int, apiKeys []string, start string, outFile string, brute bruteFunc) {
	checker := NewChecker(limit, apiKeys)

	maxWorkers, err := strconv.Atoi(workers)
	if err != nil {
		panic(err)
	}

	printer := Printer{
		mu:           &sync.Mutex{},
		ch:           make(chan PrinterData, 10),
		workersState: workerState{},
		workers:      maxWorkers,
		checker:      checker,
	}

	f, err := os.OpenFile(outFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	//defer f.Close()

	var mu sync.Mutex
	writer := func(data string) {
		mu.Lock()
		defer mu.Unlock()
		if _, err = f.WriteString(data); err != nil {
			panic(err)
		}
	}

	go printer.work()

	for i := 0; i < maxWorkers; i++ {
		go brute(i, start, checker, printer.ch, writer)
	}
}

func btcWorker(id int, start string, checker Checker, status chan PrinterData, writer func(string)) {
	max := makeBigInt("904625697166532776746648320380374280100293470930272690489102837043110636675")
	var pages uint64 = 0

	for true {
		founds := 0
		pages++
		pageNumber := getRand(max).String()
		bitcoinKeys := generateBitcoinKeys(pageNumber, 128)

		var toCheck []string

		for _, k := range bitcoinKeys {
			toCheck = append(toCheck, k.compressed)
		}

		result, found, err := checker.CheckBTC(toCheck)

		if err != nil {
			status <- PrinterData{
				number:     id,
				found:      founds,
				pageNumber: fmt.Sprintf("%s(%d adresses)", pageNumber, 128),
				error:      err,
			}
			continue
		}

		if found {
			for compressed, data := range result {
				if data.FinalBalance > 0 || data.NTx > 0 {
					founds++

					var private, uncompressed string
					for _, bitcoinKey := range bitcoinKeys {
						if bitcoinKey.compressed == compressed {
							private = bitcoinKey.private
							uncompressed = bitcoinKey.uncompressed
						}
					}

					writer(fmt.Sprintf("%v balance (total %v) (%v tx) | %v | %v | %v\n",
						data.FinalBalance, data.TotalReceived, data.NTx, compressed, uncompressed, private))
				}
			}
		}

		status <- PrinterData{
			number:     id,
			found:      founds,
			pageNumber: fmt.Sprintf("%s(%d adresses)", pageNumber, 128),
			error:      err,
		}
	}
}

func ethWorker(id int, start string, checker Checker, status chan PrinterData, writer func(string)) {
	max := makeBigInt("904625697166532776746648320380374280100293470930272690489102837043110636675")
	var pages int64 = 0
	var startPage *big.Int
	if start != "" {
		startPage = makeBigInt(findEthPrivateKeyPage(start, 20))
	}

	for true {
		founds := 0
		pages++
		var pageNumber string

		if startPage != nil {
			pageNumber = fmt.Sprintf("%d", new(big.Int).Add(startPage, big.NewInt(pages-1)))
		} else {
			pageNumber = getRand(max).String()
		}

		ethereumKeys := generateEthereumKeys(pageNumber, 20)

		var toCheck []string

		for _, k := range ethereumKeys {
			toCheck = append(toCheck, k.public)
		}

		result, found, err := checker.CheckETH(toCheck)

		if err != nil {
			status <- PrinterData{
				number:     id,
				found:      founds,
				pageNumber: fmt.Sprintf("%s(%d adresses)", pageNumber, 20),
				error:      err,
			}
			continue
		}

		if found {
			for _, data := range result {
				if data.Balance != "0" {
					founds++

					var private string
					for _, eKey := range ethereumKeys {
						if eKey.public == data.Account {
							private = eKey.private
						}
					}

					writer(fmt.Sprintf("%v balance | %v | %v\n",
						data.Balance, data.Account, private))
				}
			}
		}

		status <- PrinterData{
			number:     id,
			found:      founds,
			pageNumber: fmt.Sprintf("%s(%d adresses)", pageNumber, 20),
			error:      err,
		}
	}
}

func bscWorker(id int, start string, checker Checker, status chan PrinterData, writer func(string)) {
	max := makeBigInt("904625697166532776746648320380374280100293470930272690489102837043110636675")
	var pages int64 = 0
	var startPage *big.Int
	if start != "" {
		startPage = makeBigInt(findEthPrivateKeyPage(start, 20))
	}

	for true {
		founds := 0
		pages++
		var pageNumber string

		if startPage != nil {
			pageNumber = fmt.Sprintf("%d", new(big.Int).Add(startPage, big.NewInt(pages-1)))
		} else {
			pageNumber = getRand(max).String()
		}

		ethereumKeys := generateEthereumKeys(pageNumber, 20)

		var toCheck []string

		for _, k := range ethereumKeys {
			toCheck = append(toCheck, k.public)
		}

		result, found, err := checker.CheckBSC(toCheck)

		if err != nil {
			status <- PrinterData{
				number:     id,
				found:      founds,
				pageNumber: fmt.Sprintf("%s(%d adresses)", pageNumber, 20),
				error:      err,
			}
			continue
		}

		if found {
			for _, data := range result {
				if data.Balance != "0" {
					founds++

					var private string
					for _, eKey := range ethereumKeys {
						if eKey.public == data.Account {
							private = eKey.private
						}
					}

					writer(fmt.Sprintf("%v balance | %v | %v\n",
						data.Balance, data.Account, private))
				}
			}
		}

		status <- PrinterData{
			number:     id,
			found:      founds,
			pageNumber: fmt.Sprintf("%s(%d adresses)", pageNumber, 20),
			error:      err,
		}
	}
}

type PrinterData struct {
	number     int
	found      int
	pageNumber string
	error      error
}

type workerState struct {
	pagesTotal uint64
	found      int
	errors     int
	pageNumber string
}

type Printer struct {
	mu           *sync.Mutex
	checker      Checker
	ch           chan PrinterData
	workersState workerState
	workers      int
	lastError    error
}

func (p *Printer) work() {
	go func() {
		for data := range p.ch {
			p.mu.Lock()
			p.workersState.pagesTotal += 1
			p.workersState.found += data.found
			p.workersState.pageNumber = data.pageNumber
			if data.error != nil {
				p.workersState.errors += 1
				p.lastError = data.error
			}
			p.mu.Unlock()
		}
	}()

	ticker := time.NewTicker(time.Second * 1)

	//fmt.Print("\033[s")

	for _ = range ticker.C {
		//fmt.Print("\033[u\033[K")
		p.mu.Lock()
		fmt.Printf("\r workers[%v] proxys[%v] pages=%v found=%v errors=%v pageNumber=%v",
			p.workers,
			p.checker.AvaibleProxy(),
			p.workersState.pagesTotal,
			p.workersState.found,
			p.workersState.errors,
			p.workersState.pageNumber)
		if p.lastError != nil {
			fmt.Printf(" (last error: %v)", p.lastError)
		}
		p.mu.Unlock()
	}

}
