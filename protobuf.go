package protobuf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"unicode/utf8"
)

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
	Field     int
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

// #endregion
// #region Proto
func stringOrBytes(data []byte) interface{} {
	if len(data) == 0 {
		return string(data)
	}

	if utf8.Valid(data) {
		return string(data)
	}

	return data
}

// Target should be a pointer to a struct
func DecodeToProtoStruct(data []byte, target interface{}) error {
	// TODO: too tired for this right now
	return nil
}

// TODO: refactor this
func EncodeProtoStruct(data interface{}) []ProtoPart {
	var response []ProtoPart
	dType := reflect.TypeOf(data)
	dValue := reflect.ValueOf(data)

	if dType.Kind() == reflect.Ptr {
		dType = dType.Elem()
		dValue = dValue.Elem()
	}

	// Iterate through fields
	for i := 0; i < dType.NumField(); i++ {
		field := dType.Field(i)
		tag := field.Tag.Get("protoField")
		value := dValue.Field(i).Interface()

		if len(tag) == 0 {
			continue
		} // Ignore fields without a tag

		fieldNum, err := strconv.Atoi(tag)
		if err != nil {
			continue
		} // Ignore fields with invalid tags

		var part ProtoPart
		part.Field = fieldNum

		switch value.(type) {
		case int:
			part.Type = VARINT
			part.Value = value
			break
		case string:
			part.Type = LENDELIM
			part.Value = []byte(value.(string))
			break
		case bool:
			part.Type = VARINT
			if value.(bool) {
				part.Value = 1
			} else {
				part.Value = 0
			}
			break
		case []byte:
			part.Type = LENDELIM
			part.Value = value
			break
		case []interface{}:
			part.Type = LENDELIM
			part.Value = ArrayToProtoParts(value.([]interface{}))
			break
		case float32:
			part.Type = FIXED32
			part.Value = value
			break
		case float64:
			part.Type = FIXED64
			part.Value = value
			break
		}

		response = append(response, part)
	}

	return response
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

const scale = 1 << 16

func ArrayToProtoParts(data []interface{}) []ProtoPart {
	var res []ProtoPart

	for i, item := range data {
		var part ProtoPart
		part.Field = i + 1

		switch item.(type) {
		case int:
			part.Type = VARINT
			part.Value = item
			break
		case *big.Int:
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
		case []int:
			part.Type = LENDELIM
			part.Value = ArrayToProtoParts(item.([]interface{}))
			break
		case []string:
			part.Type = LENDELIM
			part.Value = ArrayToProtoParts(item.([]interface{}))
			break
		case float32:
			part.Type = FIXED32

			buf := make([]byte, 4)
			binary.LittleEndian.PutUint32(buf, uint32(int32(item.(float32)*scale)))

			part.Value = buf
			break
		case float64:
			part.Type = FIXED64

			buf := make([]byte, 8)
			binary.LittleEndian.PutUint64(buf, math.Float64bits(item.(float64)))

			part.Value = buf
			break
		case bool:
			part.Type = VARINT
			if item.(bool) {
				part.Value = 1
			} else {
				part.Value = 0
			}
			break
		}

		res = append(res, part)
	}

	return res
}

func EncodeProto(parts []ProtoPart) []byte {
	var buffer bytes.Buffer

	for _, part := range parts {
		// Write the type with index to the buffer
		buffer.WriteByte(byte(part.Field<<3 | part.Type))

		// buffer.WriteByte(byte(part.Type))

		switch part.Type {
		case VARINT:
			switch part.Value.(type) {
			case int:
				if part.Value == 0 {
					buffer.WriteByte(0) // Write 0 if the value is 0, didn't work previously for some reason
				}

				buffer.Write(encodeVarint(uint64(part.Value.(int))))
				break
			case int64:
				buffer.Write(encodeVarint(uint64(part.Value.(int64))))
				break
			case *big.Int:
				buffer.Write(encodeVarint(part.Value.(*big.Int).Uint64()))
				break
			}
			break
		case FIXED64:
			buffer.Write(part.Value.([]byte))
			break
		case LENDELIM:
			if part.Value == nil {
				break
			}

			switch part.Value.(type) {
			case []byte:
				length := len(part.Value.([]byte))
				buffer.Write(encodeVarint(uint64(length)))
				buffer.Write(part.Value.([]byte))
				break
			case string:
				length := len(part.Value.(string))
				buffer.Write(encodeVarint(uint64(length)))
				buffer.Write([]byte(part.Value.(string)))
				break
			case []ProtoPart:
				//fmt.Println(part.Value.([]ProtoPart))
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
		part.Field = int(fieldTypeRaw.Int64() >> 3)

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
//#region Testing
// func main() {
// 	str := "082a108082a6efc79e8491111dc3f54840216957148b0abf05402801321048656c6c6f2c2050726f746f627566213a0e50726f746f627566206279746573420501020304054a110a0d4e657374656420737472696e67106350005a080a046b65793110645a090a046b65793210c801"
// 	data, _ := hex.DecodeString(str)

// 	decoded := DecodeProto(data)
// 	ProtoPartsToArray(decoded.Parts)
// 	// fmt.Println(array)

// 	testBool := false

// 	parts := ArrayToProtoParts([]interface{}{
// 		42, 1234567890123456768, 123.456, 46.13303445314885481, 1, "Hello, Protobuf!", []byte("Protobuf bytes"), []interface{}{1, 2, 3, 4, 5}, []interface{}{
// 			"Nested string", 99}, testBool, []interface{}{"key1", 100}, []interface{}{"key2", 200}})
// 	// fmt.Println("Parts: %v, %v", parts, len(parts))

// 	EncodeProto(parts)
// 	// fmt.Println(hex.EncodeToString(encoded))

// }
//#endregion
