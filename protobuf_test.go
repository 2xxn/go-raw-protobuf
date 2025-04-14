package protobuf

import (
	"encoding/hex"
	"fmt"
	"testing"
)

// Please don't pass structs with unexported fields, it will panic
type NestedMessage struct {
	Id   int    `protoField:"1"`
	Name string `protoField:"2"`
}

type ProtoStruct struct {
	Id             int            `protoField:"1"`
	Username       string         `protoField:"2"`
	Email          string         `protoField:"3"`
	TestFloat      float32        `protoField:"4"`
	IsAdmin        bool           `protoField:"5"`
	TestOtherFloat float64        `protoField:"6"`
	NestedStruct   *NestedMessage `protoField:"7"`
}

func TestEncodeProtoStruct(t *testing.T) {
	// Test encoding
	data := &ProtoStruct{
		Id:             4588743,
		Username:       "hello",
		Email:          "admin@example.com",
		TestFloat:      1.2,
		TestOtherFloat: 1.23456789,
		IsAdmin:        true,
		NestedStruct: &NestedMessage{
			Id:   1,
			Name: "hello",
		},
	}

	parts := EncodeProtoStruct(data)
	encoded := EncodeProto(parts)

	t.Logf("Encoded: %v", hex.EncodeToString(encoded))
}

func TestDecodeProtoStruct(t *testing.T) {
	// Test decoding
	str := "08c7899802120568656c6c6f1a1161646d696e406578616d706c652e636f6d259a99993f2801311bde8342cac0f33f3a090801120568656c6c6f"
	data, _ := hex.DecodeString(str)

	var s ProtoStruct
	decoded := DecodeProto(data)
	fmt.Printf("Decoded: %+v\n", decoded.Parts)
	DecodeToProtoStruct(decoded.Parts, &s) // DecodeToProtoStruct will panic if the struct is not valid
	t.Logf("Decoded: %v", s)
	t.Logf("Decoded: %+v", *s.NestedStruct)
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
