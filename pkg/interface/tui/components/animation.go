package components

import (
	"math"
	"time"
)

// EasingFunc defines an easing function for animations
type EasingFunc func(t float64) float64

// Common easing functions

// EaseInOutCubic provides smooth cubic easing
func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	return 1 - math.Pow(-2*t+2, 3)/2
}

// EaseOutCubic provides cubic ease-out
func EaseOutCubic(t float64) float64 {
	return 1 - math.Pow(1-t, 3)
}

// EaseInCubic provides cubic ease-in
func EaseInCubic(t float64) float64 {
	return t * t * t
}

// Linear provides linear interpolation
func Linear(t float64) float64 {
	return t
}

// EaseInOutQuad provides quadratic easing
func EaseInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

// Animation represents an animation state
type Animation struct {
	Start    time.Time
	Duration time.Duration
	From     float64
	To       float64
	Easing   EasingFunc
	Done     bool
}

// NewAnimation creates a new animation
func NewAnimation(from, to float64, duration time.Duration, easing EasingFunc) *Animation {
	if easing == nil {
		easing = Linear
	}
	return &Animation{
		Start:    time.Now(),
		Duration: duration,
		From:     from,
		To:       to,
		Easing:   easing,
		Done:     false,
	}
}

// Value returns the current interpolated value
func (a *Animation) Value() float64 {
	elapsed := time.Since(a.Start)
	if elapsed >= a.Duration {
		a.Done = true
		return a.To
	}

	t := float64(elapsed) / float64(a.Duration)
	eased := a.Easing(t)
	return a.From + (a.To-a.From)*eased
}

// Progress returns the progress as a value between 0 and 1
func (a *Animation) Progress() float64 {
	elapsed := time.Since(a.Start)
	if elapsed >= a.Duration {
		return 1.0
	}
	return float64(elapsed) / float64(a.Duration)
}

// IsDone returns true if the animation is complete
func (a *Animation) IsDone() bool {
	return a.Done || time.Since(a.Start) >= a.Duration
}

// Reset resets the animation with new values
func (a *Animation) Reset(from, to float64, duration time.Duration) {
	a.Start = time.Now()
	a.From = from
	a.To = to
	a.Duration = duration
	a.Done = false
}

// PulsingAnimation represents a pulsing animation that oscillates
type PulsingAnimation struct {
	Start    time.Time
	Duration time.Duration
	Min      float64
	Max      float64
	Easing   EasingFunc
}

// NewPulsingAnimation creates a new pulsing animation
func NewPulsingAnimation(min, max float64, duration time.Duration, easing EasingFunc) *PulsingAnimation {
	if easing == nil {
		easing = EaseInOutCubic
	}
	return &PulsingAnimation{
		Start:    time.Now(),
		Duration: duration,
		Min:      min,
		Max:      max,
		Easing:   easing,
	}
}

// Value returns the current pulsing value
func (p *PulsingAnimation) Value() float64 {
	elapsed := time.Since(p.Start)
	cycle := float64(elapsed) / float64(p.Duration)
	
	// Use sine wave for smooth oscillation
	t := (math.Sin(cycle*2*math.Pi) + 1) / 2 // Normalize to 0-1
	
	eased := p.Easing(t)
	return p.Min + (p.Max-p.Min)*eased
}

// Reset resets the pulsing animation
func (p *PulsingAnimation) Reset() {
	p.Start = time.Now()
}

