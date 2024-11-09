package main

import (
	"context"
	"fmt"
	"github.com/startfellows/tongo"
	"github.com/startfellows/tongo/liteclient"
	"github.com/startfellows/tongo/wallet"
	"os"
	"strings"
)

func contains(s string, sl []string) bool {
	for i := range sl {
		if sl[i] == s {
			return true
		}
	}
	return false
}

func copySeed(old []string) []string {
	n := make([]string, len(old))
	copy(n, old)
	return n
}

func insertEmpty(seed []string, n int) []string {
	seed2 := copySeed(seed)
	if len(seed2) == n { // nil or empty slice or after last element
		return append(seed2, "")
	}
	seed2 = append(seed2[:n+1], seed2[n:]...) // index < len(a)
	seed2[n] = ""
	return seed2
}

func bruteforce(seed []string, wordNumber int, client *liteclient.Client) bool {
	for _, w := range wallet.WORDLIST {
		seed[wordNumber] = w
		if checkSeed(seed, client) {
			return true
		}
	}
	return false
}

var checksCounter = 0

func checkSeed(seed []string, client *liteclient.Client) bool {
	checksCounter++
	if checksCounter%1000 == 0 {
		fmt.Printf("Scanned wallets: %v\n", checksCounter)
	}
	addresses, err := toAddresses(seed)
	if err != nil {
		return false
	}

	for _, a := range addresses {
		state, err := client.GetAccountState(context.TODO(), a)
		if err != nil {
			continue
		}
		if state.Status == tongo.AccountActive || state.Status == tongo.AccountUninit {
			return true
		}
	}
	return false
}

func toAddresses(seed []string) ([]tongo.AccountID, error) {
	key, err := wallet.SeedToPrivateKey(strings.Join(seed, " "))
	if err != nil {
		return nil, err
	}
	w4, err := wallet.NewWallet(key, wallet.V4R2, 0, nil)
	if err != nil {
		return nil, err
	}
	w3, err := wallet.NewWallet(key, wallet.V3R2, 0, nil)
	if err != nil {
		return nil, err
	}
	return []tongo.AccountID{w3.GetAddress(), w4.GetAddress()}, nil
}

func recoverSeed(seedString string, client *liteclient.Client) (string, error) {
	seed := strings.Split(seedString, " ")
	if len(seed) < 23 || len(seed) > 24 {
		return "", fmt.Errorf("can not recover")
	}

	for _, word := range seed {
		if word == "0" {
			continue
		}
		if !contains(word, wallet.WORDLIST) {
			return "", fmt.Errorf("invalid word: %s", word)
		}
	}

	if len(seed) == 24 {
		valid := checkSeed(seed, client)
		if valid {
			fmt.Println("Seed is valid")
			return seedString, nil
		}
		for i := range seed {
			if seed[i] == "0" {
				seed2 := copySeed(seed)
				if bruteforce(seed2, i, client) {
					return strings.Join(seed2, " "), nil
				}
			}
		}
		return "", fmt.Errorf("can not find valid seed")
	}

	for i := 0; i < 24; i++ {
		if seed[i] == "0" {
			seed2 := insertEmpty(seed, i)
			if bruteforce(seed2, i, client) {
				return strings.Join(seed2, " "), nil
			}
		}
	}
	return "", fmt.Errorf("can not find valid seed")
}

func main() {
	client, err := liteclient.NewClientWithDefaultMainnet()
	if err != nil {
		panic(err)
	}

	data, err := os.ReadFile("seed.txt")
	if err != nil {
		panic(err)
	}

	seedString := string(data)
	seed, err := recoverSeed(seedString, client)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

	fmt.Printf("Valid seed:\n%s\n", seed)

	addresses, err := toAddresses(strings.Split(seed, " "))
	if err != nil {
		panic(err)
	}

	for _, a := range addresses {
		state, err := client.GetAccountState(context.TODO(), a)
		if err != nil {
			continue
		}
		if state.Status == tongo.AccountActive {
			fmt.Printf("Active wallet:\nAddress: %s\nVersion: %s\nInitialized: %v\nSeed: %s\n\n", a, "V4R2", state.Status == tongo.AccountUninit, seed)
		}
	}

	os.Exit(0)
}
