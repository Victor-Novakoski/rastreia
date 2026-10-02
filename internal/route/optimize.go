package route

import (
	"math"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
)

// Point is a position on the map, in degrees.
type Point struct {
	Lat float64 `json:"latitude"`
	Lng float64 `json:"longitude"`
}

// distance is the great-circle distance between two points in km. Straight
// lines are enough to order stops that are a few km apart; real streets
// would need a routing service.
func distance(a, b Point) float64 {
	const earthRadius = 6371.0
	rad := math.Pi / 180
	dLat := (b.Lat - a.Lat) * rad
	dLng := (b.Lng - a.Lng) * rad
	h := math.Pow(math.Sin(dLat/2), 2) + math.Cos(a.Lat*rad)*math.Cos(b.Lat*rad)*math.Pow(math.Sin(dLng/2), 2)
	return 2 * earthRadius * math.Asin(math.Min(1, math.Sqrt(h)))
}

// order returns the indexes of points in a short path visiting all of
// them, starting near start when it is given. It builds the path by always
// going to the nearest point left, then improves it with 2-opt: reversing
// any stretch that makes the path shorter, until none does. The path is open
// (the driver does not come back), which 2-opt handles by giving the missing
// edges at the ends no cost.
func order(points []Point, start *Point) []int {
	n := len(points)
	if n == 0 {
		return nil
	}
	path := nearestNeighbor(points, start)

	// at returns the point at position i of the path; -1 is the start.
	at := func(i int) (Point, bool) {
		switch {
		case i == -1 && start != nil:
			return *start, true
		case i < 0 || i >= n:
			return Point{}, false
		default:
			return points[path[i]], true
		}
	}
	edge := func(i, j int) float64 {
		a, okA := at(i)
		b, okB := at(j)
		if !okA || !okB {
			return 0
		}
		return distance(a, b)
	}

	for improved, rounds := true, 0; improved && rounds < 100; rounds++ {
		improved = false
		for i := 0; i < n-1; i++ {
			for k := i + 1; k < n; k++ {
				before := edge(i-1, i) + edge(k, k+1)
				after := edge(i-1, k) + edge(i, k+1)
				if after < before-1e-9 {
					for l, r := i, k; l < r; l, r = l+1, r-1 {
						path[l], path[r] = path[r], path[l]
					}
					improved = true
				}
			}
		}
	}
	return path
}

func nearestNeighbor(points []Point, start *Point) []int {
	n := len(points)
	visited := make([]bool, n)
	path := make([]int, 0, n)
	cur := 0
	if start != nil {
		cur = closest(points, visited, *start)
	}
	for len(path) < n {
		visited[cur] = true
		path = append(path, cur)
		if len(path) < n {
			cur = closest(points, visited, points[cur])
		}
	}
	return path
}

func closest(points []Point, visited []bool, from Point) int {
	best, bestDist := -1, math.Inf(1)
	for i, p := range points {
		if d := distance(from, p); !visited[i] && d < bestDist {
			best, bestDist = i, d
		}
	}
	return best
}

func (in optimizeInput) start() (*Point, error) {
	v := apperr.Validator{}
	v.Check((in.Latitude == nil) == (in.Longitude == nil), "latitude", "must be sent together with longitude")
	if in.Latitude != nil && in.Longitude != nil {
		v.Check(*in.Latitude >= -90 && *in.Latitude <= 90, "latitude", "must be between -90 and 90")
		v.Check(*in.Longitude >= -180 && *in.Longitude <= 180, "longitude", "must be between -180 and 180")
	}
	if err := v.Err(); err != nil || in.Latitude == nil {
		return nil, err
	}
	return &Point{Lat: *in.Latitude, Lng: *in.Longitude}, nil
}
