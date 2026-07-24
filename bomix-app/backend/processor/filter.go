package processor

import (
	"strings"

	"bomix-app/backend/types"
)

// Filter handles view-based filtering of aggregated parts
type Filter struct{}

// NewFilter creates a new filter instance
func NewFilter() *Filter {
	return &Filter{}
}

// FilterByView filters parts based on the view.
// Views: ALL, SMD, PTH, BOTTOM, NI, PROTO, MP, CCL
// See product-spec section 6.4.2
func (f *Filter) FilterByView(parts []types.AggregatedPart, view string) []types.AggregatedPart {
	switch strings.ToUpper(view) {
	case "ALL":
		return f.filterAll(parts)
	case "SMD":
		return f.filterSMD(parts)
	case "PTH":
		return f.filterPTH(parts)
	case "BOTTOM":
		return f.filterBOTTOM(parts)
	case "NI":
		return f.filterNI(parts)
	case "PROTO":
		return f.filterPROTO(parts)
	case "MP":
		return f.filterMP(parts)
	case "CCL":
		return f.filterCCL(parts)
	default:
		// Default to returning all parts
		return parts
	}
}

// filterAll returns all parts except those with bom_status = X
func (f *Filter) filterAll(parts []types.AggregatedPart) []types.AggregatedPart {
	var result []types.AggregatedPart
	for _, part := range parts {
		if part.BOMStatus != "X" {
			result = append(result, part)
		}
	}
	return result
}

// filterSMD returns only SMD parts, excluding bom_status = X
func (f *Filter) filterSMD(parts []types.AggregatedPart) []types.AggregatedPart {
	var result []types.AggregatedPart
	for _, part := range parts {
		if part.Type == "SMD" && part.BOMStatus != "X" {
			result = append(result, part)
		}
	}
	return result
}

// filterPTH returns only PTH parts, excluding bom_status = X
func (f *Filter) filterPTH(parts []types.AggregatedPart) []types.AggregatedPart {
	var result []types.AggregatedPart
	for _, part := range parts {
		if part.Type == "PTH" && part.BOMStatus != "X" {
			result = append(result, part)
		}
	}
	return result
}

// filterBOTTOM returns only BOTTOM parts, excluding bom_status = X
func (f *Filter) filterBOTTOM(parts []types.AggregatedPart) []types.AggregatedPart {
	var result []types.AggregatedPart
	for _, part := range parts {
		if part.Type == "BOTTOM" && part.BOMStatus != "X" {
			result = append(result, part)
		}
	}
	return result
}

// filterNI returns parts with bom_status = X
func (f *Filter) filterNI(parts []types.AggregatedPart) []types.AggregatedPart {
	var result []types.AggregatedPart
	for _, part := range parts {
		if part.BOMStatus == "X" {
			result = append(result, part)
		}
	}
	return result
}

// filterPROTO returns only parts with bom_status = P
func (f *Filter) filterPROTO(parts []types.AggregatedPart) []types.AggregatedPart {
	var result []types.AggregatedPart
	for _, part := range parts {
		if part.BOMStatus == "P" {
			result = append(result, part)
		}
	}
	return result
}

// filterMP returns only parts with bom_status = M
func (f *Filter) filterMP(parts []types.AggregatedPart) []types.AggregatedPart {
	var result []types.AggregatedPart
	for _, part := range parts {
		if part.BOMStatus == "M" {
			result = append(result, part)
		}
	}
	return result
}

// filterCCL returns only parts with ccl = Y and bom_status != X
func (f *Filter) filterCCL(parts []types.AggregatedPart) []types.AggregatedPart {
	var result []types.AggregatedPart
	for _, part := range parts {
		if part.CCL == "Y" && part.BOMStatus != "X" {
			result = append(result, part)
		}
	}
	return result
}
