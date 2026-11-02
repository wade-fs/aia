package knowledge

// SpatialReference captures spatial, geographic, astronomical, or dimensional positioning (Plan2.md §2).
type SpatialReference struct {
	Name        string  `json:"name" yaml:"name"`
	Coordinates *string `json:"coordinates,omitempty" yaml:"coordinates,omitempty"`
	Region      string  `json:"region,omitempty" yaml:"region,omitempty"`
	ParentArea  string  `json:"parent_area,omitempty" yaml:"parent_area,omitempty"`
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
}
