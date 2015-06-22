package main

import (
	"encoding/xml"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// extractXsd decodes the xsd into Go structs, recursively following imports
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

	//log.Println("Decoded", fname)
	schemas[root.ns()] = root
	for _, imp := range root.Imports {
		extractXsd(imp.Location)
	}
}

type schema struct {
	XMLName      xml.Name
	Ns           string        `xml:"xmlns,attr"`
	Imports      []nsimport    `xml:"import"`
	Elements     []element     `xml:"element"`
	ComplexTypes []complexType `xml:"complexType"`
	SimpleTypes  []simpleType  `xml:"simpleType"`
}

// ns parses the namespace from a value in the expected format
// http://host/namespace/v1
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
	Name           string          `xml:"name,attr"`
	Abstract       string          `xml:"abstract,attr"`
	Annotation     string          `xml:"annotation>documentation"`
	Sequence       []element       `xml:"sequence>element"`
	Attributes     []attribute     `xml:"attribute"`
	ComplexContent *complexContent `xml:"complexContent"`
	SimpleContent  *simpleContent  `xml:"simpleContent"`
}

type complexContent struct {
	Extension   *extension   `xml:"extension"`
	Restriction *restriction `xml:"restriction"`
}

type simpleContent struct {
	Extension   *extension   `xml:"extension"`
	Restriction *restriction `xml:"restriction"`
}

type extension struct {
	Base       string      `xml:"base,attr"`
	Attributes []attribute `xml:"attribute"`
	Sequence   []element   `xml:"sequence>element"`
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
