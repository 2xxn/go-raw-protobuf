package protobuf

import (
	"bytes"
	"errors"
	"math/big"
	"unicode/utf8"
)

type ZigZag struct{}

func NewZigZag() *ZigZag {
	return &ZigZag{}
}

// EncodeInt32 encodes an int32 value into a zigzag-encoded uint64 value.
func (z *ZigZag) EncodeInt32(n int) uint64 {
	return uint64((uint32(n) << 1) ^ uint32((n >> 31)))
}

// DecodeSint32 decodes a zigzag-encoded uint64 value into an int32 value.
func (z *ZigZag) DecodeSint32(n uint64) int {
	return int((n >> 1) ^ -(n & 1))
}

func stringOrBytes(data []byte) interface{} {
	if len(data) == 0 {
		return string(data)
	}

	if utf8.Valid(data) {
		return string(data)
	}

	return data
}

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
