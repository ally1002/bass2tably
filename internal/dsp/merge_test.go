package dsp

import (
	"slices"
	"testing"
)

func TestMergeOnsets(t *testing.T) {
	const gap = 2646

	tests := []struct {
		name    string
		primary []int
		extra   []int
		want    []int
	}{
		{
			name:    "interleaved beyond the gap",
			primary: []int{0, 10000, 20000},
			extra:   []int{5000, 15000},
			want:    []int{0, 5000, 10000, 15000, 20000},
		},
		{
			name:    "exact tie keeps one",
			primary: []int{1000},
			extra:   []int{1000},
			want:    []int{1000},
		},
		{
			name:    "sub-gap cluster collapses to one",
			primary: nil,
			extra:   []int{1000, 1100, 1200, 5000},
			want:    []int{1000, 5000},
		},
		{
			name:    "extra inside the gap on either side is dropped",
			primary: []int{5000},
			extra:   []int{4000, 6000},
			want:    []int{5000},
		},
		{
			name:    "primary survives its own sub-gap spacing",
			primary: []int{0, 100},
			extra:   nil,
			want:    []int{0, 100},
		},
		{
			name:    "extra before the first primary",
			primary: []int{10000},
			extra:   []int{1000},
			want:    []int{1000, 10000},
		},
		{
			name:    "only extra",
			primary: nil,
			extra:   []int{3000},
			want:    []int{3000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeOnsets(tt.primary, tt.extra, gap)
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeOnsetsDoesNotAliasPrimary(t *testing.T) {
	primary := []int{0, 10000}
	got := mergeOnsets(primary, []int{5000}, 2646)
	got[0] = -1
	if primary[0] != 0 {
		t.Errorf("primary aliased: %v", primary)
	}
}
