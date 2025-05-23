package protobuf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"
)

const (
	VARINT   = 0x00
	FIXED64  = 0x01
	LENDELIM = 0x02
	FIXED32  = 0x05
)

// #region Proto

// Target should be a pointer to a struct
func DecodeToProtoStruct(data []ProtoPart, target interface{}) error {
	getPartByFieldNum := func(fieldNum int) *ProtoPart {
		for _, part := range data {
			if part.Field == fieldNum {
				return &part
			}
		}
		return nil
	}

	dType := reflect.TypeOf(target).Elem()
	dValue := reflect.ValueOf(target).Elem()

	if dType.Kind() != reflect.Struct {
		return errors.New("target must be a pointer to a struct")
	}

	for i := 0; i < dType.NumField(); i++ {
		field := dType.Field(i)
		tag := field.Tag.Get("protoField")
		if tag == "" {
			continue
		}

		fieldNum, err := strconv.Atoi(tag)
		if err != nil {
			continue
		}

		part := getPartByFieldNum(fieldNum)
		if part == nil {
			continue
		}

		fieldValue := dValue.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		// fmt.Println("Field:", field.Name, "Field type ", field.Type.Kind(), "FieldNum:", fieldNum, "Part:", part)
		switch field.Type.Kind() {
		case reflect.Int, reflect.Int32, reflect.Int64:
			if part.Type == VARINT || part.Type == FIXED64 {
				var intVal int64
				switch v := part.Value.(type) {
				case *big.Int:
					intVal = v.Int64()
				case int, int32, uint32, int16, uint16, int8, uint8:
					intVal = int64(v.(int))
				case int64:
					intVal = v
				case []byte:
					decoded, _, err := decodeVarint(bytes.NewBuffer(v))
					if err != nil {
						return err
					}
					intVal = decoded.Int64()
				default:
					continue
				}
				fieldValue.SetInt(intVal)
			}
		case reflect.Uint, reflect.Uint32, reflect.Uint64:
			if part.Type == VARINT || part.Type == FIXED64 {
				var uintVal uint64
				switch v := part.Value.(type) {
				case *big.Int:
					uintVal = v.Uint64()
				case int, int32, uint32, int16, uint16, int8, uint8:
					uintVal = uint64(v.(int))
				case int64:
					uintVal = uint64(v)
				case []byte:
					decoded, _, err := decodeVarint(bytes.NewBuffer(v))
					if err != nil {
						return err
					}
					uintVal = decoded.Uint64()
				default:
					continue
				}
				fieldValue.SetUint(uintVal)
			}
		case reflect.String:
			if part.Type == LENDELIM {
				if v, ok := part.Value.([]byte); ok {
					fieldValue.SetString(string(v))
				}
			}
		case reflect.Bool:
			if part.Type == VARINT {
				var boolVal bool
				switch v := part.Value.(type) {
				case int, int64, uint64, int32, uint32, int16, uint16, int8, uint8:
					boolVal = v != 0
				case *big.Int:
					boolVal = v.Sign() != 0
				case []byte:
					decoded, _, err := decodeVarint(bytes.NewBuffer(v))
					if err != nil {
						return err
					}
					boolVal = decoded.Sign() != 0
				default:
					boolVal = false
				}
				fieldValue.SetBool(boolVal)
			}
		case reflect.Slice:
			if part.Type == LENDELIM {
				switch v := part.Value.(type) {
				case []ProtoPart:
					decoded := ProtoPartsToArray(v)
					fieldValue.Set(reflect.ValueOf(decoded))
				case []byte:
					// If the field is a byte slice, set it directly
					if fieldValue.Type().Elem().Kind() == reflect.Uint8 {
						fieldValue.SetBytes(v)
					} else {
						// Convert []byte to appropriate slice type
						// TODO: Do some testing to ensure this is always the case.. I'll be doomed if it isn't (so far it is)
						decoded := DecodeProto(v)
						newSlice := reflect.MakeSlice(fieldValue.Type(), len(decoded.Parts), len(decoded.Parts))
						for i, part := range decoded.Parts {
							switch part.Value.(type) {
							case *big.Int:
								part.Value = part.Value.(*big.Int).Int64()
							}
							val := reflect.ValueOf(part.Value)
							if val.Type().ConvertibleTo(fieldValue.Type().Elem()) {
								val = val.Convert(fieldValue.Type().Elem())
								newSlice.Index(i).Set(val)
							} else {
								panic(fmt.Sprintf("cannot convert %T to %v", part.Value, fieldValue.Type().Elem()))
							}
						}

						fieldValue.Set(newSlice)
					}
				}
			}
		case reflect.Float32:
			if part.Type == FIXED32 {
				if v, ok := part.Value.([]byte); ok && len(v) == 4 {
					bits := binary.LittleEndian.Uint32(v)
					fieldValue.SetFloat(float64(math.Float32frombits(bits)))
				}
			}
		case reflect.Float64:
			if part.Type == FIXED64 {
				if v, ok := part.Value.([]byte); ok && len(v) == 8 {
					bits := binary.LittleEndian.Uint64(v)
					fieldValue.SetFloat(math.Float64frombits(bits))
				}
			}
		case reflect.Struct:
			if part.Type == LENDELIM {
				decoded := DecodeProto(part.Value.([]byte))
				ptrToField := fieldValue.Addr().Interface()

				err := DecodeToProtoStruct(decoded.Parts, ptrToField)
				if err != nil {
					return err
				}
			}
		case reflect.Ptr:
			if part.Type == LENDELIM {
				structType := field.Type.Elem()
				newStruct := reflect.New(structType) // *StructType

				decoded := DecodeProto(part.Value.([]byte))
				err := DecodeToProtoStruct(decoded.Parts, newStruct.Interface())
				if err != nil {
					return err
				}

				fieldValue.Set(newStruct)
			}
		default:
			return errors.New("unsupported field type")
		}
	}

	return nil
}

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
		case int, int64, uint64, int32, uint32, int16, uint16, int8, uint8, *big.Int:
			part.Type = VARINT
			part.Value = value
			break
		case string:
			if len(value.(string)) == 0 {
				continue
			} // Ignore empty strings
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
		case float32:
			part.Type = FIXED32

			floatBytes := make([]byte, 4)
			binary.LittleEndian.PutUint32(floatBytes, math.Float32bits(value.(float32)))

			part.Value = floatBytes
			break
		case float64:
			part.Type = FIXED64

			floatBytes := make([]byte, 8)
			binary.LittleEndian.PutUint64(floatBytes, math.Float64bits(value.(float64)))

			part.Value = floatBytes
			break
		case []byte:
			part.Type = LENDELIM
			part.Value = value
			break
		case []interface{}, []string, []int, []int64, []uint64, []int32, []uint32, []int16, []uint16, []int8, []bool:
			part.Type = LENDELIM

			v := reflect.ValueOf(value)
			result := make([]interface{}, v.Len())
			for i := 0; i < v.Len(); i++ {
				result[i] = v.Index(i).Interface()
			}

			part.Value = ArrayToProtoParts(result)
			break
		default:
			// If struct / pointer to struct, encode it as a nested message
			if field.Type.Kind() == reflect.Struct || field.Type.Kind() == reflect.Ptr {
				part.Type = LENDELIM
				part.Value = EncodeProtoStruct(value)
			} else {
				continue // Ignore unsupported types
			}
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
		case int, int64, uint64, uint32, int32, uint16, int16, uint8, int8, *big.Int:
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
		case []int, []interface{}, []string:
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
		buffer.Write(encodeVarint(uint64(part.Field<<3 | part.Type)))

		switch part.Type {
		case VARINT:
			switch part.Value.(type) {
			case int, int32, uint32, uint16, int16, uint8, int8:
				if part.Value == 0 {
					buffer.WriteByte(0) // Write 0 if the value is 0, didn't work previously for some reason
				}

				buffer.Write(encodeVarint(uint64(part.Value.(int))))
				break
			case int64:
				buffer.Write(encodeVarint(uint64(part.Value.(int64))))
				break
			case uint64:
				buffer.Write(encodeVarint(part.Value.(uint64)))
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

		// Check if the field type is valid
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
			length, _, err := decodeVarint(&buffer)
			if err != nil {
				break
			}

			part.Value = buffer.Next(int(length.Uint64()))
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

// A simple wrapper to encode data, it doesn't check for errors/field numbers or anything else so it is unadvised to use it, just accomodate yourself to the functions it uses
func Encode(data []interface{}) []byte {
	parts := ArrayToProtoParts(data)
	return EncodeProto(parts)
}

// A simple wrapper to decode data, it doesn't check for errors/field numbers or even leftover bytes if there are any, it is unadvised to use it, just accomodate yourself to the functions it uses
func Decode(data []byte) []interface{} {
	decoded := DecodeProto(data)
	return ProtoPartsToArray(decoded.Parts)
}

func EncodeStruct(data interface{}) []byte {
	parts := EncodeProtoStruct(data)
	return EncodeProto(parts)
}

func DecodeStruct(data []byte, target interface{}) error {
	decoded := DecodeProto(data)
	return DecodeToProtoStruct(decoded.Parts, target)
}

//#endregion
