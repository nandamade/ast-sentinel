package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"

	past "github.com/sentinel-analyzer/ast-sentinel/pkg/ast"
	"github.com/sentinel-analyzer/ast-sentinel/pkg/cfg"
)

type Severity string

const (
	SevHigh   Severity = "HIGH"
	SevMedium Severity = "MEDIUM"
	SevLow    Severity = "LOW"
)

type Diagnostic struct {
	RuleID     string           `json:"rule_id"`
	Severity   Severity         `json:"severity"`
	Message    string           `json:"message"`
	Line       int              `json:"line"`
	Column     int              `json:"column"`
	Filename   string           `json:"filename"`
	QuickFixes []QuickFixAction `json:"quick_fixes,omitempty"`
}

type QuickFixAction struct {
	Title       string `json:"title"`
	NewText     string `json:"new_text"`
	StartOffset int    `json:"start_offset"`
	EndOffset   int    `json:"end_offset"`
}

type Rule interface {
	ID() string
	Description() string
	Analyze(pkg *past.ParsedPackage) []Diagnostic
}

type Engine struct {
	rules []Rule
}

func NewEngine() *Engine {
	return &Engine{
		rules: []Rule{
			&SQLInjectionRule{},
			&MutexDeadlockRule{},
			&UncheckedErrorRule{},
		},
	}
}

func (e *Engine) Run(pkg *past.ParsedPackage) []Diagnostic {
	var diagnostics []Diagnostic
	for _, rule := range e.rules {
		diagnostics = append(diagnostics, rule.Analyze(pkg)...)
	}
	return diagnostics
}

// SEC-001: SQL Injection via Sprintf
type SQLInjectionRule struct{}

func (r *SQLInjectionRule) ID() string { return "SEC-001" }
func (r *SQLInjectionRule) Description() string {
	return "SQL Injection vulnerability through dynamic Sprintf formatting"
}

func (r *SQLInjectionRule) Analyze(pkg *past.ParsedPackage) []Diagnostic {
	var results []Diagnostic
	for _, file := range pkg.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "Query" || sel.Sel.Name == "Exec" || sel.Sel.Name == "QueryRow" {
					if len(call.Args) > 0 {
						if innerCall, ok := call.Args[0].(*ast.CallExpr); ok {
							if innerSel, ok := innerCall.Fun.(*ast.SelectorExpr); ok {
								if innerSel.Sel.Name == "Sprintf" {
									pos := pkg.Fset.Position(call.Pos())
									results = append(results, Diagnostic{
										RuleID:   r.ID(),
										Severity: SevHigh,
										Message:  fmt.Sprintf("Potential SQL Injection detected: Format string query passed to %s. Use parameterized placeholders ($1, ?) instead.", sel.Sel.Name),
										Line:     pos.Line,
										Column:   pos.Column,
										Filename: pos.Filename,
									})
								}
							}
						}
					}
				}
			}
			return true
		})
	}
	return results
}

// CONC-001: Mutex Deadlock via Double Lock
type MutexDeadlockRule struct{}

func (r *MutexDeadlockRule) ID() string { return "CONC-001" }
func (r *MutexDeadlockRule) Description() string {
	return "Deadlock caused by consecutive mutex Lock() without Unlock()"
}

func (r *MutexDeadlockRule) Analyze(pkg *past.ParsedPackage) []Diagnostic {
	var results []Diagnostic
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			builder := cfg.NewCFGBuilder(pkg.Fset)
			graph := builder.Build(fn)
			if graph == nil {
				continue
			}

			for _, block := range graph.Blocks {
				lockedMutexes := make(map[string]token.Pos)
				for _, node := range block.Nodes {
					exprStmt, ok := node.Stmt.(*ast.ExprStmt)
					if !ok {
						continue
					}
					call, ok := exprStmt.X.(*ast.CallExpr)
					if !ok {
						continue
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						continue
					}
					mutexIdent, ok := sel.X.(*ast.Ident)
					if !ok {
						continue
					}

					if sel.Sel.Name == "Lock" {
						if prevPos, exists := lockedMutexes[mutexIdent.Name]; exists {
							pos := pkg.Fset.Position(call.Pos())
							prevPosStruct := pkg.Fset.Position(prevPos)
							results = append(results, Diagnostic{
								RuleID:   r.ID(),
								Severity: SevHigh,
								Message:  fmt.Sprintf("Potential Deadlock: Mutex '%s' locked again at line %d without unlocking (previously locked at line %d)", mutexIdent.Name, pos.Line, prevPosStruct.Line),
								Line:     pos.Line,
								Column:   pos.Column,
								Filename: pos.Filename,
							})
						} else {
							lockedMutexes[mutexIdent.Name] = call.Pos()
						}
					} else if sel.Sel.Name == "Unlock" {
						delete(lockedMutexes, mutexIdent.Name)
					}
				}
			}
		}
	}
	return results
}

// ERR-001: Unchecked Error Suppression
type UncheckedErrorRule struct{}

func (r *UncheckedErrorRule) ID() string { return "ERR-001" }
func (r *UncheckedErrorRule) Description() string {
	return "Error explicitly ignored via blank identifier '_'"
}

func (r *UncheckedErrorRule) Analyze(pkg *past.ParsedPackage) []Diagnostic {
	var results []Diagnostic
	for _, file := range pkg.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for i, lhs := range assign.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok && ident.Name == "_" {
					if i < len(assign.Rhs) {
						if _, ok := assign.Rhs[i].(*ast.CallExpr); ok {
							pos := pkg.Fset.Position(assign.Pos())
							results = append(results, Diagnostic{
								RuleID:   r.ID(),
								Severity: SevMedium,
								Message:  "Explicit error suppression using '_'. Handle the error or return it.",
								Line:     pos.Line,
								Column:   pos.Column,
								Filename: pos.Filename,
								QuickFixes: []QuickFixAction{
									{
										Title:       "Replace '_' with 'err'",
										NewText:     "err",
										StartOffset: int(ident.Pos()),
										EndOffset:   int(ident.End()),
									},
								},
							})
						}
					}
				}
			}
			return true
		})
	}
	return results
}
