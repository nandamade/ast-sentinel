package main

import (
	"os"

	"github.com/sentinel-analyzer/ast-sentinel/pkg/lsp"
)

func main() {
	server := lsp.NewLSPServer(os.Stdin, os.Stdout)
	server.Start()
}
