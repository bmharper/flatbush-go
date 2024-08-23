package flatbush

import (
	"testing"
	"time"

	"math/rand"

	"github.com/stretchr/testify/require"
)

func TestEmpty32(t *testing.T) {
	f := NewFlatbush32()
	f.Finish()
	require.Equal(t, 0, len(f.Search(0, 0, 1, 1)))
}

func TestBasic32(t *testing.T) {
	f := NewFlatbush32()
	boxes := []Box32{}
	dim := 100
	f.Reserve(int(dim * dim))
	index := 0
	for x := float32(0); x < float32(dim); x++ {
		for y := float32(0); y < float32(dim); y++ {
			// boxes.push_back({index, x + 0.1f, y + 0.1f, x + 0.9f, y + 0.9f});
			boxes = append(boxes, Box32{
				MinX:  x + 0.1,
				MinY:  y + 0.1,
				MaxX:  x + 0.9,
				MaxY:  y + 0.9,
				Index: index,
			})
			checkIndex := f.Add(x+0.1, y+0.1, x+0.9, y+0.9)
			require.Equal(t, index, checkIndex)
			index++
		}
	}
	f.Finish()

	rng := rand.New(rand.NewSource(0))

	totalResults := 0
	nSamples := 1000
	maxQueryWindow := 5
	pad := float32(3)
	for i := 0; i < nSamples; i++ {
		minx := rng.Float32()*float32(dim) - pad
		miny := rng.Float32()*float32(dim) - pad
		maxx := minx + rng.Float32()*float32(maxQueryWindow)
		maxy := miny + rng.Float32()*float32(maxQueryWindow)
		results := f.Search(minx, miny, maxx, maxy)
		totalResults += len(results)
		// brute force validation that there are no false negatives
		qbox := Box32{
			MinX: minx,
			MinY: miny,
			MaxX: maxx,
			MaxY: maxy,
		}
		for _, b := range boxes {
			if b.PositiveUnion(&qbox) {
				// if object crosses the query rectangle, then it should be included in the result set
				found := false
				for _, r := range results {
					if r == b.Index {
						found = true
						break
					}
				}
				require.True(t, found)
			}
		}
	}
	require.Greater(t, totalResults, 0)
	require.Less(t, totalResults, (maxQueryWindow+3)*(maxQueryWindow+3)*nSamples) // +3 is just padding
}

func fillSquare32(f *Flatbush32, sideLength int) {
	f.Reserve(int(sideLength * sideLength))
	for x := float32(0); x < float32(sideLength); x++ {
		for y := float32(0); y < float32(sideLength); y++ {
			f.Add(x+0.1, y+0.1, x+0.9, y+0.9)
		}
	}
	f.Finish()
}

func BenchmarkInsert32(b *testing.B) {
	dim := 1000
	start := time.Now()
	f := NewFlatbush32()
	fillSquare32(f, dim)
	end := time.Now()
	b.Logf("Time to insert %v elements: %.0f milliseconds", dim*dim, end.Sub(start).Seconds()*1000)
}

func BenchmarkQuery32(b *testing.B) {
	dim := 1000
	f := NewFlatbush32()
	fillSquare32(f, dim)

	start := time.Now()
	nquery := 10 * 1000 * 1000
	sx := 0
	sy := 0
	results := []int{}
	nresults := 0
	for i := 0; i < nquery; i++ {
		minx := float32(sx % dim)
		miny := float32(sy % dim)
		maxx := float32(minx + 5.0)
		maxy := float32(miny + 5.0)
		results = f.SearchFast(minx, miny, maxx, maxy, results)
		nresults += len(results)
		sx++
		sy++
	}
	elapsedS := time.Now().Sub(start).Seconds()
	b.Logf("Time per query, returning average of %.0f elements: %.2f nanoseconds\n", float64(nresults)/float64(nquery), elapsedS*1e9/float64(nquery))
}
