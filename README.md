# go-raw-protobuf
A lightweight Go library for encoding and decoding Protocol Buffers (protobuf) without requiring `.proto` files.  

[![Go Reference](https://pkg.go.dev/badge/github.com/2xxn/go-raw-protobuf.svg)](https://pkg.go.dev/github.com/2xxn/go-raw-protobuf)  

This library provides a simple way to work with protobuf messages dynamically, eliminating the need for precompiled `.proto` definitions. It supports basic protobuf wire types and is designed for flexibility and ease of use.  

**Disclaimer**: This library is a work in progress and may have rough edges, but it gets the job done! Contributions and feedback are welcome.  

---

## Installation  
Add the library to your project using `go get`:  
```bash
go get github.com/2xxn/go-raw-protobuf
```  

<!-- Or simply copy the file into your project.   -->
<!-- WILL BE UNSUPPORTED AS OF v2.0.0 -->

---

## Supported Types  
The library currently supports the following protobuf wire types:  

Decoding:
- **Varint** (decoded as `uint64`)
- **Length-delimited** (decoded as `[]byte` or `string` if it passes utf-8 validation)
- **Fixed32** as `[4]byte`
- **Fixed64** as `[8]byte`

Encoding:
- integers (`int`, `int32`, `int64`, `uint`, `uint32`, `uint64`, `bool`) (encoded as varint)
- floating-point numbers (`float32`, `float64`) (encoded as LittleEndian fixed32/fixed64)
- strings (`string`) (encoded as length-delimited)
- byte slices (`[]byte`) (encoded as length-delimited)
- arrays (`[]interface{}`) (encoded as length-delimited, nested message)
- nested arrays (`[][]interface{}`) (encoded as length-delimited, nested message)


---

## Usage  
If you want to verify the encoded data manually, feel free to visit [protobuf-decoder.netlify.app](https://protobuf-decoder.netlify.app/)
### Encoding a Message without a struct
Convert a slice of data into a protobuf-encoded byte slice:  
```go
data := []interface{}{123.456, "hello there!", []interface{}{true, "test"}}
encoded := Encode(data)
fmt.Println(hex.EncodeToString(encoded)) // Output: Protobuf-encoded hex string
```  

### Decoding a Message without a struct
Decode a protobuf-encoded byte slice back into a slice of data:  
```go
data, _ := hex.DecodeString("08aefb8999d532120e496e697469616c20636f6d6d6974")
decoded := Decode(data)
fmt.Println(decoded) // Output: Decoded data as a slice
```  

### Encoding a Message with a struct
```go
type ProtoStruct struct {
	Id             int     `protoField:"1"`
	Username       string  `protoField:"2"`
	Email          string  `protoField:"3"`
	TestFloat      float32 `protoField:"4"`
	IsAdmin        bool    `protoField:"5"`
	TestOtherFloat float64 `protoField:"6"`
}

data := &ProtoStruct{
	Id:             4588743,
	Username:       "hello",
	Email:          "admin@example.com",
	TestFloat:      1.2,
	TestOtherFloat: 1.23456789,
	IsAdmin:        true,
}

encoded := EncodeStruct(data)
fmt.Println(hex.EncodeToString(encoded)) // Output: Protobuf-encoded hex string
```

### Decoding a Message with a struct
```go
type ProtoStruct struct {
    Id             int     `protoField:"1"`
    Username       string  `protoField:"2"`
    Email          string  `protoField:"3"`
    TestFloat      float32 `protoField:"4"`
    IsAdmin        bool    `protoField:"5"`
    TestOtherFloat float64 `protoField:"6"`
}

data, _ := hex.DecodeString("08c7899802120568656c6c6f1a1161646d696e406578616d706c652e636f6d259a99993f2801311bde8342cac0f33f")

var s ProtoStruct
decoded := DecodeStruct(decoded.Parts, &s) // DecodeToProtoStruct will panic if the struct is not valid

fmt.Printf("%+v\n", s)
```

<details>
<summary>Old-fashioned way</summary>
<br>

### Encoding a Message  
Convert a slice of data into a protobuf-encoded byte slice:  
```go
data := []interface{}{123.456, "hello there!", []interface{}{true, "test"}}
encoded := EncodeProto(ArrayToProtoParts(data))
fmt.Println(hex.EncodeToString(encoded)) // Output: Protobuf-encoded hex string
```  

### Decoding a Message  
Decode a protobuf-encoded byte slice back into a slice of data:  
```go
data, _ := hex.DecodeString("08aefb8999d532120e496e697469616c20636f6d6d6974")
decoded := ProtoPartsToArray(DecodeProto(data).Parts)
// Loop through DecodeProto(data).Parts yourself if dealing with floating-point numbers etc or else fixed32/fixed64 will be returned as []byte and no different from utf8 invalid length-delimited data
fmt.Println(decoded) // Output: Decoded data as a slice
```  
</details>

---

## Contributing  
Found a bug or have an idea for improvement? Open an issue or submit a pull request!  

---
