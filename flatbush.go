package flatbush

// Package flatbush is a port of https://github.com/mourner/flatbush

import "math"

// Flatbush64 is a spatial index for efficient 2D queries.
// The coordinates are 64-bit floats.
type Flatbush64 struct {
	NodeSize int // Minimum 2. Default 16

	boxes         []Box64
	bounds        Box64
	hilbertValues []uint32
	levelBounds   []int
	numItems      int
}

func NewFlatbush() *Flatbush64 {
	return &Flatbush64{
		NodeSize: 16,
		bounds:   InvertedBox(),
	}
}

// Reserve enough boxes for the given number of items
func (f *Flatbush64) Reserve(size int) {
	n := size
	numNodes := n
	for n > 1 {
		n = (n + f.NodeSize - 1) / f.NodeSize
		numNodes += n
	}
	f.boxes = make([]Box64, 0, numNodes)
}

// Add a new box, and return its index.
// The index of the box is zero based, and corresponds 1:1 with the insertion of order of the boxes.
// You must add all boxes before calling Finish().
func (f *Flatbush64) Add(minX, minY, maxX, maxY float64) int {
	index := len(f.boxes)
	f.boxes = append(f.boxes, Box64{
		MinX:  minX,
		MinY:  minY,
		MaxX:  maxX,
		MaxY:  maxY,
		Index: index,
	})
	f.bounds.MinX = math.Min(f.bounds.MinX, minX)
	f.bounds.MinY = math.Min(f.bounds.MinY, minY)
	f.bounds.MaxX = math.Max(f.bounds.MaxX, maxX)
	f.bounds.MaxY = math.Max(f.bounds.MaxY, maxY)
	return index
}

// Finish builds the spatial index, so that it can be queried.
func (f *Flatbush64) Finish() {
	if f.NodeSize < 2 {
		f.NodeSize = 2
	}

	f.numItems = len(f.boxes)

	// calculate the total number of nodes in the R-tree to allocate space for
	// and the index of each tree level (used in search later)
	n := f.numItems
	numNodes := n
	f.levelBounds = append(f.levelBounds, n)
	for {
		n = (n + f.NodeSize - 1) / f.NodeSize
		numNodes += n
		f.levelBounds = append(f.levelBounds, numNodes)
		if n <= 1 {
			break
		}
	}

	width := f.bounds.MaxX - f.bounds.MinX
	height := f.bounds.MaxY - f.bounds.MinY

	f.hilbertValues = make([]uint32, len(f.boxes))
	hilbertMax := float64((1 << 16) - 1)

	// map item centers into Hilbert coordinate space and calculate Hilbert values
	for i := 0; i < len(f.boxes); i++ {
		b := f.boxes[i]
		x := uint32(hilbertMax * ((b.MinX+b.MaxX)/2 - f.bounds.MinX) / width)
		y := uint32(hilbertMax * ((b.MinY+b.MaxY)/2 - f.bounds.MinY) / height)
		f.hilbertValues[i] = hilbertXYToIndex(16, x, y)
	}

	// sort items by their Hilbert value (for packing later)
	if len(f.boxes) != 0 {
		sort(f.hilbertValues, f.boxes, 0, len(f.boxes)-1)
	}

	// generate nodes at each tree level, bottom-up
	pos := 0
	for i := 0; i < len(f.levelBounds)-1; i++ {
		end := f.levelBounds[i]

		// generate a parent node for each block of consecutive <nodeSize> nodes
		for pos < end {
			nodeBox := InvertedBox()
			nodeBox.Index = pos

			// calculate bbox for the new node
			for j := 0; j < f.NodeSize && pos < end; j++ {
				box := f.boxes[pos]
				pos++
				nodeBox.MinX = math.Min(nodeBox.MinX, box.MinX)
				nodeBox.MinY = math.Min(nodeBox.MinY, box.MinY)
				nodeBox.MaxX = math.Max(nodeBox.MaxX, box.MaxX)
				nodeBox.MaxY = math.Max(nodeBox.MaxY, box.MaxY)
			}

			// add the new node to the tree data
			f.boxes = append(f.boxes, nodeBox)
		}
	}
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Search for all boxes that overlap the given query box.
func (f *Flatbush64) Search(minX, minY, maxX, maxY float64) []int {
	results := []int{}
	return f.SearchFast(minX, minY, maxX, maxY, results)
}

// SearchFast accepts a 'results' as input. If you are performing millions of queries,
// then reusing a 'results' slice will reduce the number of allocations.
func (f *Flatbush64) SearchFast(minX, minY, maxX, maxY float64, results []int) []int {
	results = results[:0]
	if len(f.levelBounds) == 0 {
		// Must call Finish()
		return results
	}
	if len(f.boxes) == 0 {
		// Empty tree
		return results
	}

	queue := make([]int, 0, 32)
	queue = append(queue, len(f.boxes)-1)       // nodeIndex
	queue = append(queue, len(f.levelBounds)-1) // level

	for len(queue) != 0 {
		nodeIndex := queue[len(queue)-2]
		level := queue[len(queue)-1]
		queue = queue[:len(queue)-2]

		// find the end index of the node
		end := intMin(nodeIndex+f.NodeSize, f.levelBounds[level])

		// search through child nodes
		for pos := nodeIndex; pos < end; pos++ {
			// check if node bbox intersects with query bbox
			if maxX < f.boxes[pos].MinX ||
				maxY < f.boxes[pos].MinY ||
				minX > f.boxes[pos].MaxX ||
				minY > f.boxes[pos].MaxY {
				continue
			}
			if nodeIndex < f.numItems {
				// leaf item
				results = append(results, f.boxes[pos].Index)
			} else {
				// node; add it to the search queue
				queue = append(queue, f.boxes[pos].Index)
				queue = append(queue, level-1)
			}
		}
	}
	return results
}

// custom quicksort that sorts bbox data alongside the hilbert values
func sort(values []uint32, boxes []Box64, left, right int) {
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

	sort(values, boxes, left, j)
	sort(values, boxes, j+1, right)
}

type Box64 struct {
	MinX  float64
	MinY  float64
	MaxX  float64
	MaxY  float64
	Index int
}

func InvertedBox() Box64 {
	return Box64{
		MinX:  math.MaxFloat64,
		MinY:  math.MaxFloat64,
		MaxX:  -math.MaxFloat64,
		MaxY:  -math.MaxFloat64,
		Index: -1,
	}
}

func (a *Box64) PositiveUnion(b *Box64) bool {
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
