Here’s an improved version of your README with better structure, clarity, and professionalism while maintaining a friendly tone:

---

# go-raw-protobuf
A lightweight Go library for encoding and decoding Protocol Buffers (protobuf) without requiring `.proto` files.  

[![Go Reference](https://pkg.go.dev/badge/github.com/nextu1337/go-raw-protobuf.svg)](https://pkg.go.dev/github.com/nextu1337/go-raw-protobuf)  

This library provides a simple way to work with protobuf messages dynamically, eliminating the need for precompiled `.proto` definitions. It supports basic protobuf wire types and is designed for flexibility and ease of use.  

**Disclaimer**: This library is a work in progress and may have rough edges, but it gets the job done! Contributions and feedback are welcome.  

---

## Installation  
Add the library to your project using `go get`:  
```bash
go get github.com/nextu1337/go-raw-protobuf
```  

Or simply copy the file into your project.  

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

---

## Contributing  
Found a bug or have an idea for improvement? Open an issue or submit a pull request!  

---
