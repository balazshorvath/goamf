package amf0

// ContextHeader
// Object references are local to each context header.
type NCContextHeader struct {
	NameLength     uint16
	HeaderName     string
	MustUnderstand uint8
	HeaderLength   uint32 // Optional; from the spec: "(U32)-1 to specify an unknown length". Which is probably max value of a 32 bit int.
	Value          Value
}

// Message
type NCMessage struct {
	TargetUriLength   uint16
	TargetUri         string
	ResponseUriLength uint16
	ResponseUri       string
	MessageLength     uint32 // Optional; from the spec: "(U32)-1 to specify an unknown length". Which is probably max value of a 32 bit int.
	Body              Value  //
}

// Packet represents multiple messages.
// Version = 0 for AMF0
type NCPacket struct {
	Version      uint16
	HeaderCount  uint16
	MessageCount uint16
	Headers      []*NCContextHeader
}

func ParseNetConnectionPacket(data []byte) (*NCPacket, error) {
	panic("Not supported")
}
