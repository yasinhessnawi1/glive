package components

import (
	"fmt"
	"math"
	"strings"
)

// Sparkline represents a sparkline graph
type Sparkline struct {
	Data   []float64
	Width  int
	Height int
	Min    float64
	Max    float64
}

// NewSparkline creates a new sparkline
func NewSparkline(data []float64, width int) *Sparkline {
	if width <= 0 {
		width = 20
	}

	spark := &Sparkline{
		Data:   data,
		Width:  width,
		Height: 1,
	}

	if len(data) > 0 {
		spark.calculateBounds()
	}

	return spark
}

// calculateBounds calculates min and max values
func (s *Sparkline) calculateBounds() {
	if len(s.Data) == 0 {
		return
	}

	s.Min = s.Data[0]
	s.Max = s.Data[0]

	for _, v := range s.Data {
		if v < s.Min {
			s.Min = v
		}
		if v > s.Max {
			s.Max = v
		}
	}

	// Add padding to avoid edge cases
	rangeVal := s.Max - s.Min
	if rangeVal == 0 {
		rangeVal = 1
	}
	s.Min -= rangeVal * 0.1
	s.Max += rangeVal * 0.1
}

// Render renders the sparkline as a string
func (s *Sparkline) Render() string {
	if len(s.Data) == 0 {
		return strings.Repeat("─", s.Width)
	}

	// Normalize data to fit width
	normalized := s.normalizeData()

	// Sparkline characters (Unicode block elements)
	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	result := make([]rune, len(normalized))
	for i, v := range normalized {
		// Map value to block index (0-7)
		idx := int(math.Round(v * 7))
		if idx < 0 {
			idx = 0
		}
		if idx > 7 {
			idx = 7
		}
		result[i] = blocks[idx]
	}

	return string(result)
}

// normalizeData normalizes data points to fit the sparkline width
func (s *Sparkline) normalizeData() []float64 {
	if len(s.Data) == 0 {
		return nil
	}

	// If we have more data points than width, sample
	sampled := s.sampleData()

	// Normalize to 0-1 range
	normalized := make([]float64, len(sampled))
	rangeVal := s.Max - s.Min
	if rangeVal == 0 {
		rangeVal = 1
	}

	for i, v := range sampled {
		normalized[i] = (v - s.Min) / rangeVal
		if normalized[i] < 0 {
			normalized[i] = 0
		}
		if normalized[i] > 1 {
			normalized[i] = 1
		}
	}

	return normalized
}

// sampleData samples data to fit width
func (s *Sparkline) sampleData() []float64 {
	if len(s.Data) <= s.Width {
		return s.Data
	}

	// Downsample using simple averaging
	sampled := make([]float64, s.Width)
	step := float64(len(s.Data)) / float64(s.Width)

	for i := 0; i < s.Width; i++ {
		start := int(math.Round(float64(i) * step))
		end := int(math.Round(float64(i+1) * step))
		if end > len(s.Data) {
			end = len(s.Data)
		}
		if start >= end {
			start = end - 1
		}
		if start < 0 {
			start = 0
		}

		// Average values in this range
		sum := 0.0
		count := 0
		for j := start; j < end; j++ {
			sum += s.Data[j]
			count++
		}
		if count > 0 {
			sampled[i] = sum / float64(count)
		} else {
			sampled[i] = s.Data[start]
		}
	}

	return sampled
}

// AddDataPoint adds a new data point and removes the oldest if needed
func (s *Sparkline) AddDataPoint(value float64) {
	s.Data = append(s.Data, value)
	
	// Keep only last N points (where N is roughly 2x width for smoothness)
	maxPoints := s.Width * 2
	if len(s.Data) > maxPoints {
		s.Data = s.Data[len(s.Data)-maxPoints:]
	}

	s.calculateBounds()
}

// RenderWithLabel renders sparkline with a label and value
func (s *Sparkline) RenderWithLabel(label string, currentValue float64, unit string) string {
	sparkline := s.Render()
	
	if unit == "" {
		return fmt.Sprintf("%s: %s  %.1f", label, sparkline, currentValue)
	}
	return fmt.Sprintf("%s: %s  %.1f %s", label, sparkline, currentValue, unit)
}

