package pipeline

import (
	"strings"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// AdmonitionKind identifies a supported admonition type.
type AdmonitionKind int

// Supported admonition kinds.
const (
	AdmonitionNote AdmonitionKind = iota
	AdmonitionTip
	AdmonitionImportant
	AdmonitionWarning
	AdmonitionCaution
)

// kindDefinition holds the canonical label, CSS class, and default title
// of an admonition kind. Index matches the AdmonitionKind value.
type kindDefinition struct {
	name         string
	className    string
	defaultTitle string
	aliases      []string
}

var kindDefinitions = []kindDefinition{
	{name: "note", className: "admonition-note", defaultTitle: "Note", aliases: []string{"info"}},
	{name: "tip", className: "admonition-tip", defaultTitle: "Tip", aliases: []string{"success"}},
	{name: "important", className: "admonition-important", defaultTitle: "Important"},
	{name: "warning", className: "admonition-warning", defaultTitle: "Warning"},
	{name: "caution", className: "admonition-caution", defaultTitle: "Caution", aliases: []string{"danger"}},
}

// kindDefinitionFor returns the definition of a kind, or false when the
// value is outside the canonical range.
func kindDefinitionFor(kind AdmonitionKind) (kindDefinition, bool) {
	if kind < 0 || int(kind) >= len(kindDefinitions) {
		return kindDefinition{}, false
	}
	return kindDefinitions[kind], true
}

// String returns the canonical lowercase label, or an empty string for an
// invalid kind.
func (k AdmonitionKind) String() string {
	def, ok := kindDefinitionFor(k)
	if !ok {
		return ""
	}
	return def.name
}

// ClassName returns the CSS class, or an empty string for an invalid kind.
func (k AdmonitionKind) ClassName() string {
	def, ok := kindDefinitionFor(k)
	if !ok {
		return ""
	}
	return def.className
}

// DefaultTitle returns the fixed English title, or an empty string for an
// invalid kind.
func (k AdmonitionKind) DefaultTitle() string {
	def, ok := kindDefinitionFor(k)
	if !ok {
		return ""
	}
	return def.defaultTitle
}

// parseAdmonitionKind resolves a canonical or aliased label. Comparison is
// case-insensitive. The caller trims surrounding whitespace.
func parseAdmonitionKind(label string) (AdmonitionKind, bool) {
	normalized := strings.ToLower(label)
	for i, def := range kindDefinitions {
		if def.name == normalized {
			return AdmonitionKind(i), true
		}
		for _, alias := range def.aliases {
			if alias == normalized {
				return AdmonitionKind(i), true
			}
		}
	}
	return AdmonitionNote, false
}

// Admonition is a block node for a note, tip, important, warning, or caution.
type Admonition struct {
	gast.BaseBlock
	Variant AdmonitionKind
	Title   string
}

// KindAdmonition is the node kind of Admonition.
var KindAdmonition = gast.NewNodeKind("Admonition")

// Kind implements gast.Node.Kind.
func (n *Admonition) Kind() gast.NodeKind {
	return KindAdmonition
}

// Dump implements gast.Node.Dump.
func (n *Admonition) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Kind":  n.Variant.String(),
		"Title": n.Title,
	}, nil)
}

// NewAdmonition returns an Admonition node. An empty title selects the
// default title at render time.
func NewAdmonition(kind AdmonitionKind, title string) *Admonition {
	return &Admonition{Variant: kind, Title: title}
}

// admonitionHTMLRenderer renders Admonition nodes as semantic HTML boxes.
type admonitionHTMLRenderer struct {
	html.Config
}

// NewAdmonitionHTMLRenderer returns a node renderer for Admonition nodes.
func NewAdmonitionHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &admonitionHTMLRenderer{Config: html.NewConfig()}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

// RegisterFuncs implements renderer.NodeRenderer.
func (r *admonitionHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindAdmonition, r.renderAdmonition)
}

// renderAdmonition writes the admonition box. The title is escaped and is
// never treated as inline Markdown.
func (r *admonitionHTMLRenderer) renderAdmonition(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		_, _ = w.WriteString("</div>\n")
		return gast.WalkContinue, nil
	}

	node := n.(*Admonition)
	title := node.Title
	if title == "" {
		title = node.Variant.DefaultTitle()
	}

	_, _ = w.WriteString(`<div class="admonition `)
	_, _ = w.WriteString(node.Variant.ClassName())
	_, _ = w.WriteString("\" role=\"note\">\n")
	_, _ = w.WriteString(`<p class="admonition-title">`)
	_, _ = w.Write(util.EscapeHTML([]byte(title)))
	_, _ = w.WriteString("</p>\n")
	return gast.WalkContinue, nil
}

// AdmonitionExtension adds admonition rendering to Goldmark. Parsers are
// registered separately.
var AdmonitionExtension = &admonitionExtension{}

type admonitionExtension struct{}

// Extend implements goldmark.Extender.
func (e *admonitionExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithASTTransformers(
		util.Prioritized(NewAdmonitionQuoteTransformer(), 100),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewAdmonitionHTMLRenderer(), 500),
	))
}
