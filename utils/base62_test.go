package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBase62(t *testing.T) {
	r := require.New(t)
	excepted := int64(216335982)
	e := Base62EncodeMin6Max11(excepted)
	fmt.Println(e)
	actual := Base62Decode(e)
	fmt.Println(excepted, actual)
	r.Equal(excepted, actual)
	actual = Base62DecodeMin6Max11(e)
	fmt.Println(excepted, actual)
	r.Equal(excepted, actual)
}
