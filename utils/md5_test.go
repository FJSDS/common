package utils

import (
	"fmt"
	"testing"
)

func TestMD5String(t *testing.T) {
	fmt.Println(MD5String("app_secret=gXJ0WLUocorpRgy8UqqwAc6Z37KQ3eWX&func=bet_result&timestamp=123123123123"))
}
