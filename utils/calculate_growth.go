package utils

import "fmt"

func CalculateGrowthRate(current, previous int32) string {
	if previous == 0 {
		if current == 0 {
			return "0%"
		}
		return "100%"
	}
	growth := (float64(current-previous) / float64(previous)) * 100
	return fmt.Sprintf("%.2f%%", growth)
}
