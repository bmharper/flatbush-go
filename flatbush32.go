package flatbush

// Package flatbush is a port of https://github.com/mourner/flatbush

import "math"

// Flatbush32 is a spatial index for efficient 2D queries.
// The coordinates are 32-bit floats.
type Flatbush32 struct {
	NodeSize int // Minimum 2. Default 16

	boxes         []Box32
	bounds        Box32
	hilbertValues []uint32
	levelBounds   []int
	numItems      int
}

func NewFlatbush32() *Flatbush32 {
	return &Flatbush32{
		NodeSize: 16,
		bounds:   InvertedBox32(),
	}
}

// Reserve enough boxes for the given number of items
func (f *Flatbush32) Reserve(size int) {
	n := size
	numNodes := n
	for n > 1 {
		n = (n + f.NodeSize - 1) / f.NodeSize
		numNodes += n
	}
	f.boxes = make([]Box32, 0, numNodes)
}

// Add a new box, and return its index.
// The index of the box is zero based, and corresponds 1:1 with the insertion of order of the boxes.
// You must add all boxes before calling Finish().
func (f *Flatbush32) Add(minX, minY, maxX, maxY float32) int {
	index := len(f.boxes)
	f.boxes = append(f.boxes, Box32{
		MinX:  minX,
		MinY:  minY,
		MaxX:  maxX,
		MaxY:  maxY,
		Index: index,
	})
	f.bounds.MinX = min(f.bounds.MinX, minX)
	f.bounds.MinY = min(f.bounds.MinY, minY)
	f.bounds.MaxX = max(f.bounds.MaxX, maxX)
	f.bounds.MaxY = max(f.bounds.MaxY, maxY)
	return index
}

// Finish builds the spatial index, so that it can be queried.
func (f *Flatbush32) Finish() {
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
	hilbertMax := float32((1 << 16) - 1)

	// map item centers into Hilbert coordinate space and calculate Hilbert values
	for i := 0; i < len(f.boxes); i++ {
		b := f.boxes[i]
		x := uint32(hilbertMax * ((b.MinX+b.MaxX)/2 - f.bounds.MinX) / width)
		y := uint32(hilbertMax * ((b.MinY+b.MaxY)/2 - f.bounds.MinY) / height)
		f.hilbertValues[i] = hilbertXYToIndex(16, x, y)
	}

	// sort items by their Hilbert value (for packing later)
	if len(f.boxes) != 0 {
		sortValuesAndBoxes(f.hilbertValues, f.boxes, 0, len(f.boxes)-1)
	}

	// generate nodes at each tree level, bottom-up
	pos := 0
	for i := 0; i < len(f.levelBounds)-1; i++ {
		end := f.levelBounds[i]

		// generate a parent node for each block of consecutive <nodeSize> nodes
		for pos < end {
			nodeBox := InvertedBox32()
			nodeBox.Index = pos

			// calculate bbox for the new node
			for j := 0; j < f.NodeSize && pos < end; j++ {
				box := f.boxes[pos]
				pos++
				nodeBox.MinX = min(nodeBox.MinX, box.MinX)
				nodeBox.MinY = min(nodeBox.MinY, box.MinY)
				nodeBox.MaxX = max(nodeBox.MaxX, box.MaxX)
				nodeBox.MaxY = max(nodeBox.MaxY, box.MaxY)
			}

			// add the new node to the tree data
			f.boxes = append(f.boxes, nodeBox)
		}
	}
}

// Search for all boxes that overlap the given query box.
func (f *Flatbush32) Search(minX, minY, maxX, maxY float32) []int {
	results := []int{}
	return f.SearchFast(minX, minY, maxX, maxY, results)
}

// SearchFast accepts a 'results' as input. If you are performing millions of queries,
// then reusing a 'results' slice will reduce the number of allocations.
func (f *Flatbush32) SearchFast(minX, minY, maxX, maxY float32, results []int) []int {
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
		end := min(nodeIndex+f.NodeSize, f.levelBounds[level])

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
func sort32(values []uint32, boxes []Box32, left, right int) {
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

	sort32(values, boxes, left, j)
	sort32(values, boxes, j+1, right)
}

type Box32 struct {
	MinX  float32
	MinY  float32
	MaxX  float32
	MaxY  float32
	Index int
}

func InvertedBox32() Box32 {
	return Box32{
		MinX:  math.MaxFloat32,
		MinY:  math.MaxFloat32,
		MaxX:  -math.MaxFloat32,
		MaxY:  -math.MaxFloat32,
		Index: -1,
	}
}

func (a *Box32) PositiveUnion(b *Box32) bool {
	return b.MaxX >= a.MinX && b.MinX <= a.MaxX && b.MaxY >= a.MinY && b.MinY <= a.MaxY
}
