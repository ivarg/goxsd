package main

import (
	"encoding/xml"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	extracted map[string]struct{}
)

func extractXsd(fname string) ([]xsdSchema, error) {
	schemas := make([]xsdSchema, 0, 10)
	extracted = make(map[string]struct{})
	schemas, err := extractAll(fname)
	if err != nil {
		return nil, err
	}
	return schemas, nil
}

func extractAll(fname string) ([]xsdSchema, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	schema, err := extract(f)
	if err != nil {
		return nil, err
	}

	schemas := []xsdSchema{schema}
	dir, file := filepath.Split(fname)
	extracted[file] = struct{}{}
	for _, imp := range schema.Imports {
		if _, ok := extracted[imp.Location]; ok {
			continue
		}
		s, err := extractAll(filepath.Join(dir, imp.Location))
		if err != nil {
			return nil, err
		}
		schemas = append(schemas, s...)
	}
	return schemas, nil
}

func extract(r io.Reader) (xsdSchema, error) {
	var root xsdSchema
	if err := xml.NewDecoder(r).Decode(&root); err != nil {
		log.Println("Error: could not decode")
		return xsdSchema{}, err
	}

	return root, nil
}

type xsdSchema struct {
	XMLName      xml.Name
	Ns           string        `xml:"xmlns,attr"`
	Imports      []nsimport    `xml:"import"`
	Elements     []element     `xml:"element"`
	ComplexTypes []complexType `xml:"complexType"`
	SimpleTypes  []simpleType  `xml:"simpleType"`
}

// ns parses the namespace from a value in the expected format
// http://host/namespace/v1
func (s xsdSchema) ns() string {
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
