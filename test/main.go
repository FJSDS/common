package main

import (
	"fmt"
	"math"
	"time"

	"github.com/MauriceGit/skiplist"
)

type Element struct {
	I int64
	J int32
}

func NewElement(i int64, j int32) *Element {
	return &Element{i, j}
}

// Implement the interface used in skiplist
func (e *Element) ExtractKey() float64 {
	return float64(e.I/1000000) + float64(e.J)/float64(math.MaxInt32)
}
func (e *Element) String() string {
	return fmt.Sprintf("%03d", e)
}

func main() {
	list := skiplist.New()

	// Insert some elements
	start := time.Now()
	s := start.UnixNano() / 1000000
	for i := s; i < s+10000100; i++ {
		list.Insert(NewElement(int64(i), int32(s+10000100-i)))
	}
	fmt.Println(time.Now().Sub(start))

	start = time.Now()
	for i := int64(0); i < 10000100; i++ {
		el := list.GetSmallestNode()
		list.Delete(el.GetValue())
		if i == 10000000 {
			fmt.Println(el.GetValue())
		}
	}
	fmt.Println(time.Now().Sub(start))
	n := time.Now().UnixNano() / int64(time.Millisecond)
	fmt.Println(n, n<<22>>22)
}
