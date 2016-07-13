# goxsd

Goxsd is a tool for generating XML decoding Go structs from an [XSD schema](https://www.w3.org/TR/xmlschema11-1) specification. It does not yet implement the full XSD specificaiton, but goes sufficiently far to be useful for at least some use cases.

## Installation

```
go get github.com/ivarg/goxsd
```

## Usage

goxsd will default its output to stdout if an output file name is not given. Apart from a destination file, goxsd also accepts an export flag to toggle generation of exported struct names on (default is to generate unexported structs), and a prefix to be prepended to each struct name.

Any import statement in the XSD will be parsed and followed, interpreting the path as relative to the current XSD file.

```
Usage: goxsd [options] <xsd_file>

Options:
  -o <file>     Destination file [default: stdout]
  -p <package>  Package name [default: goxsd]
  -e            Generate exported structs [default: false]
  -x <prefix>   Struct name prefix [default: ""]

goxsd is a tool for generating XML decoding/encoding Go structs, according
to an XSD schema.
```

## TODOs

* Complete handling of more XSD elements is needed

* XSD namespaces are currently completely ignored, opening for undefined behavior if two namespaces are parsed with conflicting element- or type names.

* At some point, I would also like to generate validation code, that could check various rules and constraints expressed in the XSD

## License

The MIT License (MIT)

Copyright (c) 2015 Ivar Gaitan

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
