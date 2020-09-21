package utils

import (
	"bytes"
	"math"
)

// characters used for conversion
const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// converts number to base62
func Base62Encode(number int64) string {
	if number == 0 {
		return string(alphabet[0])
	}

	chars := make([]byte, 0)
	length := int64(len(alphabet))

	for number > 0 {
		result := number / length
		remainder := number % length

		chars = append(chars, alphabet[remainder])
		number = result
	}

	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}

	return string(chars)
}

func Base62EncodeMin6Max11(number int64) string {
	const baseStr = "000000"
	str := Base62Encode(number)
	if len(str) < 6 {
		return baseStr[:6-len(str)] + str
	}
	return str
}

var Base62DecodeMin6Max11 = Base62Decode

// converts base62 token to int
func Base62Decode(token string) int64 {
	number := int64(0)
	idx := 0.0
	chars := []byte(alphabet)

	charsLength := float64(len(chars))
	tokenLength := float64(len(token))

	for _, c := range []byte(token) {
		power := tokenLength - (idx + 1)
		index := int64(bytes.IndexByte(chars, c))
		number += index * int64(math.Pow(charsLength, power))
		idx++
	}

	return number
}
