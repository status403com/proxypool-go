package proxypool_test

import (
	"fmt"
	"time"

	proxypool "github.com/status403com/proxypool-go"
)

func ExampleNew() {
	pool, err := proxypool.New(proxypool.Config{
		Proxies: []string{
			"1.2.3.4:8080",
			"socks5://user:pass@5.6.7.8:1080",
			"admin:secret@9.10.11.12:3128",
		},
		Protocol: proxypool.HTTP,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(pool.Len())
	// Output: 3
}

func ExampleNew_withFlagDuration() {
	pool, _ := proxypool.New(proxypool.Config{
		Proxies:      []string{"1.2.3.4:8080"},
		FlagDuration: 5 * time.Minute,
	})

	fmt.Println(pool.Len())
	// Output: 1
}

func ExampleNew_withJSONStore() {
	pool, err := proxypool.New(proxypool.Config{
		Proxies: []string{"1.2.3.4:8080", "5.6.7.8:3128"},
		Store:   proxypool.NewJSONStore("/tmp/my-proxies.json"),
	})
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	fmt.Println(pool.Len())
	// Output: 2
}
