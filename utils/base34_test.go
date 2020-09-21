package utils

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBase34(t *testing.T) {
	r := require.New(t)
	id := uint64(math.MaxUint64)
	c := Base34Encode(id)
	fmt.Println(string(c))
	id1 := Base34Decode(c)
	r.Equal(id, id1)
}
