// Things not yet implemented:
// - enforcing use="restricted" on attributes
// - namespaces

package main

import (
	"flag"
	"io"
	"log"
	"os"
	"strings"
	"text/template"
)

var (
	xsdpath  = "/Users/igaitan/src/github.com/TV4/video-metadata-api/src/main/xsd/export"
	xsdentry = "publish-metadata.xsd"
	rootNs   = "publish-metadata"

	complTypes map[string]complexType
	simplTypes map[string]simpleType

	types map[string]struct{}

	out io.Writer

	xsdFile string
)

func init() {
	complTypes = make(map[string]complexType)
	simplTypes = make(map[string]simpleType)

	types = make(map[string]struct{})

	tt = template.New("yyy").Funcs(fmap)
	tt.Parse(attr)
	tt.Parse(child)
	tt.Parse(elem)
	tt.Parse(templ)

}

func main() {
	flag.StringVar(&xsdFile, "xsd", "", "Path to an XSD file")
	flag.Parse()

	if xsdFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	xsd, err := extractXsd(xsdFile)
	if err != nil {
		log.Fatal(err)
	}
	parse(os.Stdout, buildXmlAll(xsd))
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

func buildXmlAll(schemas []xsdSchema) []*xmlElem {
	var xelems []*xmlElem
	for _, s := range schemas {
		xelems = append(xelems, buildXml(s)...)
	}

	return xelems
}

func buildXml(s xsdSchema) []*xmlElem {
	var roots []element
	for _, e := range s.Elements {
		roots = append(roots, e)
	}
	for _, t := range s.ComplexTypes {
		complTypes[t.Name] = t
	}
	for _, t := range s.SimpleTypes {
		simplTypes[t.Name] = t
	}

	var xelems []*xmlElem
	for _, r := range roots {
		xelems = append(xelems, traverse(r))
	}
	return xelems
}

func buildFromComplexContent(xelem *xmlElem, c complexContent) {
	if c.Extension != nil {
		if c.Extension.Sequence != nil {
			for _, e := range c.Extension.Sequence {
				xelem.Children = append(xelem.Children, traverse(e))
			}
		}
		base := c.Extension.Base
		switch t := findType(base).(type) {
		case complexType:
			buildFromComplexType(xelem, t)
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

func addAttributes(xelem *xmlElem, attribs []attribute) {
	if attribs != nil {
		for _, attr := range attribs {
			typ := typeFromXsdType(stripNamespace(attr.Type))
			xelem.Attribs = append(xelem.Attribs, xmlAttrib{Name: attr.Name, Type: typ})
		}
	}
}

// A simple content can refer to a text-only complex type
func buildFromSimpleContent(xelem *xmlElem, c simpleContent) {
	if c.Extension != nil {
		if c.Extension.Attributes != nil {
			buildFromAttributes(xelem, c.Extension.Attributes)
		}

		switch t := findType(c.Extension.Base).(type) {
		case complexType:
			buildFromComplexType(xelem, t)
		case simpleType:
			buildFromSimpleType(xelem, t)
		default:
			if len(xelem.Attribs) == 0 {
				xelem.Type = typeFromXsdType(t.(string))
			}
		}
	}
	if c.Restriction != nil {
		switch t := findType(c.Restriction.Base).(type) {
		case complexType:
			buildFromComplexType(xelem, t)
		case simpleType:
			buildFromSimpleType(xelem, t)
		default:
			xelem.Type = typeFromXsdType(t.(string))
			addAttributes(xelem, c.Extension.Attributes)
		}
	}
}

func buildFromAttributes(xelem *xmlElem, attrs []attribute) {
	for _, a := range attrs {
		attr := xmlAttrib{Name: a.Name}
		switch t := findType(a.Type).(type) {
		case simpleType:
			attr.Type = typeFromXsdType(stripNamespace(t.Restriction.Base))
		default:
			attr.Type = typeFromXsdType(t.(string))
		}
		xelem.Attribs = append(xelem.Attribs, attr)
	}
}

func buildFromComplexType(xelem *xmlElem, t complexType) {
	if t.Sequence != nil {
		for _, e := range t.Sequence {
			xelem.Children = append(xelem.Children, traverse(e))
		}
	}

	if t.Attributes != nil {
		buildFromAttributes(xelem, t.Attributes)
	}

	if t.ComplexContent != nil {
		buildFromComplexContent(xelem, *t.ComplexContent)
	}

	if t.SimpleContent != nil {
		buildFromSimpleContent(xelem, *t.SimpleContent)
	}
}

func buildFromElement(xelem *xmlElem, e element) {
	xelem.Name = e.Name
	xelem.Type = e.Name
	if e.Max == "unbounded" {
		xelem.List = true
	}

	if e.Type != "" {
		switch t := findType(e.Type).(type) {
		case complexType:
			buildFromComplexType(xelem, t)
		case simpleType:
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
		buildFromComplexType(xelem, *e.ComplexType)
		return
	}

	if e.SimpleType != nil { // inline simple type
		buildFromSimpleType(xelem, *e.SimpleType)
		return
	}
}

func buildFromSimpleType(xelem *xmlElem, t simpleType) {
	xelem.Type = typeFromXsdType(stripNamespace(t.Restriction.Base))
}

func traverse(e element) *xmlElem {
	xelem := &xmlElem{Name: e.Name}

	buildFromElement(xelem, e)
	return xelem
}

func findType(name string) interface{} {
	name = stripNamespace(name)
	if t, ok := complTypes[name]; ok {
		return t
	}
	if t, ok := simplTypes[name]; ok {
		return t
	}
	return name
}

var (
	attr = "{{ define \"Attr\" }}{{ printf \"  %s \" (title .Name) }}{{ printf \"%s `xml:\\\"%s,attr\\\"`\" .Type .Name }}\n{{ end }}"

	child = "{{ define \"Child\" }}{{ printf \"  %s \" (title .Name) }}{{ if .List }}[]{{ end }}{{ printf \"%s `xml:\\\"%s\\\"`\" .Type .Name }}\n{{ end }}"

	elem = `{{ define "Elem" }}{{ printf "type %s struct {\n" (assimilate .Name) }}{{ range $a := .Attribs }}{{ template "Attr" $a }}{{ end }}{{ range $c := .Children }}{{ template "Child" $c }}{{ end }}}
{{ end }}`

	templ = `{{ template "Elem" . }}
`

	fmap = template.FuncMap{
		"title":      strings.Title,
		"assimilate": assimilate,
	}

	tt *template.Template
)

func assimilate(name string) string {
	s := strings.Split(name, "-")
	if len(s) > 1 {
		for i := 1; i < len(s); i++ {
			s[i] = strings.Title(s[i])
		}
		return strings.Join(s, "")
	}
	return name
}

func parse(out io.Writer, roots []*xmlElem) {
	for _, e := range roots {
		doParse(e, out)
	}
}

func doParse(root *xmlElem, out io.Writer) {
	if _, ok := types[root.Name]; ok {
		return
	}
	if err := tt.Execute(out, root); err != nil {
		log.Fatal(err)
	}
	types[root.Name] = struct{}{}

	for _, e := range root.Children {
		if !primitive(e) {
			doParse(e, out)
		}
	}
}

func primitive(e *xmlElem) bool {
	switch e.Type {
	case "integer", "decimal", "token", "bool", "string", "int":
		return true
	}
	return false
}

func namespace(name string) string {
	if s := strings.Split(name, ":"); len(s) > 1 {
		return s[0]
	}
	return ""
}

func stripNamespace(name string) string {
	if s := strings.Split(name, ":"); len(s) > 1 {
		return s[len(s)-1]
	}
	return name
}
