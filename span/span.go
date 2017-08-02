package span

import (
	"math"
	"fmt"
)

//
// Calculate whether a Range between two float64s overlap and by how much
// ref: https://stackoverflow.com/questions/325933/determine-whether-two-date-ranges-overlap/325964#325964
//
type Span struct {
	start, end float64,
}

func New(start, end float64) Span {
	return Span{ start: start, end: end }
}

// return true if Period p intersects Period q, else false
func (p Span) Intersects(q Span) bool {
	return (p.start < q.end) && (p.end > q.start);
}

// return the amount that Period q overlaps Period p (0 if no overlap)
func (p Span) Overlap(q Span) (float64) {
	if !p.Intersects(q) {
		return 0.0
	}
	
	return min(p.end - p.start,
		   p.end - q.start,
		   q.end - q.start,
		   q.end - p.start)
}

// return the lowest of an arbitrary number of float64s
func min(nums ...float64) (x float64) {
	x := -math.MaxFloat64
	for _, num := range nums {
		if (num < x) {
			x = num
		}
	}

	return
}

func (p Span) String() string {
	return fmt.Sprintf("Start: %f, End: %f", p.start, p.end)
}