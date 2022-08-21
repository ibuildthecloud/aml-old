package ast

import (
	"strconv"
	"strings"
)

var ext = map[string]int{
	"k":  3,
	"m":  6,
	"g":  9,
	"t":  12,
	"p":  15,
	"e":  18,
	"z":  21,
	"y":  24,
	"ki": 10,
	"mi": 20,
	"gi": 30,
	"ti": 40,
	"pi": 50,
	"ei": 60,
	"zi": 70,
	"yi": 80,
}

func (n Number) Float64() (float64, error) {
	s, mul := n.parts()
	r, err := strconv.ParseFloat(s, 64)
	return r * float64(mul), err
}

func (n Number) Int64() (int64, error) {
	s, mul := n.parts()
	r, err := strconv.ParseInt(s, 10, 64)
	return r * mul, err
}

func (n Number) parts() (string, int64) {
	s := strings.ToLower(strings.ReplaceAll(string(n), "_", ""))
	for suffix, exp := range ext {
		if strings.HasSuffix(s, suffix) {
			ret := int64(1)
			base := int64(10)
			pow := exp
			if len(suffix) == 2 {
				base = 2
			}
			for i := 0; i < pow; i++ {
				ret *= base
			}
			return s[:len(s)-len(suffix)], ret
		}
	}
	return s, 1
}
