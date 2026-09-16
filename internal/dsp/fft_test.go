package dsp

import (
	"math"
	"math/cmplx"
	"math/rand/v2"
	"testing"
)

func dft(a []complex128) []complex128 {
	n := len(a)
	out := make([]complex128, n)
	for k := 0; k < n; k++ {
		var sum complex128
		for t := 0; t < n; t++ {
			angle := -2 * math.Pi * float64(k) * float64(t) / float64(n)
			sum += a[t] * cmplx.Exp(complex(0, angle))
		}
		out[k] = sum
	}
	return out
}

func TestFFTMatchesDirectDFT(t *testing.T) {
	const n = 64
	r := rand.New(rand.NewPCG(1, 2))
	in := make([]complex128, n)
	for i := range in {
		in[i] = complex(r.NormFloat64(), r.NormFloat64())
	}

	want := dft(in)
	got := make([]complex128, n)
	copy(got, in)
	fft(got)

	var worst float64
	for k := range want {
		if e := cmplx.Abs(got[k] - want[k]); e > worst {
			worst = e
		}
	}
	if worst > 1e-9 {
		t.Errorf("max absolute error %g, want < 1e-9", worst)
	}
}

func TestFFTOfImpulseIsFlat(t *testing.T) {
	const n = 128
	a := make([]complex128, n)
	a[0] = 1
	fft(a)

	for k := range a {
		if e := math.Abs(cmplx.Abs(a[k]) - 1); e > 1e-12 {
			t.Errorf("bin %d: magnitude %g, want 1", k, cmplx.Abs(a[k]))
		}
	}
}

func TestFFTOfBinAlignedSine(t *testing.T) {
	const n = 256
	const bin = 7
	a := make([]complex128, n)
	for i := range a {
		a[i] = complex(math.Sin(2*math.Pi*bin*float64(i)/n), 0)
	}
	fft(a)

	for k := range a {
		mag := cmplx.Abs(a[k])
		if k == bin || k == n-bin {
			if math.Abs(mag-n/2) > 1e-9 {
				t.Errorf("bin %d: magnitude %g, want %d", k, mag, n/2)
			}
			continue
		}
		if mag > 1e-9 {
			t.Errorf("bin %d: magnitude %g, want 0", k, mag)
		}
	}
}

func TestFFTRoundTrip(t *testing.T) {
	const n = 64
	r := rand.New(rand.NewPCG(3, 4))
	in := make([]complex128, n)
	for i := range in {
		in[i] = complex(r.NormFloat64(), r.NormFloat64())
	}

	a := make([]complex128, n)
	copy(a, in)
	fft(a)
	for i := range a {
		a[i] = cmplx.Conj(a[i])
	}
	fft(a)

	for i := range a {
		got := cmplx.Conj(a[i])
		if e := cmplx.Abs(got - complex(n, 0)*in[i]); e > 1e-9 {
			t.Errorf("sample %d: got %v, want %v", i, got, complex(n, 0)*in[i])
		}
	}
}
