package main

import (
	"fmt"
	"io"
	"log"
	"strings"
	"text/template"
)

var (
	types map[string]struct{}

	attr = "{{ define \"Attr\" }}{{ printf \"  %s \" (title .Name) }}{{ printf \"%s `xml:\\\"%s,attr\\\"`\" .Type .Name }}\n{{ end }}"

	child = "{{ define \"Child\" }}{{ printf \"  %s \" (title .Name) }}{{ if .List }}[]{{ end }}{{ printf \"%s `xml:\\\"%s\\\"`\" .FieldType .Name }}\n{{ end }}"

	cdata = "{{ define \"Cdata\" }}{{ printf \"%s %s `xml:\\\",chardata\\\"`\\n\" (title .Name) .Type }}{{ end }}"

	elem = `{{ define "Elem" }}{{ printf "type %s struct {\n" (assimilate .Name) }}{{ range $a := .Attribs }}{{ template "Attr" $a }}{{ end }}{{ range $c := .Children }}{{ template "Child" $c }}{{ end }} {{ if .Cdata }}{{ template "Cdata" . }}{{ end }} }
	{{ end }}`

	templ = `{{ template "Elem" . }}
`

	fmap = template.FuncMap{
		"title":      strings.Title,
		"assimilate": assimilate,
	}

	tt *template.Template
)

func init() {
	types = make(map[string]struct{})

	tt = template.New("yyy").Funcs(fmap)
	tt.Parse(attr)
	tt.Parse(cdata)
	tt.Parse(child)
	tt.Parse(elem)
	tt.Parse(templ)

}

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

func generateGo(out io.Writer, roots []*xmlElem) {
	if pckg != "" {
		fmt.Fprintf(out, "package %s\n\n", pckg)
	}

	for _, e := range roots {
		doGenerate(e, out)
	}
}

func doGenerate(root *xmlElem, out io.Writer) {
	if _, ok := types[root.Name]; ok {
		return
	}
	if err := tt.Execute(out, root); err != nil {
		log.Fatal(err)
	}
	types[root.Name] = struct{}{}

	for _, e := range root.Children {
		if !primitive(e) {
			doGenerate(e, out)
		}
	}
}

func primitive(e *xmlElem) bool {
	if e.Cdata {
		return false
	}

	switch e.Type {
	case "bool", "string", "int", "float64", "time.Time":
		return true
	}
	return false
}
