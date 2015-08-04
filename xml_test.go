package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/kr/pretty"
)

type testCase struct {
	xsd   string
	xml   xmlElem
	gosrc string
}

var (
	tests = []struct {
		xsd   string
		xml   xmlElem
		gosrc string
	}{
		{
			`<schema>
			<xs:element name="studio" type="xs:string" minOccurs="0">
			</xs:element>
</schema>`,
			xmlElem{
				Name:  "studio",
				Type:  "string",
				Cdata: true,
			},
			`
			type studio struct {
				Studio string ` + "`xml:\",chardata\"`" + `
			}
			`,
		},

		{
			`<schema>
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
			xmlElem{
				Name: "titleList",
				Type: "titleList",
				Children: []*xmlElem{
					&xmlElem{
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
			`
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
	}
)

func TestGenerateGo(t *testing.T) {
	for _, tst := range tests {

		var out bytes.Buffer
		doParse(&tst.xml, &out)
		if strings.Join(strings.Fields(out.String()), "") != strings.Join(strings.Fields(tst.gosrc), "") {
			t.Errorf("Unexpected generated Go source: %s", tst.xml.Name)
			t.Logf(out.String())
			t.Logf(strings.Join(strings.Fields(out.String()), ""))
			t.Logf(strings.Join(strings.Fields(tst.gosrc), ""))
		}
	}
}

func TestBuildXmlElem(t *testing.T) {
	for _, tst := range tests {
		schema, err := extract(bytes.NewBufferString(tst.xsd))
		if err != nil {
			t.Error(err)
		}
		b := newBuilder([]xsdSchema{schema})
		elems := b.buildXML()
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
