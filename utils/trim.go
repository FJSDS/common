package utils

func TrimLeftRight(str string) string {
	for len(str) > 0 {
		first := str[0]
		if first == ' ' || first == '\t' || first == '\n' {
			str = str[1:]
		} else {
			break
		}
	}
	for len(str) > 0 {
		lastIndex := len(str) - 1
		last := str[lastIndex]
		if last == ' ' || last == '\t' || last == '\n' {
			str = str[:lastIndex]
		} else {
			break
		}
	}
	return str
}
