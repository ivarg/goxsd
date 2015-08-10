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
	output, pckg, prefix string
	exported             bool

	usage = `Usage: goxsd [options] <xsd_file>

Options:
  -o <file>     Destination file [default: stdout]
  -p <package>  Package name [default: goxsd]
  -e            Generate exported structs [default: false]
  -x <prefix>   Struct name prefix [default: ""]

goxsd is a tool for generating XML decoding/encoding Go structs, according
to an XSD schema.
`
)

func main() {
	flag.StringVar(&output, "o", "", "Name of output file")
	flag.StringVar(&pckg, "p", "goxsd", "Name of the Go package")
	flag.StringVar(&prefix, "x", "", "Name of the Go package")
	flag.BoolVar(&exported, "e", false, "Generate exported structs")
	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Println(usage)
		os.Exit(1)
	}
	xsdFile := flag.Arg(0)

	s, err := extractSchemas(xsdFile)
	if err != nil {
		log.Fatal(err)
	}

	out := os.Stdout
	if output != "" {
		if out, err = os.Create(output); err != nil {
			fmt.Println("Could not create or truncate output file:", output)
			os.Exit(1)
		}
	}

	builder := newBuilder(s)
	generateGo(out, builder.buildXML())
}

type xmlElem struct {
	Name     string
	Type     string
	List     bool
	Cdata    bool
	Attribs  []xmlAttrib
	Children []*xmlElem
}

// If this is a chardata field, the field type must point to a
// struct, even if the element type is a built-in primitive.
func (e *xmlElem) FieldType() string {
	if e.Cdata {
		return e.Name
	}
	return e.Type
}

type xmlAttrib struct {
	Name string
	Type string
}

type builder struct {
	schemas    []xsdSchema
	complTypes map[string]xsdComplexType
	simplTypes map[string]xsdSimpleType
}

func newBuilder(s []xsdSchema) builder {
	return builder{
		schemas:    s,
		complTypes: make(map[string]xsdComplexType),
		simplTypes: make(map[string]xsdSimpleType),
	}
}

func (b builder) buildXML() []*xmlElem {
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
		xelems = append(xelems, b.buildFromElement(e))
	}

	return xelems
}

// buildFromElement builds an xmlElem from an xsdElement, recursively
// traversing the XSD type information to build up an XML element hierarchy.
func (b builder) buildFromElement(e xsdElement) *xmlElem {
	xelem := &xmlElem{Name: e.Name, Type: e.Name}

	if e.isList() {
		xelem.List = true
	}

	if !e.inlineType() {
		switch t := b.findType(e.Type).(type) {
		case xsdComplexType:
			b.buildFromComplexType(xelem, t)
		case xsdSimpleType:
			b.buildFromSimpleType(xelem, t)
		case string:
			xelem.Type = t
		}
		return xelem
	}

	if e.ComplexType != nil { // inline complex type
		b.buildFromComplexType(xelem, *e.ComplexType)
		return xelem
	}

	if e.SimpleType != nil { // inline simple type
		b.buildFromSimpleType(xelem, *e.SimpleType)
		return xelem
	}

	return xelem
}

// buildFromComplexType takes an xmlElem and an xsdComplexType, containing
// XSD type information for xmlElem enrichment.
func (b builder) buildFromComplexType(xelem *xmlElem, t xsdComplexType) {
	if t.Sequence != nil { // Does the element have children?
		for _, e := range t.Sequence {
			xelem.Children = append(xelem.Children, b.buildFromElement(e))
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

// buildFromSimpleType assumes restriction child and fetches the base value,
// assuming that value is of a XSD built-in data type.
func (b builder) buildFromSimpleType(xelem *xmlElem, t xsdSimpleType) {
	xelem.Type = b.findType(t.Restriction.Base).(string)
}

func (b builder) buildFromComplexContent(xelem *xmlElem, c xsdComplexContent) {
	if c.Extension != nil {
		b.buildFromExtension(xelem, c.Extension)
	}
}

// A simple content can refer to a text-only complex type
func (b builder) buildFromSimpleContent(xelem *xmlElem, c xsdSimpleContent) {
	if c.Extension != nil {
		b.buildFromExtension(xelem, c.Extension)
	}

	if c.Restriction != nil {
		b.buildFromRestriction(xelem, c.Restriction)
	}
}

// buildFromExtension extends an existing type, simple or complex, with a
// sequence.
func (b builder) buildFromExtension(xelem *xmlElem, e *xsdExtension) {
	switch t := b.findType(e.Base).(type) {
	case xsdComplexType:
		b.buildFromComplexType(xelem, t)
	case xsdSimpleType:
		b.buildFromSimpleType(xelem, t)
		// If element is of simpleType and has attributes, it must collect
		// its value as chardata.
		if e.Attributes != nil {
			xelem.Cdata = true
		}
	default:
		xelem.Type = t.(string)
		// If element is of built-in type but has attributes, it must collect
		// its value as chardata.
		if e.Attributes != nil {
			xelem.Cdata = true
		}
	}

	if e.Sequence != nil {
		for _, e := range e.Sequence {
			xelem.Children = append(xelem.Children, b.buildFromElement(e))
		}
	}

	if e.Attributes != nil {
		b.buildFromAttributes(xelem, e.Attributes)
	}
}

func (b builder) buildFromRestriction(xelem *xmlElem, r *xsdRestriction) {
	switch t := b.findType(r.Base).(type) {
	case xsdSimpleType:
		b.buildFromSimpleType(xelem, t)
	case xsdComplexType:
		b.buildFromComplexType(xelem, t)
	case xsdComplexContent:
		panic("Restriction on complex content is not implemented")
	default:
		panic("Unexpected base type to restriction")
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

func (b builder) buildFromAttributes(xelem *xmlElem, attrs []xsdAttribute) {
	for _, a := range attrs {
		attr := xmlAttrib{Name: a.Name}
		switch t := b.findType(a.Type).(type) {
		case xsdSimpleType:
			// Get type name from simpleType
			// If Restriction.Base is a simpleType or complexType, we panic
			attr.Type = b.findType(t.Restriction.Base).(string)
		case string:
			// If empty, then simpleType is present as content, but we ignore
			// that now
			attr.Type = t
		}
		xelem.Attribs = append(xelem.Attribs, attr)
	}
}

// findType takes a type name and checks if it is a registered XSD type
// (simple or complex), in which case that type is returned. If no such
// type can be found, the XSD specific primitive types are mapped to their
// Go correspondents. If no XSD type was found, the type name itself is
// returned.
func (b builder) findType(name string) interface{} {
	name = stripNamespace(name)
	if t, ok := b.complTypes[name]; ok {
		return t
	}
	if t, ok := b.simplTypes[name]; ok {
		return t
	}

	switch name {
	case "boolean":
		return "bool"
	case "language", "Name", "token", "duration":
		return "string"
	case "long", "short", "integer", "int":
		return "int"
	case "decimal":
		return "float64"
	case "dateTime":
		return "time.Time"
	default:
		return name
	}
}

func stripNamespace(name string) string {
	if s := strings.Split(name, ":"); len(s) > 1 {
		return s[len(s)-1]
	}
	return name
}
