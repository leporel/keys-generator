# Keys.lol generator
This repository contains the key generator for [Keys.lol](https://keys.lol)

## Building and installing
1. cd to `~/go/src/github.com/sjorso/keys-generator`
2. install required packages with `go get`
3. build the executable with `go build`
4. include the executable in `$PATH`: `sudo cp keys-generator /usr/local/bin`

## Usage
For generating keys, run:

```bash
keys-generator btc <page number>
keys-generator eth <page number>
```

For searching by private key, run:
```bash
keys-generator btc-search <btc private key>
keys-generator eth-search <eth private key>
```

For brute by pages, run:
```bash
keys-generator btc-brute <number of workers> 
keys-generator eth-brute <number of workers> <ETHERSCAN_API_KEY>
keys-generator bsc-brute <number of workers> <BINANCE_API_KEY>
keys-generator bsc-brute <number of workers> <BINANCE_API_KEY> <FROM_PRIVATE_ADDRESS>

# support multiple api keys
keys-generator eth-brute <number of workers> <ETHERSCAN_API_KEY>,<ETHERSCAN_API_KEY>,<ETHERSCAN_API_KEY>
```

Btc api request rate limit is 50 / minute  
Eth api without ETHERSCAN_API_KEY limit is 1 request per 5 seconds, with api key 270 per minute

To increase the req/min limits, use a list of proxies, add them to proxy.txt  
```
user:pass@127.0.0.1:777
192.0.0.1:5555
```

## License

This project is open-sourced software licensed under the [MIT license](http://opensource.org/licenses/MIT)
