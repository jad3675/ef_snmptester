package model

// ObjectGroup represents an object group configuration
type ObjectGroup struct {
	Name    string
	Objects []string
}

// NewObjectGroup creates a new object group
func NewObjectGroup(name string) *ObjectGroup {
	return &ObjectGroup{
		Name:    name,
		Objects: []string{},
	}
}
