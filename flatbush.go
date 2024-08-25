package flatbush

// Package flatbush is a port of https://github.com/mourner/flatbush

// Flatbush is a spatial index for efficient 2D queries.
// The coordinates are one of *Coord*
type Flatbush[T Coord] struct {
	NodeSize int // Minimum 2. Default 16

	boxes         []Box[T]
	bounds        Box[T]
	hilbertValues []uint32
	levelBounds   []int
	numItems      int
}

// Create a new Flatbush with one *Coord* type
func NewFlatbush[T Coord]() *Flatbush[T] {
	return &Flatbush[T]{
		NodeSize: 16,
		bounds:   InvertedBox[T](),
	}
}

// Reserve enough boxes for the given number of items
func (f *Flatbush[T]) Reserve(size int) {
	n := size
	numNodes := n
	for n > 1 {
		n = (n + f.NodeSize - 1) / f.NodeSize
		numNodes += n
	}
	f.boxes = make([]Box[T], 0, numNodes)
}

// Add a new box, and return its index.
// The index of the box is zero based, and corresponds 1:1 with the insertion of order of the boxes.
// You must add all boxes before calling Finish().
func (f *Flatbush[T]) Add(minX, minY, maxX, maxY T) int {
	index := len(f.boxes)
	f.boxes = append(f.boxes, Box[T]{
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
func (f *Flatbush[T]) Finish() {
	f.numItems = len(f.boxes)
	f.NodeSize, f.levelBounds, f.hilbertValues, f.boxes = finishIndexBuild(f.NodeSize, f.boxes, f.bounds)
}

// Search for all boxes that overlap the given query box.
func (f *Flatbush[T]) Search(minX, minY, maxX, maxY T) []int {
	results := []int{}
	return f.SearchFast(minX, minY, maxX, maxY, results)
}

// SearchFast accepts a 'results' as input. If you are performing millions of queries,
// then reusing a 'results' slice will reduce the number of allocations.
func (f *Flatbush[T]) SearchFast(minX, minY, maxX, maxY T, results []int) []int {
	return searchInTree(f.NodeSize, f.numItems, f.levelBounds, f.boxes, minX, minY, maxX, maxY, results)
}
