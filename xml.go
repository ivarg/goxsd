// Things not yet implemented:
// - enforcing use="restricted" on attributes
// - namespaces

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	xsdFile string
	usage   = `Usage:

  goxsd <xsd>

Arguments:

  xsd     Path to a valid XSD file

goxsd is a tool for generating XML decoding Go structs, according to an XSD
schema.

The argument is expected to be the path to a valid XSD schema file. Any import
statements in that file will be be followed and parsed. The resulting set of
Go structs will be printed on stdout.
`
)

func main() {
	flag.StringVar(&xsdFile, "xsd", "", "Path to an XSD file")
	flag.Parse()

	if xsdFile == "" {
		fmt.Println(usage)
		os.Exit(1)
	}

	s, err := extractSchemas(xsdFile)
	if err != nil {
		log.Fatal(err)
	}
	builder := newXmlBuilder(s)
	parse(os.Stdout, builder.buildXml())
}

type xmlElem struct {
	Name     string
	Type     string
	List     bool
	Value    string
	Attribs  []xmlAttrib
	Children []*xmlElem
}

type xmlAttrib struct {
	Name string
	Type string
}

type xmlBuilder struct {
	schemas    []xsdSchema
	complTypes map[string]xsdComplexType
	simplTypes map[string]xsdSimpleType
}

func newXmlBuilder(s []xsdSchema) xmlBuilder {
	return xmlBuilder{
		schemas:    s,
		complTypes: make(map[string]xsdComplexType),
		simplTypes: make(map[string]xsdSimpleType),
	}
}

func (b xmlBuilder) buildXml() []*xmlElem {
	var roots []xsdElement
	for _, s := range b.schemas {
		for _, e := range s.Elements {
			roots = append(roots, e)
		}
		for _, t := range s.ComplexTypes {
			b.complTypes[t.Name] = t
		}
		for _, t := range s.SimpleTypes {
			b.simplTypes[t.Name] = t
		}
	}

	var xelems []*xmlElem
	for _, e := range roots {
		xelems = append(xelems, b.traverse(e))
	}

	return xelems
}

func (b xmlBuilder) buildFromComplexContent(xelem *xmlElem, c xsdComplexContent) {
	if c.Extension != nil {
		if c.Extension.Sequence != nil {
			for _, e := range c.Extension.Sequence {
				xelem.Children = append(xelem.Children, b.traverse(e))
			}
		}
		base := c.Extension.Base
		switch t := b.findType(base).(type) {
		case xsdComplexType:
			b.buildFromComplexType(xelem, t)
		}

	}
}

func typeFromXsdType(typ string) string {
	switch typ {
	case "boolean":
		typ = "bool"
	case "language", "dateTime", "Name", "token":
		typ = "string"
	case "long", "short", "integer", "int":
		typ = "int"
	case "decimal":
		typ = "float64"
	}
	return typ
}

func addAttributes(xelem *xmlElem, attribs []xsdAttribute) {
	if attribs != nil {
		for _, attr := range attribs {
			typ := typeFromXsdType(stripNamespace(attr.Type))
			xelem.Attribs = append(xelem.Attribs, xmlAttrib{Name: attr.Name, Type: typ})
		}
	}
}

// A simple content can refer to a text-only complex type
func (b xmlBuilder) buildFromSimpleContent(xelem *xmlElem, c xsdSimpleContent) {
	if c.Extension != nil {
		if c.Extension.Attributes != nil {
			b.buildFromAttributes(xelem, c.Extension.Attributes)
		}

		switch t := b.findType(c.Extension.Base).(type) {
		case xsdComplexType:
			b.buildFromComplexType(xelem, t)
		case xsdSimpleType:
			buildFromSimpleType(xelem, t)
		default:
			if len(xelem.Attribs) == 0 {
				xelem.Type = typeFromXsdType(t.(string))
			}
		}
	}
	if c.Restriction != nil {
		switch t := b.findType(c.Restriction.Base).(type) {
		case xsdComplexType:
			b.buildFromComplexType(xelem, t)
		case xsdSimpleType:
			buildFromSimpleType(xelem, t)
		default:
			xelem.Type = typeFromXsdType(t.(string))
			addAttributes(xelem, c.Extension.Attributes)
		}
	}
}

func (b xmlBuilder) buildFromAttributes(xelem *xmlElem, attrs []xsdAttribute) {
	for _, a := range attrs {
		attr := xmlAttrib{Name: a.Name}
		switch t := b.findType(a.Type).(type) {
		case xsdSimpleType:
			attr.Type = typeFromXsdType(stripNamespace(t.Restriction.Base))
		default:
			attr.Type = typeFromXsdType(t.(string))
		}
		xelem.Attribs = append(xelem.Attribs, attr)
	}
}

func (b xmlBuilder) buildFromComplexType(xelem *xmlElem, t xsdComplexType) {
	if t.Sequence != nil {
		for _, e := range t.Sequence {
			xelem.Children = append(xelem.Children, b.traverse(e))
		}
	}

	if t.Attributes != nil {
		b.buildFromAttributes(xelem, t.Attributes)
	}

	if t.ComplexContent != nil {
		b.buildFromComplexContent(xelem, *t.ComplexContent)
	}

	if t.SimpleContent != nil {
		b.buildFromSimpleContent(xelem, *t.SimpleContent)
	}
}

func (b xmlBuilder) buildFromElement(xelem *xmlElem, e xsdElement) {
	xelem.Name = e.Name
	xelem.Type = e.Name
	if e.Max == "unbounded" {
		xelem.List = true
	}

	if e.Type != "" {
		switch t := b.findType(e.Type).(type) {
		case xsdComplexType:
			b.buildFromComplexType(xelem, t)
		case xsdSimpleType:
			buildFromSimpleType(xelem, t)
		default:
			switch typ := stripNamespace(e.Type); typ {
			case "boolean":
				xelem.Type = "bool"
			case "language", "dateTime", "Name", "token":
				xelem.Type = "string"
			case "long", "short", "integer":
				xelem.Type = "int"
			case "decimal":
				xelem.Type = "float64"
			default:
				xelem.Type = typ
			}
		}
		return
	}

	if e.ComplexType != nil { // inline complex type
		b.buildFromComplexType(xelem, *e.ComplexType)
		return
	}

	if e.SimpleType != nil { // inline simple type
		buildFromSimpleType(xelem, *e.SimpleType)
		return
	}
}

func buildFromSimpleType(xelem *xmlElem, t xsdSimpleType) {
	xelem.Type = typeFromXsdType(stripNamespace(t.Restriction.Base))
}

func (b xmlBuilder) traverse(e xsdElement) *xmlElem {
	xelem := &xmlElem{Name: e.Name}

	b.buildFromElement(xelem, e)
	return xelem
}

func (b xmlBuilder) findType(name string) interface{} {
	name = stripNamespace(name)
	if t, ok := b.complTypes[name]; ok {
		return t
	}
	if t, ok := b.simplTypes[name]; ok {
		return t
	}
	return name
}

func stripNamespace(name string) string {
	if s := strings.Split(name, ":"); len(s) > 1 {
		return s[len(s)-1]
	}
	return name
}
