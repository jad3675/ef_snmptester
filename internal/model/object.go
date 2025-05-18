package model

// Object represents an SNMP object with OIDs
type Object struct {
	Name               string
	MIB                string
	DiscoveryAttribute string
	Attributes         map[string]Attribute
}

// Attribute represents an SNMP attribute with OID
type Attribute struct {
	OID        string
	Name       string
	Syntax     string
	Rediscover string
}

// NewObject creates a new object
func NewObject(name string) *Object {
	return &Object{
		Name:       name,
		Attributes: make(map[string]Attribute),
	}
}
