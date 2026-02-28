package utils

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

type imageAltOnlyRenderer struct{}

func (imageAltOnlyRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, func(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
		return ast.WalkContinue, nil
	})
}

func ToHTML(content string) (string, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.Table),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(content), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func ToHTMLPreview(content string) (string, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.Table),
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(
				util.Prioritized(imageAltOnlyRenderer{}, 0),
			),
			html.WithUnsafe(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(content), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
