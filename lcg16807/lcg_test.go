package lcg16807

import (
	"math"
	"testing"
)

func TestRandFloat(t *testing.T) {
	r := RandInit(1)
	for i := 0; i < 10e6; i++ {
		f := r.RandFloat()
		if f < 0 || f > 1.0 {
			t.Fatal("out of limits: ", f)
		}
	}
}

func TestRandIntUniform(t *testing.T) {
	r := RandInit(1)

	tot := int32(0)
	for i := 0; i < 10e6; i++ {
		x := r.RandIntUniform(6, 10)
		if x < 6 || x > 10 {
			t.Fatal("out of limits: ", x)
		}
		tot += x
	}

	avg := float64(tot) / 10e6
	diff := math.Abs(8 - avg)
	if diff > 10e-5 {
		t.Fatal("mean too far from expected:", avg, diff)
	}
}
