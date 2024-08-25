package flatbush

import "math"

// Define a type constraint for all numeric types we're interested in
type Coord interface {
	int8 | int16 | int32 | int64 | float32 | float64
}

// This method of using structs was a workaround for a bug in the Go 1.22 compiler/linker.
type MinMaxInt8 struct {
	Min int8
	Max int8
}

type MinMaxInt16 struct {
	Min int16
	Max int16
}

type MinMaxInt32 struct {
	Min int32
	Max int32
}

type MinMaxInt64 struct {
	Min int64
	Max int64
}

type MinMaxFloat32 struct {
	Min float32
	Max float32
}

type MinMaxFloat64 struct {
	Min float64
	Max float64
}

// MinMaxValueOfType returns the minimum and maximum value for the given numeric type T.
func MinMaxValueOfType[T Coord](min, max *T) {
	// Use type assertion to determine the type of T
	switch any(*min).(type) {
	case int8:
		mm := MinMaxInt8{Min: math.MinInt8, Max: math.MaxInt8}
		*min = any(mm.Min).(T)
		*max = any(mm.Max).(T)
	case int16:
		mm := MinMaxInt16{Min: math.MinInt16, Max: math.MaxInt16}
		*min = any(mm.Min).(T)
		*max = any(mm.Max).(T)
	case int32:
		mm := MinMaxInt32{Min: math.MinInt32, Max: math.MaxInt32}
		*min = any(mm.Min).(T)
		*max = any(mm.Max).(T)
	case int64:
		mm := MinMaxInt64{Min: math.MinInt64, Max: math.MaxInt64}
		*min = any(mm.Min).(T)
		*max = any(mm.Max).(T)
	case float32:
		mm := MinMaxFloat32{Min: -math.MaxFloat32, Max: math.MaxFloat32}
		*min = any(mm.Min).(T)
		*max = any(mm.Max).(T)
	case float64:
		mm := MinMaxFloat64{Min: -math.MaxFloat64, Max: math.MaxFloat64}
		*min = any(mm.Min).(T)
		*max = any(mm.Max).(T)
	default:
		// This panic should never be reached due to the Numeric constraint
		panic("Unsupported type")
	}
}

type Box[T Coord] struct {
	MinX  T
	MinY  T
	MaxX  T
	MaxY  T
	Index int
}

func InvertedBox[T Coord]() Box[T] {
	var min, max T
	MinMaxValueOfType[T](&min, &max)
	return Box[T]{MinX: max, MinY: max, MaxX: min, MaxY: min, Index: -1}
}

func (a *Box[T]) PositiveUnion(b *Box[T]) bool {
	return b.MaxX >= a.MinX && b.MinX <= a.MaxX && b.MaxY >= a.MinY && b.MinY <= a.MaxY
}

func hilbertXYToIndex(n uint32, x uint32, y uint32) uint32 {
	x = x << (16 - n)
	y = y << (16 - n)

	var A, B, C, D uint32

	// Initial prefix scan round, prime with x and y
	{
		a := uint32(x ^ y)
		b := uint32(0xFFFF ^ a)
		c := uint32(0xFFFF ^ (x | y))
		d := uint32(x & (y ^ 0xFFFF))

		A = a | (b >> 1)
		B = (a >> 1) ^ a

		C = ((c >> 1) ^ (b & (d >> 1))) ^ c
		D = ((a & (c >> 1)) ^ (d >> 1)) ^ d
	}

	{
		a := A
		b := B
		c := C
		d := D

		A = ((a & (a >> 2)) ^ (b & (b >> 2)))
		B = ((a & (b >> 2)) ^ (b & ((a ^ b) >> 2)))

		C ^= ((a & (c >> 2)) ^ (b & (d >> 2)))
		D ^= ((b & (c >> 2)) ^ ((a ^ b) & (d >> 2)))
	}

	{
		a := A
		b := B
		c := C
		d := D

		A = ((a & (a >> 4)) ^ (b & (b >> 4)))
		B = ((a & (b >> 4)) ^ (b & ((a ^ b) >> 4)))

		C ^= ((a & (c >> 4)) ^ (b & (d >> 4)))
		D ^= ((b & (c >> 4)) ^ ((a ^ b) & (d >> 4)))
	}

	// Final round and projection
	{
		a := A
		b := B
		c := C
		d := D

		C ^= ((a & (c >> 8)) ^ (b & (d >> 8)))
		D ^= ((b & (c >> 8)) ^ ((a ^ b) & (d >> 8)))
	}

	// Undo transformation prefix scan
	a := uint32(C ^ (C >> 1))
	b := uint32(D ^ (D >> 1))

	// Recover index bits
	i0 := uint32(x ^ y)
	i1 := uint32(b | (0xFFFF ^ (i0 | a)))

	return ((interleave(i1) << 1) | interleave(i0)) >> (32 - 2*n)
}

// From https://github.com/rawrunprotected/hilbert_curves (public domain)
func interleave(x uint32) uint32 {
	x = (x | (x << 8)) & 0x00FF00FF
	x = (x | (x << 4)) & 0x0F0F0F0F
	x = (x | (x << 2)) & 0x33333333
	x = (x | (x << 1)) & 0x55555555
	return x
}

// custom quicksort that sorts bbox data alongside the hilbert values
func sortValuesAndBoxes[TBox any](values []uint32, boxes []TBox, left, right int) {
	if left >= right {
		return
	}

	pivot := values[(left+right)>>1]
	i := left - 1
	j := right + 1

	for {
		i++
		for values[i] < pivot {
			i++
		}
		j--
		for values[j] > pivot {
			j--
		}
		if i >= j {
			break
		}
		values[i], values[j] = values[j], values[i]
		boxes[i], boxes[j] = boxes[j], boxes[i]
	}

	sortValuesAndBoxes(values, boxes, left, j)
	sortValuesAndBoxes(values, boxes, j+1, right)
}

// Finish builds the spatial index, so that it can be queried.
func finishIndexBuild[T Coord](nodeSize int, boxes []Box[T], bounds Box[T]) (int, []int, []uint32, []Box[T]) {
	if nodeSize < 2 {
		nodeSize = 2
	}

	numItems := len(boxes)

	// calculate the total number of nodes in the R-tree to allocate space for
	// and the index of each tree level (used in search later)
	n := numItems
	numNodes := n
	levelBounds := []int{n}
	for {
		n = (n + nodeSize - 1) / nodeSize
		numNodes += n
		levelBounds = append(levelBounds, numNodes)
		if n <= 1 {
			break
		}
	}

	width := float64(bounds.MaxX - bounds.MinX)
	height := float64(bounds.MaxY - bounds.MinY)

	hilbertValues := make([]uint32, len(boxes))
	hilbertMax := float64((1 << 16) - 1)

	// map item centers into Hilbert coordinate space and calculate Hilbert values
	for i := 0; i < len(boxes); i++ {
		b := boxes[i]
		x := uint32(hilbertMax * (((float64(b.MinX)+float64(b.MaxX))/2 - float64(bounds.MinX)) / width))
		y := uint32(hilbertMax * (((float64(b.MinY)+float64(b.MaxY))/2 - float64(bounds.MinY)) / height))
		hilbertValues[i] = hilbertXYToIndex(16, x, y)
	}

	// sort items by their Hilbert value (for packing later)
	if len(boxes) != 0 {
		sortValuesAndBoxes(hilbertValues, boxes, 0, len(boxes)-1)
	}

	// generate nodes at each tree level, bottom-up
	pos := 0
	for i := 0; i < len(levelBounds)-1; i++ {
		end := levelBounds[i]

		// generate a parent node for each block of consecutive <nodeSize> nodes
		for pos < end {
			nodeBox := InvertedBox[T]()
			nodeBox.Index = pos

			// calculate bbox for the new node
			for j := 0; j < nodeSize && pos < end; j++ {
				box := boxes[pos]
				pos++
				nodeBox.MinX = min(nodeBox.MinX, box.MinX)
				nodeBox.MinY = min(nodeBox.MinY, box.MinY)
				nodeBox.MaxX = max(nodeBox.MaxX, box.MaxX)
				nodeBox.MaxY = max(nodeBox.MaxY, box.MaxY)
			}

			// add the new node to the tree data
			boxes = append(boxes, nodeBox)
		}
	}

	return nodeSize, levelBounds, hilbertValues, boxes
}

// searchInTree accepts a 'results' as input. If you are performing millions of queries,
// then reusing a 'results' slice will reduce the number of allocations.
func searchInTree[T Coord](nodeSize, numItems int, levelBounds []int, boxes []Box[T], minX, minY, maxX, maxY T, results []int) []int {
	results = results[:0]
	if len(levelBounds) == 0 {
		// Must call Finish()
		return results
	}
	if len(boxes) == 0 {
		// Empty tree
		return results
	}

	queue := make([]int, 0, 32)
	queue = append(queue, len(boxes)-1)       // nodeIndex
	queue = append(queue, len(levelBounds)-1) // level

	for len(queue) != 0 {
		nodeIndex := queue[len(queue)-2]
		level := queue[len(queue)-1]
		queue = queue[:len(queue)-2]

		// find the end index of the node
		end := min(nodeIndex+nodeSize, levelBounds[level])

		// search through child nodes
		for pos := nodeIndex; pos < end; pos++ {
			// check if node bbox intersects with query bbox
			if maxX < boxes[pos].MinX ||
				maxY < boxes[pos].MinY ||
				minX > boxes[pos].MaxX ||
				minY > boxes[pos].MaxY {
				continue
			}
			if nodeIndex < numItems {
				// leaf item
				results = append(results, boxes[pos].Index)
			} else {
				// node; add it to the search queue
				queue = append(queue, boxes[pos].Index)
				queue = append(queue, level-1)
			}
		}
	}
	return results
}
