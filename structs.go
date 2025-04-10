package protobuf

type ProtoDecoded struct {
	Parts    []ProtoPart
	LeftOver []byte
}

type ProtoPart struct {
	ByteRange []int
	Type      int
	Field     int
	Value     interface{}
}
