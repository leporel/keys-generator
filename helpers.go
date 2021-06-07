package main

import (
	"bufio"
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"
)

var one = big.NewInt(1)

func makeBigInt(number string) *big.Int {
	i, success := new(big.Int).SetString(number, 10)

	if !success {
		log.Fatal("Failed to create BigInt from string")
	}

	return i
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

type RoundRobin interface {
	Next() (rbElement, error)
	Len() int
	Add(elem rbElement)
	Delete(elem rbElement) bool
}

type roundrobin struct {
	list []rbElement
	next int
	mu   *sync.Mutex
}

func roundRobinNew(List []rbElement) RoundRobin {
	return &roundrobin{
		list: List,
		mu:   &sync.Mutex{},
	}
}

func (r *roundrobin) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.list)
}

func (r *roundrobin) Add(elem rbElement) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.list = append(r.list, elem)
	return
}

func (r *roundrobin) Delete(elem rbElement) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, element := range r.list {
		if element == elem {
			r.list = append(r.list[:i], r.list[i+1:]...)
			return true
		}
	}
	return false
}

func (r *roundrobin) Next() (rbElement, error) {
	if len(r.list) == 0 {
		return nil, fmt.Errorf("elements dose not exist")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	if r.next == len(r.list) {
		r.next = 1
	}

	return r.list[(int(r.next)-1)%len(r.list)], nil
}
