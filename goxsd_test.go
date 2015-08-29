package main

import (
	"bytes"
	"encoding/xml"
	"reflect"
	"strings"
	"testing"

	"github.com/kr/pretty"
)

type testCase struct {
	xsd   string
	xml   xmlTree
	gosrc string
}

var (
	tests = []struct {
		exported bool
		prefix   string
		xsd      string
		xml      xmlTree
		gosrc    string
	}{

		{
			exported: false,
			prefix:   "",
			xsd: `<schema>
	<element name="titleList" type="titleListType">
	</element>
	<complexType name="titleListType">
		<sequence>
			<element name="title" type="originalTitleType" maxOccurs="unbounded" />
		</sequence>
	</complexType>
	<complexType name="originalTitleType">
		<simpleContent>
			<extension base="titleType">
				<attribute name="original" type="boolean">
				</attribute>
			</extension>
		</simpleContent>
	</complexType>
	<complexType name="titleType">
		<simpleContent>
			<restriction base="textType">
				<maxLength value="300" />
			</restriction>
		</simpleContent>
	</complexType>
	<complexType name="textType">
		<simpleContent>
			<extension base="string">
				<attribute name="language" type="language">
				</attribute>
			</extension>
		</simpleContent>
	</complexType>
</schema>`,
			xml: xmlTree{
				Name: "titleList",
				Type: "titleList",
				Children: []*xmlTree{
					&xmlTree{
						Name:  "title",
						Type:  "string",
						Cdata: true,
						List:  true,
						Attribs: []xmlAttrib{
							{Name: "language", Type: "string"},
							{Name: "original", Type: "bool"},
						},
					},
				},
			},
			gosrc: `
type titleList struct {
	Title []title ` + "`xml:\"title\"`" + `
}

type title struct {
	Language string ` + "`xml:\"language,attr\"`" + `
	Original bool ` + "`xml:\"original,attr\"`" + `
	Title    string ` + "`xml:\",chardata\"`" + `
}

				`,
		},

		{
			exported: false,
			prefix:   "",
			xsd: `<schema>
	<element name="tagList">
		<complexType>
			<sequence>
				<element name="tag" type="tagReferenceType" minOccurs="0" maxOccurs="unbounded" />
			</sequence>
		</complexType>
	</element>
	<complexType name="tagReferenceType">
		<simpleContent>
			<extension base="nidType">
				<attribute name="type" type="tagTypeType" use="required" />
			</extension>
		</simpleContent>
	</complexType>
	<simpleType name="nidType">
		<restriction base="string">
			<pattern value="[0-9a-zA-Z\-]+" />
		</restriction>
	</simpleType>
	<simpleType name="tagTypeType">
		<restriction base="string">
		</restriction>
	</simpleType>
</schema>`,
			xml: xmlTree{
				Name: "tagList",
				Type: "tagList",
				Children: []*xmlTree{
					&xmlTree{
						Name:  "tag",
						Type:  "string",
						List:  true,
						Cdata: true,
						Attribs: []xmlAttrib{
							{Name: "type", Type: "string"},
						},
					},
				},
			},
			gosrc: `
type tagList struct {
	Tag []tag ` + "`xml:\"tag\"`" + `
}

type tag struct {
	Type string ` + "`xml:\"type,attr\"`" + `
	Tag string ` + "`xml:\",chardata\"`" + `
}
			`,
		},

		{
			exported: false,
			prefix:   "",
			xsd: `<schema>
				<element name="tagId" type="tagReferenceType" />
	<complexType name="tagReferenceType">
		<simpleContent>
			<extension base="string">
				<attribute name="type" type="string" use="required" />
			</extension>
		</simpleContent>
	</complexType>
</schema>`,
			xml: xmlTree{
				Name:  "tagId",
				Type:  "string",
				List:  false,
				Cdata: true,
				Attribs: []xmlAttrib{
					{Name: "type", Type: "string"},
				},
			},
			gosrc: `
type tagID struct {
	Type string ` + "`xml:\"type,attr\"`" + `
	TagID string ` + "`xml:\",chardata\"`" + `
}
			`,
		},

		{
			exported: true,
			prefix:   "xxx",
			xsd: `<schema>
	<element name="url" type="tagReferenceType" />
	<complexType name="tagReferenceType">
		<simpleContent>
			<extension base="string">
				<attribute name="type" type="string" use="required" />
			</extension>
		</simpleContent>
	</complexType>
</schema>`,
			xml: xmlTree{
				Name:  "url",
				Type:  "string",
				List:  false,
				Cdata: true,
				Attribs: []xmlAttrib{
					{Name: "type", Type: "string"},
				},
			},
			gosrc: `
type XxxURL struct {
	Type string ` + "`xml:\"type,attr\"`" + `
	URL string ` + "`xml:\",chardata\"`" + `
}
			`,
		},
	}
)

func removeComments(buf bytes.Buffer) bytes.Buffer {
	lines := strings.Split(buf.String(), "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "//") {
			lines = append(lines[:i], lines[i+1:]...)
		}
	}
	return *bytes.NewBufferString(strings.Join(lines, "\n"))
}

func TestBuildXmlElem(t *testing.T) {
	for _, tst := range tests {
		var schema xsdSchema
		if err := xml.Unmarshal([]byte(tst.xsd), &schema); err != nil {
			t.Error(err)
		}

		bldr := builder{
			schemas:    []xsdSchema{schema},
			complTypes: make(map[string]xsdComplexType),
			simplTypes: make(map[string]xsdSimpleType),
		}
		elems := bldr.buildXML()
		if len(elems) != 1 {
			t.Errorf("wrong number of xml elements")
		}
		e := elems[0]
		if !reflect.DeepEqual(tst.xml, *e) {
			t.Errorf("Unexpected XML element: %s", e.Name)
			pretty.Println(tst.xml)
			pretty.Println(e)
		}
	}
}

func TestGenerateGo(t *testing.T) {
	for _, tst := range tests {
		var out bytes.Buffer
		g := generator{prefix: tst.prefix, exported: tst.exported}
		g.do(&out, []*xmlTree{&tst.xml})
		out = removeComments(out)
		if strings.Join(strings.Fields(out.String()), "") != strings.Join(strings.Fields(tst.gosrc), "") {
			t.Errorf("Unexpected generated Go source: %s", tst.xml.Name)
			t.Logf(out.String())
			t.Logf(strings.Join(strings.Fields(out.String()), ""))
			t.Logf(strings.Join(strings.Fields(tst.gosrc), ""))
		}
	}
}

func TestLintTitle(t *testing.T) {
	for i, tt := range []struct {
		input, want string
	}{
		{"foo cpu baz", "FooCPUBaz"},
		{"test Id", "TestID"},
		{"json and html", "JSONAndHTML"},
	} {
		if got := lintTitle(tt.input); got != tt.want {
			t.Errorf("[%d] title(%q) = %q, want %q", i, tt.input, got, tt.want)
		}
	}
}

func TestSquish(t *testing.T) {
	for i, tt := range []struct {
		input, want string
	}{
		{"Foo CPU Baz", "FooCPUBaz"},
		{"Test ID", "TestID"},
		{"JSON And HTML", "JSONAndHTML"},
	} {
		if got := squish(tt.input); got != tt.want {
			t.Errorf("[%d] squish(%q) = %q, want %q", i, tt.input, got, tt.want)
		}
	}
}

func TestReplace(t *testing.T) {
	for i, tt := range []struct {
		input, want string
	}{
		{"foo Cpu baz", "foo CPU baz"},
		{"test Id", "test ID"},
		{"Json and Html", "JSON and HTML"},
	} {
		if got := initialisms.Replace(tt.input); got != tt.want {
			t.Errorf("[%d] replace(%q) = %q, want %q", i, tt.input, got, tt.want)
		}
	}

	c := len(initialismPairs)

	for i := 0; i < c; i++ {
		input, want := initialismPairs[i], initialismPairs[i+1]

		if got := initialisms.Replace(input); got != want {
			t.Errorf("[%d] replace(%q) = %q, want %q", i, input, got, want)
		}

		i++
	}
}
