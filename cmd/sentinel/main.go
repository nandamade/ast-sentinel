package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"io/ioutil"
	"os"

	"github.com/sentinel-analyzer/ast-sentinel/pkg/analyzer"
	past "github.com/sentinel-analyzer/ast-sentinel/pkg/ast"
	"github.com/sentinel-analyzer/ast-sentinel/pkg/cfg"
)

func main() {
	filePath := flag.String("file", "", "Path file Go yang akan dianalisis")
	jsonOutput := flag.Bool("json", false, "Output format JSON")
	exportCFG := flag.Bool("cfg", false, "Export CFG ke format Graphviz DOT")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("Error: Parameter -file wajib diisi")
		os.Exit(1)
	}

	content, err := ioutil.ReadFile(*filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	parsedPkg, err := past.ParseSource(*filePath, string(content))
	if err != nil {
		fmt.Printf("Error parsing source: %v\n", err)
		os.Exit(1)
	}

	if *exportCFG {
		builder := cfg.NewCFGBuilder(parsedPkg.Fset)
		for _, file := range parsedPkg.Files {
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok {
					graph := builder.Build(fn)
					if graph != nil {
						fmt.Println(graph.DotExport())
					}
				}
			}
		}
		return
	}

	engine := analyzer.NewEngine()
	diagnostics := engine.Run(parsedPkg)

	if *jsonOutput {
		data, _ := json.MarshalIndent(diagnostics, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("--- AST-Sentinel Analysis Report [%d issues] ---\n", len(diagnostics))
		for _, d := range diagnostics {
			fmt.Printf("[%s] [%s] %s:%d:%d -> %s\n", d.Severity, d.RuleID, d.Filename, d.Line, d.Column, d.Message)
		}
	}
}
