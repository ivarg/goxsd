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
			`<schema><element name="ivar"></element></schema>`,
			xmlElem{
				Name: "ivar",
				Type: "ivar",
			},
			"type ivar struct {}",
		},
		{
			`<schema><element name="ivar" type="string"></element></schema>`,
			xmlElem{
				Name: "ivar",
				Type: "string",
			},
			"",
		},

		{
			`<schema>
	<element name="titleList" type="titleListType">
		<annotation>
			<documentation>Title list for content. At least one occurrence with the language used in the
				market/country displaying the content.
			</documentation>
		</annotation>
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
					<annotation>
						<documentation>Marks a title as the original title
						</documentation>
					</annotation>
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
					<annotation>
						<documentation>code in ISO 639-1</documentation>
					</annotation>
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
						Name: "title",
						Type: "string",
						List: true,
						Attribs: []xmlAttrib{
							{Name: "language", Type: "string"},
							{Name: "original", Type: "bool"},
						},
					},
				},
			},
			`
type titleList struct {
	Title []string ` + "`xml:\"title\"`" + `
}
				`,
		},
	}
)

func TestBuildXml(t *testing.T) {
	// Generate and write Go structs from XSD
	//for _, xsd := range xsds {
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

		var out bytes.Buffer
		doParse(e, &out)
		//t.Log(out.String())
		if strings.Join(strings.Fields(out.String()), "") != strings.Join(strings.Fields(tst.gosrc), "") {
			t.Errorf("Unexpected generated Go source: %s", e.Name)
			t.Logf(strings.Join(strings.Fields(out.String()), ""))
			t.Logf(strings.Join(strings.Fields(tst.gosrc), ""))
		}
	}
}
