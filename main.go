package main

import (
	"github.com/tokend/erc20-withdraw-svc/internal/cli"
	"os"
)

func main() {
	if !cli.Run(os.Args) {
		os.Exit(1)
	}
}
