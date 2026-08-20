package types

type MarkConfig struct {
	MermaidScale  float64
	D2Scale       float64
	ASCIIScale    float64
	DropFirstH1   bool
	StripNewlines bool
	Features      []string
	ImageAlign    string

	// Cloud indicates the target is Confluence Cloud, whose editor (fabric)
	// only supports pixel-based table column widths. When set, percentage
	// colwidth directives are converted to pixels so they survive editor
	// round-trips.
	Cloud bool
}
