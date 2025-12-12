package main

import (
	"fmt"

	"github.com/gamassss/url-shortener/pkg/generator"
)

func main() {
	for _ = range 10 {
		fmt.Println(generator.GenerateShortCode())
	}
}
