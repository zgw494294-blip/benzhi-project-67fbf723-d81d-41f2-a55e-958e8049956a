package domain

import "math"

func round(value float64) float64 {
	return math.Round(value*100) / 100
}
