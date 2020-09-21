package utils

var baseStr = "0123456789ABCDEFGHJKLMNPQRSTUVWXYZ"
var baseMap = make(map[byte]uint64)

func init() {
	for i, v := range StringToBytes(baseStr) {
		baseMap[v] = uint64(i)
	}
}
func Base34EncodeToString(id uint64) string {
	return BytesToString(Base34Encode(id))
}

func Base34Encode(id uint64) []byte {
	quotient := id
	mod := uint64(0)
	l := make([]byte, 0, 13)
	for quotient != 0 {
		mod = quotient % 34
		quotient = quotient / 34
		l = append(l, baseStr[int(mod)])
	}
	ls := len(l)
	for i := 0; i < ls/2; i++ {
		l[i], l[ls-1-i] = l[ls-1-i], l[i]
	}
	if ls == 0 {
		l = append(l, '0')
	}
	return l
}

func Base34Decode(bs []byte) uint64 {
	ls := len(bs)
	if ls == 0 {
		return 0
	}
	var res, r uint64
	for i := ls - 1; i >= 0; i-- {
		v, ok := baseMap[bs[i]]
		if !ok {
			return 0
		}
		var b uint64 = 1
		for j := uint64(0); j < r; j++ {
			b *= 34
		}
		res += b * uint64(v)
		r++
	}
	return res
}

func Base34DecodeString(str string) uint64 {
	return Base34Decode(StringToBytes(str))
}
