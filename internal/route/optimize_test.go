package route

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// pathLength is the length of the path through points in the given order,
// from start when it is given.
func pathLength(points []Point, start *Point, idx []int) float64 {
	total := 0.0
	prev := start
	for _, i := range idx {
		if prev != nil {
			total += distance(*prev, points[i])
		}
		prev = &points[i]
	}
	return total
}

func TestDistance(t *testing.T) {
	// Praça da Sé to Avenida Paulista (MASP), about 2.6 km in a straight line.
	se, masp := Point{-23.5503, -46.6339}, Point{-23.5614, -46.6559}
	assert.InDelta(t, 2.6, distance(se, masp), 0.3)
	assert.Zero(t, distance(se, se))
}

func TestOrder_Line(t *testing.T) {
	// Five stops on a line, given out of order.
	points := []Point{{0, 0.03}, {0, 0.01}, {0, 0.04}, {0, 0}, {0, 0.02}}

	got := order(points, nil)
	assert.Contains(t, [][]int{{3, 1, 4, 0, 2}, {2, 0, 4, 1, 3}}, got, "walks the line from one end")

	got = order(points, &Point{0, 0.05})
	assert.Equal(t, []int{2, 0, 4, 1, 3}, got, "starts from the end nearest the driver")
}

func TestOrder_TwoOptRemovesCrossing(t *testing.T) {
	// Nearest neighbor from the start goes A, B and then has to come back
	// across; 2-opt must find a path no longer than the best one.
	start := &Point{0, 0}
	points := []Point{{0, 0.010}, {0, 0.011}, {0.02, 0.0105}, {-0.001, 0.03}}
	got := order(points, start)
	assert.ElementsMatch(t, []int{0, 1, 2, 3}, got)

	best := pathLength(points, start, got)
	for _, p := range permutations(4) {
		assert.LessOrEqual(t, best, pathLength(points, start, p)*1.2, "close to the best order %v", p)
	}
}

func TestOrder_Empty(t *testing.T) {
	assert.Empty(t, order(nil, nil))
	assert.Equal(t, []int{0}, order([]Point{{1, 1}}, &Point{0, 0}))
}

func permutations(n int) [][]int {
	if n == 1 {
		return [][]int{{0}}
	}
	var out [][]int
	for _, p := range permutations(n - 1) {
		for i := 0; i <= len(p); i++ {
			q := append(append(append([]int{}, p[:i]...), n-1), p[i:]...)
			out = append(out, q)
		}
	}
	return out
}

func TestOptimizeInput(t *testing.T) {
	lat, lng := -23.5, -46.6
	start, err := optimizeInput{Latitude: &lat, Longitude: &lng}.start()
	assert.NoError(t, err)
	assert.Equal(t, &Point{lat, lng}, start)

	start, err = optimizeInput{}.start()
	assert.NoError(t, err)
	assert.Nil(t, start)

	_, err = optimizeInput{Latitude: &lat}.start()
	assert.Error(t, err)
	bad := 200.0
	_, err = optimizeInput{Latitude: &lat, Longitude: &bad}.start()
	assert.Error(t, err)
}
