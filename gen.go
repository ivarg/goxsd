package main

import (
	"encoding/json"
	"html/template"
	"os"
	"strings"
)

var (
	mapfile = "/Users/igaitan/src/github.com/TV4/search-import-api/_mappings/default.json"
)

const (
	date = iota
	nested
	str
)

type prop struct {
	Name  string
	Type  string
	Props []prop
}

func jsonmain() {
	f, err := os.Open(mapfile)
	if err != nil {
		panic(err)
	}

	var root map[string]interface{}

	if err = json.NewDecoder(f).Decode(&root); err != nil {
		panic(err)
	}
	f.Close()
	//mappings := root["mappings"].(map[string]interface{})
	//def := mappings["_default_"].(map[string]interface{})
	//properties := def["properties"].(map[string]interface{})

	//props := separate(findProps(properties))

	//tt := template.Must(template.New("yyy").Funcs(fmap).Parse(structTempl))
	////t := template.Must(template.New("xxx").Funcs(fmap).Parse(contentTempl))
	//tt.Parse(contentTempl)
	//if err := tt.Execute(os.Stdout, props); err != nil {
	//log.Fatal(err)
	//}
}

//func findProps(props map[string]interface{}) []prop {
//var blacklist []string
//var res []prop
//for item, blob := range props {
//pp := prop{Name: item}
//attrs := blob.(map[string]interface{})
//if t, ok := attrs["type"]; ok {
//pp.Type = t.(string)
//}
//if p, ok := attrs["properties"]; ok {
//pp.Props = findProps(p.(map[string]interface{}))
//}
//if copyto, ok := attrs["copy_to"]; ok {
//blacklist = append(blacklist, copyto)
//}
//res = append(res, pp)
//}
//return res
//}

type indexProp struct {
	Simpl, Compl []prop
}

func separate(props prop) indexProp {
	var simpl []prop
	var compl []prop
	for _, p := range props.Props {
		if p.Props == nil {
			simpl = append(simpl, p)
		} else {
			compl = append(compl, p)
		}
	}
	return indexProp{Simpl: simpl, Compl: compl}
}

var fmap2 = template.FuncMap{
	"title":     func(s string) string { return strings.Title(s) },
	"isComplex": func(p prop) bool { return p.Props != nil },
	"type": func(p prop) string {
		switch p.Type {
		case "string":
			return "string"
		case "boolean":
			return "bool"
		case "long":
			return "int"
		case "date":
			return "string"
		default:
			panic("unknown type: " + p.Type)
		}
	},
}

var dummyTempl = `{{ define "Tstruct" }} {{ .Name }} {{ end }}`

var structTempl = `{{ define "Tstruct" }}struct {
{{ range $q := .Props }} {{ title $q.Name }}{{ if isComplex $q }} {{ template "Tstruct" $q }} {{ else }} {{ type $q }} {{ end }}
{{ end }}
}{{ end }}`

var contentTempl = `
package main

type IndexStruct struct {
{{ range $p := .Simpl }} {{ title $p.Name }} {{ type $p }} {{ end }}
{{ range $p := .Compl }} {{ title $p.Name }} {{ template "Tstruct" $p }}
{{ end }}
}
`
