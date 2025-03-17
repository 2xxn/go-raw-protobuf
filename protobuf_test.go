package protobuf

import (
	"encoding/hex"
	"testing"
)

// Please don't pass structs with unexported fields, it will panic
type ProtoStruct struct {
	Id       int    `protoField:"1"`
	Username string `protoField:"2"`
	Email    string `protoField:"3"`
	IsAdmin  bool   `protoField:"5"`
}

func TestEncodeProtoStruct(t *testing.T) {
	// Test encoding
	data := &ProtoStruct{
		Id:       4588743,
		Username: "hello",
		Email:    "admin@example.com",
		IsAdmin:  true,
	}

	parts := EncodeProtoStruct(data)
	encoded := EncodeProto(parts)

	t.Logf("Encoded: %v", hex.EncodeToString(encoded))
}

func TestEncoding(t *testing.T) {
	// Test encoding
	data := []interface{}{
		1,
		"hello",
		[]interface{}{
			1,
			2,
			3,
		},
	}
	parts := ArrayToProtoParts(data)
	encoded := EncodeProto(parts)
	if encoded == nil {
		t.Errorf("Failed to encode data")
	}

	if hex.EncodeToString(encoded) != "0801120568656c6c6f1a06080110021803" {
		t.Errorf("Failed to encode data correctly") // Unless..? Should be checked every major update
	}
}

func TestDecoding(t *testing.T) {
	// Test decoding
	str := "0801120568656c6c6f1a06080110021803"
	data, _ := hex.DecodeString(str)
	decoded := DecodeProto(data)

	if len(decoded.LeftOver) > 0 {
		t.Errorf("Failed to decode data correctly")
	}

	if len(decoded.Parts) != 3 {
		t.Errorf("Failed to decode data correctly")
	}

	// t.Logf("Decoded: %v", ProtoPartsToArray(decoded.Parts))
}
