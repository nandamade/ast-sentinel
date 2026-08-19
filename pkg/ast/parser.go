package ast

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
)

// ParsedPackage menyimpan syntax tree dan informasi tipe data file Go.
type ParsedPackage struct {
	Fset  *token.FileSet
	Files []*ast.File
	Types *types.Info
	Pkg   *types.Package
}

// ParseSource mengubah kode program menjadi AST serta melakukan type checking.
func ParseSource(filename string, src string) (*ParsedPackage, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("ast parse error: %w", err)
	}

	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
	}

	config := types.Config{
		Importer: importer.Default(),
		Error:    func(err error) {},
	}

	pkg, _ := config.Check("main", fset, []*ast.File{file}, info)

	return &ParsedPackage{
		Fset:  fset,
		Files: []*ast.File{file},
		Types: info,
		Pkg:   pkg,
	}, nil
}
