package protobuf

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"unicode/utf8"
)

//#region README
/*
github.com/nextu1337's Go snippet for encoding and decoding Protobuf messages without proto files.
Feel free to use this code in your projects, it's atroucious but it (kinda) works.

Can be downloaded as a library using `go get github.com/nextu1337/go-proto-raw` or by adding the file to your project.

Supported types:
- Varint (enc/dec as uint64)
- Length-delimited (enc as []byte, decode as []byte, string or protobuf message)
- Fixed32
- Fixed64

Usage:
- Encode a message:

data := []interface{}{123, "hello there!", []interface{}{123, "test"}}
fmt.Println(hex.EncodeToString(EncodeProto(ArrayToProtoParts(data))))

- Decode a message:

data, _ := hex.DecodeString("007b020c68656c6c6f207468657265210208007b020474657374")
fmt.Println(ProtoPartsToArray(DecodeProto(data).Parts))
*/
//#endregion

const (
	VARINT   = 0x00
	FIXED64  = 0x01
	LENDELIM = 0x02
	FIXED32  = 0x05
)

type ProtoDecoded struct {
	Parts    []ProtoPart
	LeftOver []byte
}

type ProtoPart struct {
	ByteRange []int
	Type      int
	Value     interface{}
}

//#region ZigZag

type ZigZag struct{}

func NewZigZag() *ZigZag {
	return &ZigZag{}
}

func (z *ZigZag) EncodeInt32(n int) uint64 {
	return uint64((uint32(n) << 1) ^ uint32((n >> 31)))
}

func (z *ZigZag) DecodeSint32(n uint64) int {
	return int((n >> 1) ^ -(n & 1))
}

//#endregion
//#region Varint

func encodeVarint(value uint64) []byte {
	var buffer bytes.Buffer

	for value > 0 {
		sevenBits := value & 0x7f
		value >>= 7

		if value > 0 {
			sevenBits |= 0x80
		}

		buffer.WriteByte(byte(sevenBits))
	}

	return buffer.Bytes()
}

func decodeVarint(buffer *bytes.Buffer) (*big.Int, int, error) {
	res := big.NewInt(0)
	shift := 0
	bytesRead := 0

	for {
		byteRead, err := buffer.ReadByte()
		if err != nil {
			return nil, 0, errors.New("unexpected EOF while decoding varint")
		}

		bytesRead++

		multiplier := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(shift)), nil)
		thisByteValue := new(big.Int).Mul(big.NewInt(int64(byteRead&0x7f)), multiplier)
		res.Add(res, thisByteValue)
		shift += 7

		if byteRead < 0x80 {
			break
		}
	}

	return res, bytesRead, nil
}

//#endregion
//#region Proto

func stringOrBytes(data []byte) interface{} {
	if len(data) == 0 {
		return string(data)
	}

	if utf8.Valid(data) {
		return string(data)
	}

	return data
}

func ProtoPartsToArray(parts []ProtoPart) []interface{} {
	var res []interface{}

	for _, part := range parts {
		if part.Type == LENDELIM {
			decoded := DecodeProto(part.Value.([]byte))

			if len(part.Value.([]byte)) > 0 && len(decoded.LeftOver) == 0 {
				res = append(res, ProtoPartsToArray(decoded.Parts))
				continue
			}

			value := stringOrBytes(part.Value.([]byte))
			res = append(res, value)

			continue
		}

		res = append(res, part.Value)
	}

	return res
}

func ArrayToProtoParts(data []interface{}) []ProtoPart {
	var res []ProtoPart

	for _, item := range data {
		var part ProtoPart

		switch item.(type) {
		case int:
			part.Type = VARINT
			part.Value = item
			break
		case int64:
			part.Type = VARINT
			part.Value = item
			break
		case string:
			part.Type = LENDELIM
			part.Value = []byte(item.(string))
			break
		case []byte:
			part.Type = LENDELIM
			part.Value = item
			break
		case []interface{}:
			part.Type = LENDELIM
			part.Value = ArrayToProtoParts(item.([]interface{}))
			break
		}

		res = append(res, part)
	}

	return res
}

func EncodeProto(parts []ProtoPart) []byte {
	var buffer bytes.Buffer

	for _, part := range parts {
		// Write the type to the buffer
		buffer.WriteByte(byte(part.Type))

		switch part.Type {
		case VARINT:
			// convert int to uint64
			buffer.Write(encodeVarint(uint64(part.Value.(int))))
			break
		case FIXED64:
			buffer.Write(part.Value.([]byte))
			break
		case LENDELIM:
			if part.Value == nil {
				break
			}

			fmt.Println(part.Value)

			switch part.Value.(type) {
			case []byte:
				length := len(part.Value.([]byte))
				buffer.Write(encodeVarint(uint64(length)))
				buffer.Write(part.Value.([]byte))
				break
			case []ProtoPart:
				fmt.Println(part.Value.([]ProtoPart))
				encoded := EncodeProto(part.Value.([]ProtoPart))
				length := len(encoded)
				buffer.Write(encodeVarint(uint64(length)))
				buffer.Write(encoded)
				break
			}

			break
		case FIXED32:
			buffer.Write(part.Value.([]byte))
			break
		}
	}

	return buffer.Bytes()
}

func DecodeProto(data []byte) ProtoDecoded {
	var buffer bytes.Buffer
	var response ProtoDecoded
	var parts []ProtoPart
	buffer.Write(data)

	totalBytes := buffer.Len()

	for buffer.Len() > 0 {
		var part ProtoPart

		// Use current offset (amount of already read)
		part.ByteRange = []int{totalBytes - buffer.Len()}

		fieldTypeRaw, _, err := decodeVarint(&buffer)
		if err != nil {
			break
		}

		fieldType := int(fieldTypeRaw.Int64() & 0x07)

		if fieldType > LENDELIM && fieldType != FIXED32 {
			break
		}

		switch fieldType {
		case VARINT:
			value, _, err := decodeVarint(&buffer)
			if err != nil {
				break
			}

			part.Value = value
			break
		case FIXED64:
			part.Value = buffer.Next(8)
			break
		case LENDELIM:
			length := buffer.Next(1)[0]
			data := buffer.Next(int(length))

			part.Value = data
			break
		case FIXED32:
			part.Value = buffer.Next(4)
			break
		default:
			break // Unknown type
		}

		part.Type = fieldType
		part.ByteRange = append(part.ByteRange, totalBytes-buffer.Len())
		parts = append(parts, part)
	}

	response.Parts = parts
	response.LeftOver = buffer.Bytes()

	return response
}

//#endregion
