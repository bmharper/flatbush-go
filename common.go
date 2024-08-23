package flatbush

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
func sortValuesAndBoxes[TBox Box32 | Box64](values []uint32, boxes []TBox, left, right int) {
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

/*
// searchInTree accepts a 'results' as input. If you are performing millions of queries,
// then reusing a 'results' slice will reduce the number of allocations.
func searchInTree[TFloat float32 | float64, TBox Box32 | Box64](levelBounds []int, boxes []TBox, minX, minY, maxX, maxY TFloat, results []int) []int {
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
		end := min(nodeIndex+f.NodeSize, levelBounds[level])

		// search through child nodes
		for pos := nodeIndex; pos < end; pos++ {
			// check if node bbox intersects with query bbox
			if maxX < boxes[pos].MinX ||
				maxY < boxes[pos].MinY ||
				minX > boxes[pos].MaxX ||
				minY > boxes[pos].MaxY {
				continue
			}
			if nodeIndex < f.numItems {
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
*/
