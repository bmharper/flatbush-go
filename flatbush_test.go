package flatbush

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEmpty(t *testing.T) {
	testEmpty[float32](t)
	testEmpty[float64](t)
}

func testEmpty[TFloat float32 | float64](t *testing.T) {
	f := NewFlatbush[TFloat]()
	f.Finish()
	require.Equal(t, 0, len(f.Search(0, 0, 1, 1)))
}

func TestBasic(t *testing.T) {
	testBasic[float32](t)
	testBasic[float64](t)
}

func testBasic[TFloat float32 | float64](t *testing.T) {
	f := NewFlatbush[TFloat]()
	boxes := []Box[TFloat]{}
	dim := 100
	f.Reserve(int(dim * dim))
	index := 0
	for x := TFloat(0); x < TFloat(dim); x++ {
		for y := TFloat(0); y < TFloat(dim); y++ {
			// boxes.push_back({index, x + 0.1f, y + 0.1f, x + 0.9f, y + 0.9f});
			boxes = append(boxes, Box[TFloat]{
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
	pad := TFloat(3)
	for i := 0; i < nSamples; i++ {
		minx := TFloat(rng.Float32())*TFloat(dim) - pad
		miny := TFloat(rng.Float32())*TFloat(dim) - pad
		maxx := minx + TFloat(rng.Float32())*TFloat(maxQueryWindow)
		maxy := miny + TFloat(rng.Float32())*TFloat(maxQueryWindow)
		results := f.Search(minx, miny, maxx, maxy)
		totalResults += len(results)
		// brute force validation that there are no false negatives
		qbox := Box[TFloat]{
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

func fillSquare[TFloat float32 | float64](f *Flatbush[TFloat], sideLength int) {
	f.Reserve(int(sideLength * sideLength))
	for x := TFloat(0); x < TFloat(sideLength); x++ {
		for y := TFloat(0); y < TFloat(sideLength); y++ {
			f.Add(x+0.1, y+0.1, x+0.9, y+0.9)
		}
	}
	f.Finish()
}

func BenchmarkInsert32(b *testing.B) {
	benchmarkInsert[float32](b)
}

func BenchmarkInsert64(b *testing.B) {
	benchmarkInsert[float64](b)
}

func benchmarkInsert[TFloat float32 | float64](b *testing.B) {
	dim := 1000
	start := time.Now()
	f := NewFlatbush[TFloat]()
	fillSquare(f, dim)
	end := time.Now()
	b.Logf("Time to insert %v elements: %.0f milliseconds", dim*dim, end.Sub(start).Seconds()*1000)
}

func BenchmarkQuery32(b *testing.B) {
	benchmarkQuery[float32](b)
}

func BenchmarkQuery64(b *testing.B) {
	benchmarkQuery[float64](b)
}

func benchmarkQuery[TFloat float32 | float64](b *testing.B) {
	dim := 1000
	f := NewFlatbush[TFloat]()
	fillSquare(f, dim)

	start := time.Now()
	nquery := 10 * 1000 * 1000
	sx := 0
	sy := 0
	results := []int{}
	nresults := 0
	for i := 0; i < nquery; i++ {
		minx := TFloat(sx % dim)
		miny := TFloat(sy % dim)
		maxx := TFloat(minx + 5.0)
		maxy := TFloat(miny + 5.0)
		results = f.SearchFast(minx, miny, maxx, maxy, results)
		nresults += len(results)
		sx++
		sy++
	}
	elapsedS := time.Now().Sub(start).Seconds()
	b.Logf("Time per query, returning average of %.0f elements: %.2f nanoseconds\n", float64(nresults)/float64(nquery), elapsedS*1e9/float64(nquery))
}
