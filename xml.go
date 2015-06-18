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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	xsdpath  = "/Users/igaitan/src/github.com/TV4/video-metadata-api/src/main/xsd/export"
	xsdentry = "publish-metadata.xsd"

	elements   map[string]element
	complTypes map[string]complexType
	simplTypes map[string]simpleType
	schemas    map[string]schema

	fmap = template.FuncMap{
		"stripNs": stripNamespace,
		"title":   strings.Title,
	}
)

func init() {
	elements = make(map[string]element)
	complTypes = make(map[string]complexType)
	simplTypes = make(map[string]simpleType)
	schemas = make(map[string]schema)
}

func main() {
	process(xsdentry)
	postprocess()

	//fmt.Println("top elements", elements)
	//rootElem := schemas[xsdentry].Elements[0]
	//generate(rootElem)

	//for _, t := range complTypes {
	//pretty.Println(t)
	//}

	//parse()
}

func postprocess() {
	for _, s := range schemas {
		for _, e := range s.Elements {
			elements[e.Name] = e
			fmt.Println("top level element", e.Name)
		}
		for _, t := range s.ComplexTypes {
			complTypes[t.Name] = t
		}
		for _, t := range s.SimpleTypes {
			simplTypes[t.Name] = t
		}
	}
}

var (
	//elem = "{{ define \"Elem\" }}{{ printf \" %s %s `xml:\\\"%s\\\"`\" (title .Name) (stripNs .Type) (stripNs .Type) }}\n{{ end }}"
	elem = `{{ define "Elem" }}{{ printf "type %s struct {\n" (title .Name) (stripNs .Type) }}{{ end }}`

	simple = `{{ define "Simple" }}{{ printf "type %s %s \n" .Name (stripNs .Restriction.Base) }}{{ end }}`

	complx = `{{ define "Complex" }}{{ printf "type %s struct {\n" .Name }}{{ range $e := .Sequence }}{{ template "Elem" $e }}{{ end }}
}
{{ end }}`

	templ = `{{ range $t := . }}{{ template "Complex" $t }}{{ end }}`
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

func generate(elem element) {
	fmt.Println("Generating element", elem.Name)
	if elem.Type != "" {
		elem.Type = stripNamespace(elem.Type)

		if t, ok := simplTypes[elem.Type]; ok {
			fmt.Println("Found external simple type", t.Name)
		} else if t, ok := complTypes[elem.Type]; ok {
			fmt.Println("Found external complex type", t.Name)
		} else {
			panic("couldn't find type: " + elem.Type)
		}
		return
	}

	if elem.SimpleType != nil {
		fmt.Println("Found inline simple type")
	} else if elem.ComplexType != nil {
		fmt.Println("Found inline complex type")
		seq := elem.ComplexType.Sequence
		for _, e := range seq {
			generate(e)
		}
	} else {
		panic("element without content: " + elem.Type)
	}
}

func process(fname string) {
	if _, ok := schemas[fname]; ok {
		return
	}

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

	schemas[fname] = root
	for _, imp := range root.Imports {
		process(imp.Location)
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

///////////////////////

type xmlElem struct {
	Value    string
	Attribs  []xmlAttrib
	Children []xmlElem
}

type xmlAttrib struct {
	Name  string
	Value string
}

//////////////////

type schema struct {
	XMLName      xml.Name
	Imports      []nsimport    `xml:"import"`
	Elements     []element     `xml:"element"`
	ComplexTypes []complexType `xml:"complexType"`
	SimpleTypes  []simpleType  `xml:"simpleType"`
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
	ComplexType *complexType `xml:"complexType"`
	SimpleType  *simpleType  `xml:"simpleType"`
}

type complexType struct {
	Name           string           `xml:"name,attr"`
	Abstract       string           `xml:"abstract,attr"`
	Annotation     string           `xml:"annotation>documentation"`
	Sequence       []element        `xml:"sequence>element"`
	ComplexContent []complexContent `xml:"complexContent"`
	SimpleContent  []simpleContent  `xml:"simpleContent"`
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
