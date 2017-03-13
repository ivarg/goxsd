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

	s, err := parseXSDFile(xsdFile)
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

	bldr := newBuilder(s)

	gen := generator{
		pkg:      pckg,
		prefix:   prefix,
		exported: exported,
	}

	if err := gen.do(out, bldr.buildXML()); err != nil {
		fmt.Println("Code generation failed unexpectedly:", err.Error())
		os.Exit(1)
	}
}

// xmlTree is the representation of an XML element node in a tree. It
// contains information about whether
// - it is of a basic data type or a composite type (in which case its
//   type equals its name)
// - if it represents a list of children to its parent
// - if it has children of its own
// - any attributes
// - if the element contains any character data
type xmlTree struct {
	Name     string
	Type     string
	List     bool
	Cdata    bool
	Attribs  []xmlAttrib
	Children []*xmlTree
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

// newBuilder creates a new initialized builder populated with the given
// xsdSchema slice.
func newBuilder(schemas []xsdSchema) *builder {
	return &builder{
		schemas:    schemas,
		complTypes: make(map[string]xsdComplexType),
		simplTypes: make(map[string]xsdSimpleType),
	}
}

// buildXML generates and returns a tree of xmlTree objects based on a set of
// parsed XSD schemas.
func (b *builder) buildXML() []*xmlTree {
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

	var xelems []*xmlTree
	for _, e := range roots {
		xelems = append(xelems, b.buildFromElement(e))
	}

	return xelems
}

// buildFromElement builds an xmlTree from an xsdElement, recursively
// traversing the XSD type information to build up an XML element hierarchy.
func (b *builder) buildFromElement(e xsdElement) *xmlTree {
	xelem := &xmlTree{Name: e.Name, Type: e.Name}

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

// buildFromComplexType takes an xmlTree and an xsdComplexType, containing
// XSD type information for xmlTree enrichment.
func (b *builder) buildFromComplexType(xelem *xmlTree, t xsdComplexType) {
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
func (b *builder) buildFromSimpleType(xelem *xmlTree, t xsdSimpleType) {
	xelem.Type = b.findType(t.Restriction.Base).(string)
}

func (b *builder) buildFromComplexContent(xelem *xmlTree, c xsdComplexContent) {
	if c.Extension != nil {
		b.buildFromExtension(xelem, c.Extension)
	}
}

// A simple content can refer to a text-only complex type
func (b *builder) buildFromSimpleContent(xelem *xmlTree, c xsdSimpleContent) {
	if c.Extension != nil {
		b.buildFromExtension(xelem, c.Extension)
	}

	if c.Restriction != nil {
		b.buildFromRestriction(xelem, c.Restriction)
	}
}

// buildFromExtension extends an existing type, simple or complex, with a
// sequence.
func (b *builder) buildFromExtension(xelem *xmlTree, e *xsdExtension) {
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

func (b *builder) buildFromRestriction(xelem *xmlTree, r *xsdRestriction) {
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

func (b *builder) buildFromAttributes(xelem *xmlTree, attrs []xsdAttribute) {
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
func (b *builder) findType(name string) interface{} {
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
	case "language", "Name", "token", "duration", "anyURI", "normalizedString":
		return "string"
	case "long", "short", "integer", "int":
		return "int"
	case "unsignedShort":
		return "uint16"
	case "decimal":
		return "float64"
	case "dateTime", "date":
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
