// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package templscan

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/a-h/templ/parser/v2"
)

const templFileExtension = ".templ"

type AttributeKind string

const (
	AttributeKindConstant   AttributeKind = "constant"
	AttributeKindExpression AttributeKind = "expression"
	AttributeKindBool       AttributeKind = "bool"
)

type File struct {
	Path       string
	Attributes []Attribute
}

type Attribute struct {
	Element string
	Name    string
	Kind    AttributeKind
	Value   string
	Line    int
	Column  int
}

func ScanDir(root string) ([]File, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan templ root %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scan templ root %q: not a directory", root)
	}

	var files []File
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case "testdata", "vendor":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		if filepath.Ext(entry.Name()) != templFileExtension {
			return nil
		}

		file, err := ParseFile(path)
		if err != nil {
			return err
		}
		files = append(files, file)
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(files, func(a, b File) int {
		return strings.Compare(a.Path, b.Path)
	})
	return files, nil
}

func ParseFile(path string) (File, error) {
	templateFile, err := parser.Parse(path)
	if err != nil {
		return File{}, fmt.Errorf("parse %s: %w", path, err)
	}

	scanner := scanner{path: path}
	for _, node := range templateFile.Nodes {
		scanner.scanTemplateFileNode(node)
	}
	return File{Path: path, Attributes: scanner.attributes}, nil
}

type scanner struct {
	path       string
	attributes []Attribute
}

func (scanner *scanner) scanTemplateFileNode(node parser.TemplateFileNode) {
	switch node := node.(type) {
	case *parser.HTMLTemplate:
		scanner.scanNodes(node.Children)
	}
}

func (scanner *scanner) scanNodes(nodes []parser.Node) {
	for _, node := range nodes {
		scanner.scanNode(node)
	}
}

func (scanner *scanner) scanNode(node parser.Node) {
	switch node := node.(type) {
	case *parser.Element:
		scanner.scanAttributes(node.Name, node.Attributes)
		scanner.scanNodes(node.Children)
	case *parser.RawElement:
		scanner.scanAttributes(node.Name, node.Attributes)
	case *parser.ScriptElement:
		scanner.scanAttributes("script", node.Attributes)
	case *parser.IfExpression:
		scanner.scanNodes(node.Then)
		for _, elseIf := range node.ElseIfs {
			scanner.scanNodes(elseIf.Then)
		}
		scanner.scanNodes(node.Else)
	case *parser.ForExpression:
		scanner.scanNodes(node.Children)
	case *parser.SwitchExpression:
		for _, switchCase := range node.Cases {
			scanner.scanNodes(switchCase.Children)
		}
	case *parser.TemplElementExpression:
		scanner.scanNodes(node.Children)
	}
}

func (scanner *scanner) scanAttributes(element string, attributes []parser.Attribute) {
	for _, attribute := range attributes {
		scanner.scanAttribute(element, attribute)
	}
}

func (scanner *scanner) scanAttribute(element string, attribute parser.Attribute) {
	switch attribute := attribute.(type) {
	case *parser.ConstantAttribute:
		scanner.appendAttribute(element, attribute.Key.String(), AttributeKindConstant, attribute.Value, attribute.Range.From)
	case *parser.ExpressionAttribute:
		scanner.appendAttribute(element, attribute.Key.String(), AttributeKindExpression, attribute.Expression.Value, attribute.Range.From)
	case *parser.BoolConstantAttribute:
		scanner.appendAttribute(element, attribute.Key.String(), AttributeKindBool, "", attribute.Range.From)
	case *parser.BoolExpressionAttribute:
		scanner.appendAttribute(element, attribute.Key.String(), AttributeKindExpression, attribute.Expression.Value, attribute.Range.From)
	case *parser.ConditionalAttribute:
		for _, thenAttribute := range attribute.Then {
			scanner.scanAttribute(element, thenAttribute)
		}
		for _, elseAttribute := range attribute.Else {
			scanner.scanAttribute(element, elseAttribute)
		}
	}
}

func (scanner *scanner) appendAttribute(element string, name string, kind AttributeKind, value string, position parser.Position) {
	scanner.attributes = append(scanner.attributes, Attribute{
		Element: element,
		Name:    name,
		Kind:    kind,
		Value:   strings.TrimSpace(value),
		Line:    int(position.Line),
		Column:  int(position.Col),
	})
}
