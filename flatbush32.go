package flatbush

// Package flatbush is a port of https://github.com/mourner/flatbush

// Flatbush32 is a spatial index for efficient 2D queries.
// The coordinates are 32-bit floats.
type Flatbush32 struct {
	NodeSize int // Minimum 2. Default 16

	boxes         []Box[float32]
	bounds        Box[float32]
	hilbertValues []uint32
	levelBounds   []int
	numItems      int
}

// Create a new float32 Flatbush
func NewFlatbush32() *Flatbush32 {
	return &Flatbush32{
		NodeSize: 16,
		bounds:   InvertedBox[float32](),
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
	f.boxes = make([]Box[float32], 0, numNodes)
}

// Add a new box, and return its index.
// The index of the box is zero based, and corresponds 1:1 with the insertion of order of the boxes.
// You must add all boxes before calling Finish().
func (f *Flatbush32) Add(minX, minY, maxX, maxY float32) int {
	index := len(f.boxes)
	f.boxes = append(f.boxes, Box[float32]{
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
	f.numItems = len(f.boxes)
	f.NodeSize, f.levelBounds, f.hilbertValues, f.boxes = finishIndexBuild(f.NodeSize, f.boxes, f.bounds)
}

// Search for all boxes that overlap the given query box.
func (f *Flatbush32) Search(minX, minY, maxX, maxY float32) []int {
	results := []int{}
	return f.SearchFast(minX, minY, maxX, maxY, results)
}

// SearchFast accepts a 'results' as input. If you are performing millions of queries,
// then reusing a 'results' slice will reduce the number of allocations.
func (f *Flatbush32) SearchFast(minX, minY, maxX, maxY float32, results []int) []int {
	return searchInTree(f.NodeSize, f.numItems, f.levelBounds, f.boxes, minX, minY, maxX, maxY, results)
}
