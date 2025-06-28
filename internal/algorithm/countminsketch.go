package algorithm

import (
	"hash/fnv"
	"math"
)

// CountMinSketch implements the Count-Min Sketch algorithm for frequency estimation.
type CountMinSketch struct {
	depth     int
	width     int
	matrix    [][]uint64
	hashFuncs []hashFunc
}

type hashFunc func(data []byte, seed uint32) uint32

// NewCountMinSketch creates a new Count-Min Sketch with the given error rate and confidence.
func NewCountMinSketch(epsilon float64, delta float64) *CountMinSketch {
	// Calculate depth and width based on error rate (epsilon) and confidence (delta)
	depth := int(math.Ceil(math.Log(1 / delta)))
	width := int(math.Ceil(math.E / epsilon))

	// Initialize the matrix
	matrix := make([][]uint64, depth)
	for i := range matrix {
		matrix[i] = make([]uint64, width)
	}

	// Initialize hash functions
	hashFuncs := make([]hashFunc, depth)
	for i := range hashFuncs {
		hashFuncs[i] = func(data []byte, s uint32) uint32 {
			h := fnv.New32a()
			h.Write(data)
			h.Write([]byte{byte(s), byte(s >> 8), byte(s >> 16), byte(s >> 24)})
			return h.Sum32()
		}
	}

	return &CountMinSketch{
		depth:     depth,
		width:     width,
		matrix:    matrix,
		hashFuncs: hashFuncs,
	}
}

// Add adds a value to the sketch.
func (cms *CountMinSketch) Add(key []byte, count uint64) {
	for i := 0; i < cms.depth; i++ {
		j := cms.hashFuncs[i](key, uint32(i)) % uint32(cms.width)
		cms.matrix[i][j] += count
	}
}

// Estimate estimates the frequency of a value.
func (cms *CountMinSketch) Estimate(key []byte) uint64 {
	var min uint64 = math.MaxUint64

	for i := 0; i < cms.depth; i++ {
		j := cms.hashFuncs[i](key, uint32(i)) % uint32(cms.width)
		if cms.matrix[i][j] < min {
			min = cms.matrix[i][j]
		}
	}

	return min
}

// Reset resets the sketch.
func (cms *CountMinSketch) Reset() {
	for i := range cms.matrix {
		for j := range cms.matrix[i] {
			cms.matrix[i][j] = 0
		}
	}
}

// Decay applies exponential decay to all counts
func (cms *CountMinSketch) Decay(factor float64) {
	for i := range cms.matrix {
		for j := range cms.matrix[i] {
			cms.matrix[i][j] = uint64(float64(cms.matrix[i][j]) * factor)
		}
	}
}
