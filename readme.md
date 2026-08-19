sentinel - minimalist go static analyzer and lsp server
=====================================================
sentinel is a fast, lightweight, and zero-bloat static analysis engine
and Language Server Protocol (LSP) daemon written in Go. It operates directly
on Abstract Syntax Trees (AST) and constructs Control Flow Graphs (CFG) to
detect security vulnerabilities, concurrency deadlocks, and bad patterns.

It follows the suckless philosophy: simple, clear, modular, and hackable.


Features
--------
- AST & Type Resolution: Direct AST traversal using go/ast and go/types.
- Control Flow Graph (CFG): Generates basic execution blocks for functions.
- Concurrency Deadlock Analysis: Detects double locking on sync.Mutex.
- Security Checks: Detects SQL Injections via dynamic format strings.
- LSP Server: Standard Input/Output JSON-RPC 2.0 interface for vim/neovim.
- CFG Exporter: Outputs Graphviz DOT format for visual graph analysis.


Architecture
------------
- pkg/ast      : Parser and type check bindings.
- pkg/cfg      : Basic block graph builder and DOT exporter.
- pkg/analyzer : Dataflow analysis rules (SEC-001, CONC-001, ERR-001).
- pkg/lsp      : Stdio JSON-RPC 2.0 protocol handler.
- cmd/sentinel : CLI binary.
- cmd/sentinel-lsp : Language Server binary.


Requirements
------------
In order to build sentinel you need:
- Go 1.22 or newer
- POSIX-compliant system (Linux/BSD/macOS)


Installation
------------
Build the binaries:

    $ go build -o bin/sentinel ./cmd/sentinel
    $ go build -o bin/sentinel-lsp ./cmd/sentinel-lsp

Optionally install into PATH:

    # cp bin/sentinel /usr/local/bin/
    # cp bin/sentinel-lsp /usr/local/bin/


Usage
-----
Run static analysis on a target file:

    $ sentinel -file main.go

Output formatted JSON report:

    $ sentinel -file main.go -json

Export Control Flow Graph in Graphviz DOT format:

    $ sentinel -file main.go -cfg > main.dot
    $ dot -Tpng main.dot -o main.png


LSP Integration (Neovim)
------------------------
Add the following snippet to your neovim lspconfig lua setup:

    local configs = require('lspconfig.configs')
    local lspconfig = require('lspconfig')

    if not configs.sentinel then
      configs.sentinel = {
        default_config = {
          cmd = { 'sentinel-lsp' },
          filetypes = { 'go' },
          root_dir = lspconfig.util.root_pattern('go.mod', '.git'),
        },
      }
    end

    lspconfig.sentinel.setup{}


Rules
-----
SEC-001  High    SQL Injection via dynamic Sprintf inside db.Query/Exec
CONC-001 High    Deadlock caused by consecutive Mutex.Lock()
ERR-001  Medium  Explicit error suppression using '_' identifier

### Forking & Local Development

If you fork this repository, update the module path in `go.mod` and internal imports to match your GitHub username:

    $ go mod edit -module github.com/YOUR_USERNAME/ast-sentinel
    $ find . -type f -name "*.go" -exec sed -i 's|github.com/ORIGINAL_OWNER/ast-sentinel|github.com/YOUR_USERNAME/ast-sentinel|g' {} +
    $ go mod tidy
