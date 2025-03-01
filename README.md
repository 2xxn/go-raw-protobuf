Here’s an improved version of your README with better structure, clarity, and professionalism while maintaining a friendly tone:

---

# go-proto-raw  
A lightweight Go library for encoding and decoding Protocol Buffers (protobuf) without requiring `.proto` files.  

[![Go Reference](https://pkg.go.dev/badge/github.com/nextu1337/go-proto-raw.svg)](https://pkg.go.dev/github.com/nextu1337/go-proto-raw)  

This library provides a simple way to work with protobuf messages dynamically, eliminating the need for precompiled `.proto` definitions. It supports basic protobuf wire types and is designed for flexibility and ease of use.  

**Disclaimer**: This library is a work in progress and may have rough edges, but it gets the job done! Contributions and feedback are welcome.  

---

## Installation  
Add the library to your project using `go get`:  
```bash
go get github.com/nextu1337/go-proto-raw
```  

Or simply copy the file into your project.  

---

## Supported Types  
The library currently supports the following protobuf wire types:  
- **Varint** (encoded/decoded as `uint64`)  
- **Length-delimited** (encoded as `[]byte`, decoded as `[]byte`, `string`, or nested protobuf messages)  
- **Fixed32**  
- **Fixed64**  

---

## Usage  

### Encoding a Message  
Convert a slice of data into a protobuf-encoded byte slice:  
```go
data := []interface{}{123, "hello there!", []interface{}{123, "test"}}
encoded := EncodeProto(ArrayToProtoParts(data))
fmt.Println(hex.EncodeToString(encoded)) // Output: Protobuf-encoded hex string
```  

### Decoding a Message  
Decode a protobuf-encoded byte slice back into a slice of data:  
```go
data, _ := hex.DecodeString("08aefb8999d532120e496e697469616c20636f6d6d6974")
decoded := ProtoPartsToArray(DecodeProto(data).Parts)
fmt.Println(decoded) // Output: Decoded data as a slice
```  

---

## Contributing  
Found a bug or have an idea for improvement? Open an issue or submit a pull request!  

---
