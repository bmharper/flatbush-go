package flatbush

// Package flatbush is a port of https://github.com/mourner/flatbush

// Flatbush is a spatial index for efficient 2D queries.
// The coordinates are either 32-bit or 64-bit floats.
type Flatbush[TFloat float32 | float64] struct {
	NodeSize int // Minimum 2. Default 16

	boxes         []Box[TFloat]
	bounds        Box[TFloat]
	hilbertValues []uint32
	levelBounds   []int
	numItems      int
}

// Create a new Flatbush with either 32-bit or 64-bit floats.
func NewFlatbush[TFloat float32 | float64]() *Flatbush[TFloat] {
	return &Flatbush[TFloat]{
		NodeSize: 16,
		bounds:   InvertedBox[TFloat](),
	}
}

// Reserve enough boxes for the given number of items
func (f *Flatbush[TFloat]) Reserve(size int) {
	n := size
	numNodes := n
	for n > 1 {
		n = (n + f.NodeSize - 1) / f.NodeSize
		numNodes += n
	}
	f.boxes = make([]Box[TFloat], 0, numNodes)
}

// Add a new box, and return its index.
// The index of the box is zero based, and corresponds 1:1 with the insertion of order of the boxes.
// You must add all boxes before calling Finish().
func (f *Flatbush[TFloat]) Add(minX, minY, maxX, maxY TFloat) int {
	index := len(f.boxes)
	f.boxes = append(f.boxes, Box[TFloat]{
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
func (f *Flatbush[TFloat]) Finish() {
	f.numItems = len(f.boxes)
	f.NodeSize, f.levelBounds, f.hilbertValues, f.boxes = finishIndexBuild(f.NodeSize, f.boxes, f.bounds)
}

// Search for all boxes that overlap the given query box.
func (f *Flatbush[TFloat]) Search(minX, minY, maxX, maxY TFloat) []int {
	results := []int{}
	return f.SearchFast(minX, minY, maxX, maxY, results)
}

// SearchFast accepts a 'results' as input. If you are performing millions of queries,
// then reusing a 'results' slice will reduce the number of allocations.
func (f *Flatbush[TFloat]) SearchFast(minX, minY, maxX, maxY TFloat, results []int) []int {
	return searchInTree(f.NodeSize, f.numItems, f.levelBounds, f.boxes, minX, minY, maxX, maxY, results)
}
