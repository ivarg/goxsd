// 1. Element of a specified type
// ...
//   <element name> <element type>
// ...
//
// 2. Element of no type
// ...
//   <element name> struct {
//   }
// ...
//
// type <element name> struct {
// ...
// }

package main

import (
	"encoding/xml"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/kr/pretty"
)

var (
	xsdpath  = "/Users/igaitan/src/github.com/TV4/video-metadata-api/src/main/xsd/export"
	xsdentry = "publish-metadata.xsd"
	rootNs   = "publish-metadata"

	elements   map[string]element
	complTypes map[string]complexType
	simplTypes map[string]simpleType
	schemas    map[string]schema
)

func init() {
	elements = make(map[string]element)
	complTypes = make(map[string]complexType)
	simplTypes = make(map[string]simpleType)
	schemas = make(map[string]schema)
}

func main() {
	extractXsd(xsdentry)
	buildXmlStructs()

	//fmt.Println("top elements", elements)
	//generate(rootElem)

	//for _, t := range complTypes {
	//pretty.Println(t)
	//}

	//parse()
}

func extractXsd(fname string) {
	// TODO(ivar): check if this file has already been extracted
	loc := filepath.Join(xsdpath, fname)
	f, err := os.Open(loc)
	if err != nil {
		log.Println("Error: could not open", loc)
		return
	}
	defer f.Close()
	buf, err := ioutil.ReadAll(f)
	if err != nil {
		log.Println("Error: could not read", loc)
		return
	}
	var root schema
	if err := xml.Unmarshal(buf, &root); err != nil {
		log.Println("Error: could not unmarshal", loc)
		return
	}

	if _, ok := schemas[root.ns()]; ok {
		return
	}

	schemas[root.ns()] = root
	for _, imp := range root.Imports {
		extractXsd(imp.Location)
	}
}

type xmlElem struct {
	Name     string
	Type     string
	Value    string
	Attribs  []xmlAttrib
	Children []xmlElem
}

type xmlAttrib struct {
	Name  string
	Value string
}

func buildXmlStructs() xmlElem {
	for _, s := range schemas {
		for _, e := range s.Elements {
			elements[e.Name] = e
		}
		for _, t := range s.ComplexTypes {
			complTypes[t.Name] = t
		}
		for _, t := range s.SimpleTypes {
			simplTypes[t.Name] = t
		}
	}

	rootElem := schemas[rootNs].Elements[0]
	xelem := traverse(rootElem)
	pretty.Println(xelem)
	return xelem
}

func traverse(e element) xmlElem {
	//fmt.Println("traversing", e.Name)
	xelem := xmlElem{Name: e.Name}

	// If the element type is external, we need to look it up to see how
	// to lay out the struct.
	if e.Type != "" { // external type reference
		typ := findType(e.Type)
		//fmt.Println(typ)
		// If complex type, we will add children and recursively traverse them
		switch t := typ.(type) {
		case complexType:
			if t.Sequence != nil {
				//fmt.Println("children:")
				for _, e := range t.Sequence {
					//fmt.Println("  ", e.Name, e.Type)
					xelem.Children = append(xelem.Children, traverse(e))
				}
			}
			if t.Attributes != nil {
				for _, a := range t.Attributes {
					xelem.Attribs = append(xelem.Attribs, xmlAttrib{Name: a.Name})
				}
			}
			// If it is not complex, we must map it to primitive type
		case simpleType:
			xelem.Type = stripNamespace(t.Restriction.Base)
		case string:
			println("YYYYY NIY")
		default:
			panic("unknown type: " + typ.(string))
		}
		//pretty.Println(xelem)
		return xelem
	}

	if e.ComplexType != nil { // inline complex type
		if e.ComplexType.Sequence != nil {
			//fmt.Println("children:")
			for _, e := range e.ComplexType.Sequence {
				//fmt.Println("  ", e.Name, e.Type)
				xelem.Children = append(xelem.Children, traverse(e))
			}
		}
		//pretty.Println(xelem)
		return xelem
	}

	if e.SimpleType != nil { // inline simple type
		xelem.Type = stripNamespace(e.SimpleType.Restriction.Base)
		return xelem
	}

	println("ZZZZZZZ NIY")
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
	child = "{{ define \"Child\" }}{{ printf \"  %s %s `xml:\\\"%s\\\"`\" (title .Name) .Name .Name  }}\n{{ end }}"

	//elem = "{{ define \"Elem\" }}{{ printf \" %s %s `xml:\\\"%s\\\"`\" (title .Name) (stripNs .Type) (stripNs .Type) }}\n{{ end }}"
	elem = `{{ define "Elem" }}{{ printf "type %s struct {\n" (title .Name) (stripNs .Type) }}{{ end }}`

	simple = `{{ define "Simple" }}{{ printf "type %s %s \n" .Name (stripNs .Restriction.Base) }}{{ end }}`

	complx = `{{ define "Complex" }}{{ printf "type %s struct {\n" .Name }}{{ range $e := .Sequence }}{{ template "Elem" $e }}{{ end }}
}
{{ end }}`

	templ = `{{ range $t := . }}{{ template "Complex" $t }}{{ end }}`

	fmap = template.FuncMap{
		"stripNs": stripNamespace,
		"title":   strings.Title,
	}
)

func parse() {
	tt := template.New("yyy").Funcs(fmap)
	tt.Parse(elem)
	tt.Parse(simple)
	tt.Parse(complx)
	tt.Parse(templ)
	if err := tt.Execute(os.Stdout, complTypes); err != nil {
		log.Fatal(err)
	}
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

type schema struct {
	XMLName      xml.Name
	Ns           string        `xml:"xmlns,attr"`
	Imports      []nsimport    `xml:"import"`
	Elements     []element     `xml:"element"`
	ComplexTypes []complexType `xml:"complexType"`
	SimpleTypes  []simpleType  `xml:"simpleType"`
}

func (s schema) ns() string {
	split := strings.Split(s.Ns, "/")
	if len(split) > 2 {
		return split[len(split)-2]
	}
	return ""
}

type nsimport struct {
	Location string `xml:"schemaLocation,attr"`
}

type element struct {
	Name        string       `xml:"name,attr"`
	Type        string       `xml:"type,attr"`
	Default     string       `xml:"default,attr"`
	Min         string       `xml:"minOccurs,attr"`
	Max         string       `xml:"maxOccurs,attr"`
	Annotation  string       `xml:"annotation>documentation"`
	ComplexType *complexType `xml:"complexType"` // inline complex type
	SimpleType  *simpleType  `xml:"simpleType"`  // inline simple type
}

type complexType struct {
	Name           string           `xml:"name,attr"`
	Abstract       string           `xml:"abstract,attr"`
	Annotation     string           `xml:"annotation>documentation"`
	Sequence       []element        `xml:"sequence>element"`
	ComplexContent []complexContent `xml:"complexContent"`
	SimpleContent  []simpleContent  `xml:"simpleContent"`
	Attributes     []attribute      `xml:"attribute"`
}

type complexContent struct {
	Extension extension `xml:"extension"`
}

type simpleContent struct {
	Extension extension `xml:"extension"`
}

type extension struct {
	Base      string    `xml:"base,attr"`
	Attribute attribute `xml:"attribute"`
	Sequence  []element `xml:"sequence>element"`
}

type attribute struct {
	Name       string `xml:"name,attr"`
	Type       string `xml:"type,attr"`
	Use        string `xml:"use,attr"`
	Annotation string `xml:"annotation>documentation"`
}

type simpleType struct {
	Name        string      `xml:"name,attr"`
	Annotation  string      `xml:"annotation>documentation"`
	Restriction restriction `xml:"restriction"`
}

type restriction struct {
	Base        string        `xml:"base,attr"`
	Pattern     pattern       `xml:"pattern"`
	Enumeration []enumeration `xml:"enumeration"`
}

type pattern struct {
	Value string `xml:"value,attr"`
}

type enumeration struct {
	Value string `xml:"value,attr"`
}
