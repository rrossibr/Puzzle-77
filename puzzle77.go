package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"math/big"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"golang.org/x/crypto/ripemd160"
)

var (
	startHex      = "400000000000000000"
	endHex        = "7fffffffffffffffff"
	targetAddress = "1PWo3JeB9jrGwfHDNpdGK54CRas7fsVzXU"
)

// Base58
var b58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

func base58Encode(input []byte) []byte {
	var result []byte
	x := new(big.Int).SetBytes(input)
	base := big.NewInt(58)
	mod := new(big.Int)
	zero := big.NewInt(0)

	for x.Cmp(zero) > 0 {
		x.DivMod(x, base, mod)
		result = append(result, b58Alphabet[mod.Int64()])
	}

	// reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	for _, b := range input {
		if b == 0x00 {
			result = append([]byte{b58Alphabet[0]}, result...)
		} else {
			break
		}
	}

	return result
}

func hash160(data []byte) []byte {
	h1 := sha256.Sum256(data)
	r := ripemd160.New()
	r.Write(h1[:])
	return r.Sum(nil)
}

func checksum(data []byte) []byte {
	h1 := sha256.Sum256(data)
	h2 := sha256.Sum256(h1[:])
	return h2[:4]
}

func privKeyToWIF(priv *btcec.PrivateKey, compressed bool) string {
	privBytes := priv.Serialize()
	payload := []byte{0x80}
	payload = append(payload, privBytes...)
	if compressed {
		payload = append(payload, 0x01)
	}
	full := append(payload, checksum(payload)...)
	return string(base58Encode(full))
}

func privKeyToAddress(priv *btcec.PrivateKey) string {
	pubKey := priv.PubKey()
	pubKeyHash := hash160(pubKey.SerializeCompressed())
	payload := append([]byte{0x00}, pubKeyHash...) // 0x00 = mainnet
	full := append(payload, checksum(payload)...)
	return string(base58Encode(full))
}

func printGlobalProgress(done, total *big.Int) {
	percent := new(big.Float).Quo(new(big.Float).SetInt(done), new(big.Float).SetInt(total))
	percentF, _ := percent.Float64()

	barWidth := 40
	filled := int(percentF * float64(barWidth))

	bar := strings.Repeat("█", filled) + strings.Repeat("-", barWidth-filled)
	fmt.Printf("\r🧮 Testadas: %s / %s  [%s] %5.2f%%",
		done.String(), total.String(), bar, percentF*100)
}

func worker(id int, wg *sync.WaitGroup, start, end *big.Int, found *uint32, globalCount *big.Int, mu *sync.Mutex) {
	defer wg.Done()
	one := big.NewInt(1)
	k := new(big.Int).Set(start)

	for k.Cmp(end) <= 0 {
		if atomic.LoadUint32(found) == 1 {
			return
		}

		privBytes := k.Bytes()
		if len(privBytes) < 32 {
			pad := make([]byte, 32-len(privBytes))
			privBytes = append(pad, privBytes...)
		}

		privKey, _ := btcec.PrivKeyFromBytes(privBytes)
		addr := privKeyToAddress(privKey)

		mu.Lock()
		globalCount.Add(globalCount, one)
		mu.Unlock()

		if addr == targetAddress {
			atomic.StoreUint32(found, 1)
			wif := privKeyToWIF(privKey, true)
			fmt.Printf("\n\n🎯 ENCONTRADO pelo worker %d\n", id)
			fmt.Println("Endereço:", addr)
			fmt.Println("WIF:", wif)
			fmt.Println("Chave privada:", hex.EncodeToString(privBytes))
			os.Exit(0)
		}

		k.Add(k, one)
	}
}

func main() {
	// Argumento para número de CPUs
	cores := flag.Int("cores", runtime.NumCPU(), "Número de núcleos a usar")
	flag.Parse()

	runtime.GOMAXPROCS(*cores)
	fmt.Printf("🚀 Iniciando varredura com %d núcleos\n", *cores)

	startInt := new(big.Int)
	endInt := new(big.Int)
	startInt.SetString(startHex, 16)
	endInt.SetString(endHex, 16)

	totalRange := new(big.Int).Sub(endInt, startInt)
	totalRange.Add(totalRange, big.NewInt(1))

	chunkSize := new(big.Int).Div(totalRange, big.NewInt(int64(*cores)))

	var wg sync.WaitGroup
	var mu sync.Mutex
	var found uint32
	globalCount := big.NewInt(0)

	// Barra de progresso global
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			mu.Lock()
			current := new(big.Int).Set(globalCount)
			mu.Unlock()
			printGlobalProgress(current, totalRange)
		}
	}()

	startTime := time.Now()

	for i := 0; i < *cores; i++ {
		wg.Add(1)
		workerStart := new(big.Int).Add(startInt, new(big.Int).Mul(chunkSize, big.NewInt(int64(i))))
		workerEnd := new(big.Int).Sub(new(big.Int).Add(workerStart, chunkSize), big.NewInt(1))
		if i == *cores-1 {
			workerEnd = endInt
		}
		go worker(i, &wg, workerStart, workerEnd, &found, globalCount, &mu)
	}

	wg.Wait()

	fmt.Println("\n🏁 Finalizado em", time.Since(startTime))
}
