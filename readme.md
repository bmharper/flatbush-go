# Flatbush (Go port)

This is a Go port of https://github.com/mourner/flatbush.

# Usage

```go
// Create a new flatbush spatial index (must be either float32 or float64)
f := flatbush.NewFlatbush[float64]()

// Populate the tree
for _, b := range boxes {
	f.Add(b.MinX, b.MinY, b.MaxX, b.MaxY)
}

// Finish creation of the spatial index
f.Finish()

// Find all boxes that overlap the given bounding box
results := f.Search(minX, minY, maxX, maxY)

// Results is an []int, containing zero-based indices of the objects in the tree,
// according to their insertion order.
```

# Times

Measurements taken on Intel Core i7-11850H @ 2.50GHz

- Time to insert 1000000 elements: 91 milliseconds
- Time per query, returning average of 25 elements: 400 nanoseconds
