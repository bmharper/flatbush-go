package flatbush

import (
	"testing"
	"time"

	"math/rand"

	"github.com/stretchr/testify/require"
)

func TestEmpty(t *testing.T) {
	f := NewFlatbush()
	f.Finish()
	require.Equal(t, 0, len(f.Search(0, 0, 1, 1)))
}

func TestBasic(t *testing.T) {
	f := NewFlatbush()
	boxes := []Box64{}
	dim := 100
	f.Reserve(int(dim * dim))
	index := 0
	for x := float64(0); x < float64(dim); x++ {
		for y := float64(0); y < float64(dim); y++ {
			// boxes.push_back({index, x + 0.1f, y + 0.1f, x + 0.9f, y + 0.9f});
			boxes = append(boxes, Box64{
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
	pad := float64(3)
	for i := 0; i < nSamples; i++ {
		minx := rng.Float64()*float64(dim) - pad
		miny := rng.Float64()*float64(dim) - pad
		maxx := minx + rng.Float64()*float64(maxQueryWindow)
		maxy := miny + rng.Float64()*float64(maxQueryWindow)
		results := f.Search(minx, miny, maxx, maxy)
		totalResults += len(results)
		// brute force validation that there are no false negatives
		qbox := Box64{
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

func fillSquare(f *Flatbush64, sideLength int) {
	f.Reserve(int(sideLength * sideLength))
	for x := float64(0); x < float64(sideLength); x++ {
		for y := float64(0); y < float64(sideLength); y++ {
			f.Add(x+0.1, y+0.1, x+0.9, y+0.9)
		}
	}
	f.Finish()
}

func BenchmarkInsert(b *testing.B) {
	dim := 1000
	start := time.Now()
	f := NewFlatbush()
	fillSquare(f, dim)
	end := time.Now()
	b.Logf("Time to insert %v elements: %.0f milliseconds", dim*dim, end.Sub(start).Seconds()*1000)
}

func BenchmarkQuery(b *testing.B) {
	dim := 1000
	f := NewFlatbush()
	fillSquare(f, dim)

	start := time.Now()
	nquery := 10 * 1000 * 1000
	sx := 0
	sy := 0
	results := []int{}
	nresults := 0
	for i := 0; i < nquery; i++ {
		minx := float64(sx % dim)
		miny := float64(sy % dim)
		maxx := float64(minx + 5.0)
		maxy := float64(miny + 5.0)
		results = f.SearchFast(minx, miny, maxx, maxy, results)
		nresults += len(results)
		sx++
		sy++
	}
	elapsedS := time.Now().Sub(start).Seconds()
	b.Logf("Time per query, returning average of %.0f elements: %.2f nanoseconds\n", float64(nresults)/float64(nquery), elapsedS*1e9/float64(nquery))
}
