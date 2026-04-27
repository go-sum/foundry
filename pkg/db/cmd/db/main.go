package main

import (
	"fmt"
	"os"

	dbcli "github.com/go-sum/foundry/pkg/db/cli"
)

func main() {
	if err := dbcli.NewRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
