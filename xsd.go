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

func extractSchemas(fname string) ([]xsdSchema, error) {
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

// xsdSchema is the Go representation of an XSD schema.
type xsdSchema struct {
	XMLName      xml.Name
	Ns           string           `xml:"xmlns,attr"`
	Imports      []xsdImport      `xml:"import"`
	Elements     []xsdElement     `xml:"element"`
	ComplexTypes []xsdComplexType `xml:"complexType"`
	SimpleTypes  []xsdSimpleType  `xml:"simpleType"`
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

type xsdImport struct {
	Location string `xml:"schemaLocation,attr"`
}

type xsdElement struct {
	Name        string          `xml:"name,attr"`
	Type        string          `xml:"type,attr"`
	Default     string          `xml:"default,attr"`
	Min         string          `xml:"minOccurs,attr"`
	Max         string          `xml:"maxOccurs,attr"`
	Annotation  string          `xml:"annotation>documentation"`
	ComplexType *xsdComplexType `xml:"complexType"` // inline complex type
	SimpleType  *xsdSimpleType  `xml:"simpleType"`  // inline simple type
}

func (e xsdElement) isList() bool {
	return e.Max == "unbounded"
}

func (e xsdElement) inlineType() bool {
	return e.Type == ""
}

type xsdComplexType struct {
	Name           string             `xml:"name,attr"`
	Abstract       string             `xml:"abstract,attr"`
	Annotation     string             `xml:"annotation>documentation"`
	Sequence       []xsdElement       `xml:"sequence>element"`
	Attributes     []xsdAttribute     `xml:"attribute"`
	ComplexContent *xsdComplexContent `xml:"complexContent"`
	SimpleContent  *xsdSimpleContent  `xml:"simpleContent"`
}

type xsdComplexContent struct {
	Extension   *xsdExtension   `xml:"extension"`
	Restriction *xsdRestriction `xml:"restriction"`
}

type xsdSimpleContent struct {
	Extension   *xsdExtension   `xml:"extension"`
	Restriction *xsdRestriction `xml:"restriction"`
}

type xsdExtension struct {
	Base       string         `xml:"base,attr"`
	Attributes []xsdAttribute `xml:"attribute"`
	Sequence   []xsdElement   `xml:"sequence>element"`
}

type xsdAttribute struct {
	Name       string `xml:"name,attr"`
	Type       string `xml:"type,attr"`
	Use        string `xml:"use,attr"`
	Annotation string `xml:"annotation>documentation"`
}

type xsdSimpleType struct {
	Name        string         `xml:"name,attr"`
	Annotation  string         `xml:"annotation>documentation"`
	Restriction xsdRestriction `xml:"restriction"`
}

type xsdRestriction struct {
	Base        string           `xml:"base,attr"`
	Pattern     xsdPattern       `xml:"pattern"`
	Enumeration []xsdEnumeration `xml:"enumeration"`
}

type xsdPattern struct {
	Value string `xml:"value,attr"`
}

type xsdEnumeration struct {
	Value string `xml:"value,attr"`
}
