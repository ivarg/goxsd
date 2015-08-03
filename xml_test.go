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
			`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name='Root'>
    <xs:complexType>
      <xs:sequence>
        <xs:element name='Customers'>
          <xs:complexType>
            <xs:sequence>
              <xs:element name='Customer' type='CustomerType' minOccurs='0' maxOccurs='unbounded' />
            </xs:sequence>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:complexType name='CustomerType'>
    <xs:sequence>
      <xs:element name='CompanyName' type='xs:string'/>
    </xs:sequence>
    <xs:attribute name='CustomerID' type='xs:token'/>
    <xs:attribute name='MyInt' type='MyAttr'/>
  </xs:complexType>
  <xs:simpleType name='MyAttr'>
	<xs:restriction base="xs:integer">
    </xs:restriction>
  </xs:simpleType>
</xs:schema>
`,
			xmlElem{
				Name: "Root",
				Type: "Root",
				Children: []*xmlElem{
					&xmlElem{
						Name: "Customers",
						Type: "Customers",
						Children: []*xmlElem{
							&xmlElem{
								Name: "Customer",
								Type: "Customer",
								List: true,
								Attribs: []xmlAttrib{
									xmlAttrib{Name: "CustomerID", Type: "string"},
									xmlAttrib{Name: "MyInt", Type: "int"},
								},
								Children: []*xmlElem{
									&xmlElem{
										Name: "CompanyName",
										Type: "string",
									},
								},
							},
						},
					},
				},
			},
			`
type Root struct {
	Customers Customers ` + "`xml:\"Customers\"`" + `
}

type Customers struct {
	Customer []Customer ` + "`xml:\"Customer\"`" + `
}

type Customer struct {
	CustomerID string ` + "`xml:\"CustomerID,attr\"`" + `
	MyInt int ` + "`xml:\"MyInt,attr\"`" + `
	CompanyName string ` + "`xml:\"CompanyName\"`" + `
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
