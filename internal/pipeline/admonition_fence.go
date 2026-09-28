package pipeline

import (
	"bytes"
	"strings"

	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

const admonitionFenceMinLength = 3

// NewAdmonitionFenceParser returns a block parser for ::: admonition fences.
func NewAdmonitionFenceParser() parser.BlockParser {
	return &admonitionFenceParser{}
}

type admonitionFenceParser struct{}

// Trigger implements parser.BlockParser.
func (b *admonitionFenceParser) Trigger() []byte {
	return []byte{':'}
}

// Open implements parser.BlockParser. A fence opens on a colon run of three
// or more, followed by a space and a known type.
func (b *admonitionFenceParser) Open(parent gast.Node, reader text.Reader, pc parser.Context) (gast.Node, parser.State) {
	line, _ := reader.PeekLine()
	_, pos := util.IndentWidth(line, reader.LineOffset())
	kind, title, ok := parseFenceOpenLine(line, pos)
	if !ok {
		return nil, parser.NoChildren
	}
	return NewAdmonition(kind, title), parser.NoChildren
}

// Continue implements parser.BlockParser. Only the innermost open fence may
// close, so outer fences keep their content around inner fences. Open
// paragraphs do not block a closing line.
func (b *admonitionFenceParser) Continue(node gast.Node, reader text.Reader, pc parser.Context) parser.State {
	// Paragraphs are skipped so a closing fence terminates an open
	// paragraph instead of becoming lazy continuation.
	blocks := pc.OpenedBlocks()
	innermost := node
	for i := len(blocks) - 1; i >= 0; i-- {
		if !gast.IsParagraph(blocks[i].Node) {
			innermost = blocks[i].Node
			break
		}
	}
	if innermost != node {
		return parser.Continue | parser.HasChildren
	}

	line, _ := reader.PeekLine()
	w, pos := util.IndentWidth(line, reader.LineOffset())
	if w <= 3 && isFenceCloseLine(line, pos) {
		reader.AdvanceToEOL()
		return parser.Close
	}
	return parser.Continue | parser.HasChildren
}

// Close implements parser.BlockParser.
func (b *admonitionFenceParser) Close(node gast.Node, reader text.Reader, pc parser.Context) {
}

// CanInterruptParagraph implements parser.BlockParser.
func (b *admonitionFenceParser) CanInterruptParagraph() bool {
	return false
}

// CanAcceptIndentedLine implements parser.BlockParser.
func (b *admonitionFenceParser) CanAcceptIndentedLine() bool {
	return false
}

// parseFenceOpenLine reports the kind and title when the line opens a fence.
// A space is required between the colon run and the type.
func parseFenceOpenLine(line []byte, offset int) (AdmonitionKind, string, bool) {
	line = line[offset:]
	colons := 0
	for colons < len(line) && line[colons] == ':' {
		colons++
	}
	if colons < admonitionFenceMinLength {
		return AdmonitionNote, "", false
	}

	rest := line[colons:]
	if len(rest) == 0 || (rest[0] != ' ' && rest[0] != '\t') {
		return AdmonitionNote, "", false
	}
	rest = bytes.TrimSpace(rest)
	if len(rest) == 0 {
		return AdmonitionNote, "", false
	}

	label := rest
	title := ""
	if idx := bytes.IndexAny(rest, " \t"); idx >= 0 {
		label = rest[:idx]
		title = strings.TrimSpace(string(rest[idx+1:]))
	}
	kind, ok := parseAdmonitionKind(string(label))
	if !ok {
		return AdmonitionNote, "", false
	}
	return kind, title, true
}

// isFenceCloseLine reports whether the line is a closing fence: a colon run
// of three or more followed only by whitespace.
func isFenceCloseLine(line []byte, offset int) bool {
	line = line[offset:]
	colons := 0
	for colons < len(line) && line[colons] == ':' {
		colons++
	}
	if colons < admonitionFenceMinLength {
		return false
	}
	return util.IsBlank(line[colons:])
}
