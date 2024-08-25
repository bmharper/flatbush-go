package flatbush

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGenericSpecialization(t *testing.T) {
	{
		a, b := MinMaxValueOfType[int8]()
		require.Equal(t, int8(math.MinInt8), a)
		require.Equal(t, int8(math.MaxInt8), b)
	}
	{
		a, b := MinMaxValueOfType[int16]()
		require.Equal(t, int16(math.MinInt16), a)
		require.Equal(t, int16(math.MaxInt16), b)
	}
	{
		a, b := MinMaxValueOfType[int32]()
		require.Equal(t, int32(math.MinInt32), a)
		require.Equal(t, int32(math.MaxInt32), b)
	}
	{
		a, b := MinMaxValueOfType[int64]()
		require.Equal(t, int64(math.MinInt64), a)
		require.Equal(t, int64(math.MaxInt64), b)
	}
	{
		a, b := MinMaxValueOfType[float32]()
		require.Equal(t, -float32(math.MaxFloat32), a)
		require.Equal(t, float32(math.MaxFloat32), b)
	}
	{
		a, b := MinMaxValueOfType[float64]()
		require.Equal(t, -float64(math.MaxFloat64), a)
		require.Equal(t, float64(math.MaxFloat64), b)
	}
}

func TestEmpty(t *testing.T) {
	testEmpty[float32](t)
	testEmpty[float64](t)
}

func testEmpty[T Coord](t *testing.T) {
	f := NewFlatbush[T]()
	f.Finish()
	require.Equal(t, 0, len(f.Search(0, 0, 1, 1)))
}

func TestBasic(t *testing.T) {
	testBasic[int8](t, 10) // to avoid int8 overflow, we need a much smaller test grid
	testBasic[int16](t, 100)
	testBasic[int32](t, 100)
	testBasic[int64](t, 100)
	testBasic[float32](t, 100)
	testBasic[float64](t, 100)
}

func testBasic[T Coord](t *testing.T, dim int) {
	f := NewFlatbush[T]()
	boxes := []Box[T]{}
	// We create a square of dim * dim objects, which are 10 units apart
	f.Reserve(int(dim * dim))
	index := 0
	for x := T(0); x < T(dim); x++ {
		for y := T(0); y < T(dim); y++ {
			boxes = append(boxes, Box[T]{
				MinX:  x + 1,
				MinY:  y + 1,
				MaxX:  x + 9,
				MaxY:  y + 9,
				Index: index,
			})
			checkIndex := f.Add(x+1, y+1, x+9, y+9)
			require.Equal(t, index, checkIndex)
			index++
		}
	}
	f.Finish()

	rng := rand.New(rand.NewSource(0))

	totalResults := 0
	nSamples := 1000
	maxQueryWindow := 5
	pad := T(3)
	for i := 0; i < nSamples; i++ {
		minx := T(rng.Float32()*10*float32(dim)) - pad
		miny := T(rng.Float32()*10*float32(dim)) - pad
		maxx := minx + T(rng.Float32()*10*float32(maxQueryWindow))
		maxy := miny + T(rng.Float32()*10*float32(maxQueryWindow))
		results := f.Search(minx, miny, maxx, maxy)
		totalResults += len(results)
		// brute force validation that there are no false negatives
		qbox := Box[T]{
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

func fillSquare[T Coord](f *Flatbush[T], sideDim, spacing int) {
	f.Reserve(int(sideDim * sideDim))
	for x := T(0); x < T(sideDim); x++ {
		for y := T(0); y < T(sideDim); y++ {
			xp := x * T(spacing)
			yp := y * T(spacing)
			f.Add(xp+1, yp+1, xp+T(spacing)-1, yp+T(spacing)-1)
		}
	}
	f.Finish()
}

func BenchmarkInsertInt16(b *testing.B) {
	benchmarkInsert[int16](b)
}

func BenchmarkInsertFloat32(b *testing.B) {
	benchmarkInsert[float32](b)
}

func BenchmarkInsertFloat64(b *testing.B) {
	benchmarkInsert[float64](b)
}

func benchmarkInsert[T Coord](b *testing.B) {
	dim := 1000
	start := time.Now()
	f := NewFlatbush[T]()
	fillSquare(f, dim, 10)
	end := time.Now()
	b.Logf("Time to insert %v elements: %.0f milliseconds", dim*dim, end.Sub(start).Seconds()*1000)
}

func BenchmarkQueryInt16(b *testing.B) {
	benchmarkQuery[int16](b)
}

func BenchmarkQueryFloat32(b *testing.B) {
	benchmarkQuery[float32](b)
}

func BenchmarkQueryFloat64(b *testing.B) {
	benchmarkQuery[float64](b)
}

func benchmarkQuery[T Coord](b *testing.B) {
	dim := 1000
	f := NewFlatbush[T]()
	fillSquare(f, dim, 10)

	start := time.Now()
	nquery := 10 * 1000 * 1000
	sx := 0
	sy := 0
	results := []int{}
	nresults := 0
	for i := 0; i < nquery; i++ {
		minx := T(sx % (dim * 10))
		miny := T(sy % (dim * 10))
		maxx := T(minx + 4*10)
		maxy := T(miny + 4*10)
		results = f.SearchFast(minx, miny, maxx, maxy, results)
		nresults += len(results)
		sx += 13
		sy += 17
	}
	elapsedS := time.Now().Sub(start).Seconds()
	b.Logf("Time per query, returning average of %.0f elements: %.2f nanoseconds\n", float64(nresults)/float64(nquery), elapsedS*1e9/float64(nquery))
}
