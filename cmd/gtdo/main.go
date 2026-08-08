// Command gtdo is a Go port of the todo.txt-cli (todo.sh) command line
// interface, with byte-identical output for the in-scope actions.
package main

import (
	"os"
	"path/filepath"

	"github.com/guerrero/gtdo/internal/cli"
)

func main() {
	cli.ProgName = filepath.Base(os.Args[0])
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
