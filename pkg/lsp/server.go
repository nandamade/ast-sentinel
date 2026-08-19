package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/textproto"
	"strconv"

	"github.com/sentinel-analyzer/ast-sentinel/pkg/analyzer"
	past "github.com/sentinel-analyzer/ast-sentinel/pkg/ast"
)

type Request struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  interface{}      `json:"result,omitempty"`
}

type Notification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type PublishDiagnosticsParams struct {
	URI         string    `json:"uri"`
	Diagnostics []LSPDiag `json:"diagnostics"`
}

type LSPDiag struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Source   string `json:"source"`
	Message  string `json:"message"`
	Code     string `json:"code"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type LSPServer struct {
	engine *analyzer.Engine
	reader *bufio.Reader
	writer io.Writer
}

func NewLSPServer(r io.Reader, w io.Writer) *LSPServer {
	return &LSPServer{
		engine: analyzer.NewEngine(),
		reader: bufio.NewReader(r),
		writer: w,
	}
}

func (s *LSPServer) Start() {
	tp := textproto.NewReader(s.reader)
	for {
		header, err := tp.ReadMIMEHeader()
		if err != nil {
			break
		}
		lengthStr := header.Get("Content-Length")
		if lengthStr == "" {
			continue
		}
		length, _ := strconv.Atoi(lengthStr)
		body := make([]byte, length)
		_, err = io.ReadFull(s.reader, body)
		if err != nil {
			continue
		}

		var req Request
		json.Unmarshal(body, &req)
		s.handleRequest(&req)
	}
}

func (s *LSPServer) handleRequest(req *Request) {
	switch req.Method {
	case "initialize":
		s.sendResponse(req.ID, map[string]interface{}{
			"capabilities": map[string]interface{}{
				"textDocumentSync": 1,
			},
		})
	case "textDocument/didOpen", "textDocument/didChange", "textDocument/didSave":
		var params struct {
			TextDocument struct {
				URI  string `json:"uri"`
				Text string `json:"text"`
			} `json:"textDocument"`
		}
		if err := json.Unmarshal(req.Params, &params); err == nil && params.TextDocument.Text != "" {
			s.analyzeAndPublish(params.TextDocument.URI, params.TextDocument.Text)
		}
	}
}

func (s *LSPServer) analyzeAndPublish(uri string, source string) {
	parsedPkg, err := past.ParseSource("main.go", source)
	if err != nil {
		return
	}

	diags := s.engine.Run(parsedPkg)
	lspDiags := make([]LSPDiag, 0)

	for _, d := range diags {
		sev := 2
		if d.Severity == analyzer.SevHigh {
			sev = 1
		}
		lspDiags = append(lspDiags, LSPDiag{
			Range: Range{
				Start: Position{Line: d.Line - 1, Character: d.Column - 1},
				End:   Position{Line: d.Line - 1, Character: d.Column + 10},
			},
			Severity: sev,
			Source:   "AST-Sentinel",
			Message:  d.Message,
			Code:     d.RuleID,
		})
	}

	s.sendNotification(Notification{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params:  PublishDiagnosticsParams{URI: uri, Diagnostics: lspDiags},
	})
}

func (s *LSPServer) sendResponse(id *json.RawMessage, result interface{}) {
	s.writeJSON(Response{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *LSPServer) sendNotification(notif Notification) {
	s.writeJSON(notif)
}

func (s *LSPServer) writeJSON(v interface{}) {
	data, _ := json.Marshal(v)
	msg := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data)
	fmt.Fprint(s.writer, msg)
}
