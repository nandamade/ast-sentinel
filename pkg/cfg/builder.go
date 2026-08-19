package cfg

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

type BlockType int

const (
	BlockEntry BlockType = iota
	BlockStandard
	BlockCond
	BlockExit
)

type Node struct {
	ID   int
	Stmt ast.Stmt
}

type BasicBlock struct {
	ID           int
	Type         BlockType
	Nodes        []Node
	Predecessors []*BasicBlock
	Successors   []*BasicBlock
	Terminator   ast.Stmt
}

type ControlFlowGraph struct {
	FnDecl *ast.FuncDecl
	Entry  *BasicBlock
	Exit   *BasicBlock
	Blocks []*BasicBlock
	fset   *token.FileSet
}

type CFGBuilder struct {
	fset     *token.FileSet
	blockSeq int
	nodeSeq  int
}

func NewCFGBuilder(fset *token.FileSet) *CFGBuilder {
	return &CFGBuilder{fset: fset}
}

func (b *CFGBuilder) newBlock(btype BlockType) *BasicBlock {
	b.blockSeq++
	return &BasicBlock{
		ID:           b.blockSeq,
		Type:         btype,
		Nodes:        make([]Node, 0),
		Predecessors: make([]*BasicBlock, 0),
		Successors:   make([]*BasicBlock, 0),
	}
}

func AddEdge(from, to *BasicBlock) {
	if from == nil || to == nil {
		return
	}
	for _, succ := range from.Successors {
		if succ == to {
			return
		}
	}
	from.Successors = append(from.Successors, to)
	to.Predecessors = append(to.Predecessors, from)
}

func (b *CFGBuilder) Build(fn *ast.FuncDecl) *ControlFlowGraph {
	if fn.Body == nil {
		return nil
	}

	cfg := &ControlFlowGraph{
		FnDecl: fn,
		Entry:  b.newBlock(BlockEntry),
		Exit:   b.newBlock(BlockExit),
		Blocks: make([]*BasicBlock, 0),
		fset:   b.fset,
	}

	curr := cfg.Entry
	curr = b.buildBlock(fn.Body, curr, cfg.Exit)

	if curr != nil && curr != cfg.Exit {
		AddEdge(curr, cfg.Exit)
	}

	visited := make(map[int]bool)
	var collect func(blk *BasicBlock)
	collect = func(blk *BasicBlock) {
		if blk == nil || visited[blk.ID] {
			return
		}
		visited[blk.ID] = true
		cfg.Blocks = append(cfg.Blocks, blk)
		for _, succ := range blk.Successors {
			collect(succ)
		}
	}
	collect(cfg.Entry)
	collect(cfg.Exit)

	return cfg
}

func (b *CFGBuilder) buildBlock(stmt ast.Stmt, current *BasicBlock, exitBlock *BasicBlock) *BasicBlock {
	if current == nil {
		return nil
	}

	switch s := stmt.(type) {
	case *ast.BlockStmt:
		for _, child := range s.List {
			current = b.buildBlock(child, current, exitBlock)
		}
		return current

	case *ast.IfStmt:
		if s.Init != nil {
			current = b.buildBlock(s.Init, current, exitBlock)
		}

		condBlock := b.newBlock(BlockCond)
		if s.Cond != nil {
			b.nodeSeq++
			condBlock.Nodes = append(condBlock.Nodes, Node{ID: b.nodeSeq, Stmt: &ast.ExprStmt{X: s.Cond}})
		}
		AddEdge(current, condBlock)

		thenBlock := b.newBlock(BlockStandard)
		AddEdge(condBlock, thenBlock)
		afterThen := b.buildBlock(s.Body, thenBlock, exitBlock)

		afterElse := (*BasicBlock)(nil)
		if s.Else != nil {
			elseBlock := b.newBlock(BlockStandard)
			AddEdge(condBlock, elseBlock)
			afterElse = b.buildBlock(s.Else, elseBlock, exitBlock)
		} else {
			afterElse = condBlock
		}

		joinBlock := b.newBlock(BlockStandard)
		if afterThen != nil {
			AddEdge(afterThen, joinBlock)
		}
		if afterElse != nil && afterElse != condBlock {
			AddEdge(afterElse, joinBlock)
		} else if s.Else == nil {
			AddEdge(condBlock, joinBlock)
		}

		return joinBlock

	case *ast.ReturnStmt:
		b.nodeSeq++
		current.Nodes = append(current.Nodes, Node{ID: b.nodeSeq, Stmt: s})
		current.Terminator = s
		AddEdge(current, exitBlock)
		return nil

	default:
		b.nodeSeq++
		current.Nodes = append(current.Nodes, Node{ID: b.nodeSeq, Stmt: s})
		return current
	}
}

func (cfg *ControlFlowGraph) DotExport() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("digraph CFG_%s {\n", cfg.FnDecl.Name.Name))
	sb.WriteString("  node [shape=box, fontname=\"Courier\"];\n")

	for _, b := range cfg.Blocks {
		label := fmt.Sprintf("Block %d\\n", b.ID)
		switch b.Type {
		case BlockEntry:
			label += "[ENTRY]"
		case BlockExit:
			label += "[EXIT]"
		case BlockCond:
			label += "[CONDITION]"
		}
		for _, n := range b.Nodes {
			label += fmt.Sprintf("\\n- node_%d", n.ID)
		}
		sb.WriteString(fmt.Sprintf("  B%d [label=\"%s\"];\n", b.ID, label))
	}

	for _, b := range cfg.Blocks {
		for _, succ := range b.Successors {
			sb.WriteString(fmt.Sprintf("  B%d -> B%d;\n", b.ID, succ.ID))
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}
