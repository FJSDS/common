package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMD5String(t *testing.T) {
	r := require.New(t)
	fmt.Println(MD5String("app_secret=gXJ0WLUocorpRgy8UqqwAc6Z37KQ3eWX&func=bet_result&timestamp=123123123123"))
	b := bytes.NewBuffer(nil)
	b.Grow(4096)

	err := json.NewEncoder(b).Encode(map[int64]interface{}{1: "123", 2: "1231"})
	r.NoError(err)
	fmt.Println(b.String())
}
