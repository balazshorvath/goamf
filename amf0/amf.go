package amf0

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// Spec @ https://www.adobe.com/content/dam/acom/en/devnet/pdf/amf0-file-format-specification.pdf
type Marker byte

const (
	Number        Marker = 0x00 // 8 bytes IEEE-754 double - network/big endian
	Boolean              = 0x01 // byte, 0 false, true otherwise
	String               = 0x02 // 2 bytes for length, rest is the UTF-8 string
	Object               = 0x03 //
	Movieclip            = 0x04 // Reserved, not supported
	Null                 = 0x05
	Undefined            = 0x06
	Reference            = 0x07 // 2 bytes - If the exact same object appears more than once, points to an index in a table of previously serialized objects
	ECMAArray            = 0x08 // 4 bytes as assoc count, string keys -> need more info
	ObjectEnd            = 0x09 // 0x00, 0x00, 0x09 show the end of na object. This is not a regular type it is preceded by the 'Object' type. At least in theory.
	StrictArray          = 0x0A // 4 bytes for length
	Date                 = 0x0B // 2 bytes unsigned time zone, but it's not supported and should be set to 0, followed by 8 bytes of double timestamp of millis
	LongString           = 0x0C // 4 bytes for length, UTF-8
	Unsupported          = 0x0D // Error may be appropriate
	Recordset            = 0x0E // Reserved, not supported
	XmlDocument          = 0x0F // 4 bytes for length and UTF-8
	TypedObject          = 0x10 // 2 bytes for length of the 'class name', UTF-8, -> then object
	AvmPlusObject        = 0x11 // The marker indicated, that the following object is AMF3 encoded
)

// Value represents an AMF value with a type, a value and optionally a name.
// A TypedObject's name is it's class name.
// ECMAArrays and Objects have named properties.
// Reference types are already resolved, there are no such types to be found in this tree.
type Value struct {
	Marker Marker
	Name   string
	Value  interface{}
}

type Parser struct {
	reader     io.Reader
	references []*Value
	bytesRead  int
}

func New(reader io.Reader) *Parser {
	return &Parser{
		reader: reader,
	}
}

func (p *Parser) Parse() (value *Value, bytesRead int, err error) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(error)
			if !ok {
				err = errors.New(fmt.Sprintf("%v", r))
			} else {
				err = e
			}
		}
	}()

	data := p.readBytes(p.reader, 1)
	marker := Marker(data[0])
	value = &Value{
		Marker: marker,
	}
	p.parseValue(value)
	return value, p.bytesRead, nil
}

func (p *Parser) parseValue(value *Value) {
	switch value.Marker {
	case Number:
		value.Value = p.readDouble()
	case Boolean:
		data := p.readBytes(p.reader, 1)
		value.Value = data[0] != 0
	case LongString, XmlDocument, String:
		str, _ := p.readString(value.Marker)
		value.Value = str
	case Object:
		value.Value = p.parseProperties()
		p.references = append(p.references, value)
	case Null, Undefined:
		value.Value = nil
	case Reference:
		data := p.readBytes(p.reader, 2)
		index := binary.BigEndian.Uint16(data)
		if int(index) > len(p.references)-1 {
			panic("reference index is greater, than the amount of available reference objects")
		}
		ref := p.references[index]
		value.Value = ref.Value
		value.Marker = ref.Marker
	case ECMAArray:
		// Length ignored, because assoc arrays should have 'ObjectEnd'
		_ = p.readBytes(p.reader, 4)
		properties := p.parseProperties()
		value.Value = properties
		p.references = append(p.references, value)
	case StrictArray:
		// Length
		data := p.readBytes(p.reader, 4)
		length := int(binary.BigEndian.Uint32(data))
		// Marker
		data = p.readBytes(p.reader, 1)
		arrayMarker := Marker(data[0])
		// Collect
		var values []*Value
		for i := 0; i < length; i++ {
			arrayValue := &Value{
				Marker: arrayMarker,
			}
			p.parseValue(arrayValue)
			values = append(values, arrayValue)
		}
		value.Value = values
	case Date:
		// not supported
		_ = p.readBytes(p.reader, 2)
		data := p.readBytes(p.reader, 8)
		value.Value = math.Float64frombits(binary.BigEndian.Uint64(data))
	case TypedObject:
		// Class name
		name, _ := p.readString(String)
		value.Name = name
		// Props
		value.Value = p.parseProperties()
		p.references = append(p.references, value)
	case AvmPlusObject:
		panic("amf3 is not supported")
	case Unsupported, Recordset, Movieclip:
		panic(fmt.Sprintf("unsupported type %d", value.Marker))
	default:
	}
}

func (p *Parser) parseProperties() []*Value {
	var properties []*Value
	for {
		name, nameLength := p.readString(String)
		// Check if 'ObjectEnd'
		if nameLength == 0 {
			data := p.readBytes(p.reader, 1)
			// Should be always this way
			if data[0] == ObjectEnd {
				break
			} else {
				panic("i'm stupid and i fucked up")
			}
		}
		data := p.readBytes(p.reader, 1)
		marker := Marker(data[0])
		property := &Value{
			Marker: marker,
			Name:   name,
		}
		p.parseValue(property)
		properties = append(properties, property)
	}
	return properties
}

func (p *Parser) readDouble() float64 {
	data := p.readBytes(p.reader, 8)
	return math.Float64frombits(binary.BigEndian.Uint64(data))
}

func (p *Parser) readString(marker Marker) (string, int) {
	var nameLength int32
	if marker == String {
		data := p.readBytes(p.reader, 2)
		nameLength = int32(binary.BigEndian.Uint16(data))
	} else if marker == LongString || marker == XmlDocument {
		data := p.readBytes(p.reader, 4)
		nameLength = int32(binary.BigEndian.Uint32(data))
	}
	data := p.readBytes(p.reader, int(nameLength))
	return string(data), int(nameLength)
}

func (p *Parser) readBytes(reader io.Reader, length int) []byte {
	buffer := make([]byte, length)
	n, err := io.ReadFull(reader, buffer)
	p.bytesRead += n
	if err != nil {
		panic(err)
	}
	return buffer
}
