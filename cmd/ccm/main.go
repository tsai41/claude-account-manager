package main

import (
	"fmt"
	"os"

	"github.com/tsai41/claude-account-manager/internal/cli"
)

func main() {
	if err := cli.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
