package values

import (
	"fmt"
	"time"
)

// ProjectID represents a project identifier value object
type ProjectID struct {
	value string
}

// NewProjectID creates a new ProjectID from a string
func NewProjectID(id string) (*ProjectID, error) {
	if id == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	return &ProjectID{value: id}, nil
}

// GenerateProjectID generates a new unique project ID
func GenerateProjectID() *ProjectID {
	id := fmt.Sprintf("project-%d", time.Now().UnixNano())
	return &ProjectID{value: id}
}

// String returns the string representation of the ProjectID
func (id *ProjectID) String() string {
	return id.value
}

// Value returns the underlying string value
func (id *ProjectID) Value() string {
	return id.value
}

// Equals checks if two ProjectIDs are equal
func (id *ProjectID) Equals(other *ProjectID) bool {
	if other == nil {
		return false
	}
	return id.value == other.value
}
