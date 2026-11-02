package knowledge

// TemporalReference anchors knowledge to dates, eras, or relative sequences (Plan2.md §2).
type TemporalReference struct {
	AnchorYear    *int   `json:"anchor_year,omitempty" yaml:"anchor_year,omitempty"`
	AnchorMonth   *int   `json:"anchor_month,omitempty" yaml:"anchor_month,omitempty"`
	AnchorDay     *int   `json:"anchor_day,omitempty" yaml:"anchor_day,omitempty"`
	StartYear     *int   `json:"start_year,omitempty" yaml:"start_year,omitempty"`
	EndYear       *int   `json:"end_year,omitempty" yaml:"end_year,omitempty"`
	EraOrDynasty  string `json:"era_or_dynasty,omitempty" yaml:"era_or_dynasty,omitempty"`
	RawExpression string `json:"raw_expression,omitempty" yaml:"raw_expression,omitempty"`
	SequenceOrder int    `json:"sequence_order,omitempty" yaml:"sequence_order,omitempty"`
}

// Year returns the point anchor year or start year if present.
func (t *TemporalReference) Year() int {
	if t.AnchorYear != nil {
		return *t.AnchorYear
	}
	if t.StartYear != nil {
		return *t.StartYear
	}
	return 0
}

// IsRange reports whether this temporal reference spans across multiple years.
func (t *TemporalReference) IsRange() bool {
	return t.StartYear != nil && t.EndYear != nil && *t.EndYear > *t.StartYear
}

// ContainsYear checks if a given year falls within the bounds of this temporal reference.
func (t *TemporalReference) ContainsYear(year int) bool {
	if t.AnchorYear != nil && *t.AnchorYear == year {
		return true
	}
	if t.StartYear != nil && t.EndYear != nil {
		return year >= *t.StartYear && year <= *t.EndYear
	}
	if t.StartYear != nil && year >= *t.StartYear {
		return true
	}
	if t.EndYear != nil && year <= *t.EndYear {
		return true
	}
	return false
}
