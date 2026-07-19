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

// FilterByView filters parts based on the view and mode.
// Views: ALL, SMD, PTH, BOTTOM, NI, PROTO, MP, CCL
// Modes: NPI, MP
// See product-spec section 6.4.2
func (f *Filter) FilterByView(parts []types.AggregatedPart, view string, mode string) []types.AggregatedPart {
	switch strings.ToUpper(view) {
	case "ALL":
		return f.filterAll(parts, mode)
	case "SMD":
		return f.filterSMD(parts, mode)
	case "PTH":
		return f.filterPTH(parts, mode)
	case "BOTTOM":
		return f.filterBOTTOM(parts, mode)
	case "NI":
		return f.filterNI(parts, mode)
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
// Mode filtering: NPI shows I+P, MP shows I+M
func (f *Filter) filterAll(parts []types.AggregatedPart, mode string) []types.AggregatedPart {
	var result []types.AggregatedPart
	for _, part := range parts {
		if part.BOMStatus == "X" {
			continue
		}
		if f.matchesMode(part.BOMStatus, mode) {
			result = append(result, part)
		}
	}
	return result
}

// filterSMD returns only SMD parts, excluding bom_status = X
// Mode filtering applies
func (f *Filter) filterSMD(parts []types.AggregatedPart, mode string) []types.AggregatedPart {
	var result []types.AggregatedPart
	for _, part := range parts {
		if part.Type != "SMD" {
			continue
		}
		if part.BOMStatus == "X" {
			continue
		}
		if f.matchesMode(part.BOMStatus, mode) {
			result = append(result, part)
		}
	}
	return result
}

// filterPTH returns only PTH parts, excluding bom_status = X
// Mode filtering applies
func (f *Filter) filterPTH(parts []types.AggregatedPart, mode string) []types.AggregatedPart {
	var result []types.AggregatedPart
	for _, part := range parts {
		if part.Type != "PTH" {
			continue
		}
		if part.BOMStatus == "X" {
			continue
		}
		if f.matchesMode(part.BOMStatus, mode) {
			result = append(result, part)
		}
	}
	return result
}

// filterBOTTOM returns only BOTTOM parts, excluding bom_status = X
// Mode filtering applies
func (f *Filter) filterBOTTOM(parts []types.AggregatedPart, mode string) []types.AggregatedPart {
	var result []types.AggregatedPart
	for _, part := range parts {
		if part.Type != "BOTTOM" {
			continue
		}
		if part.BOMStatus == "X" {
			continue
		}
		if f.matchesMode(part.BOMStatus, mode) {
			result = append(result, part)
		}
	}
	return result
}

// filterNI returns parts based on mode:
// NPI mode: bom_status = X OR M
// MP mode: bom_status = X OR P
func (f *Filter) filterNI(parts []types.AggregatedPart, mode string) []types.AggregatedPart {
	var result []types.AggregatedPart
	for _, part := range parts {
		if mode == "NPI" {
			if part.BOMStatus == "X" || part.BOMStatus == "M" {
				result = append(result, part)
			}
		} else {
			// MP mode
			if part.BOMStatus == "X" || part.BOMStatus == "P" {
				result = append(result, part)
			}
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

// filterCCL returns only parts with ccl = Y
func (f *Filter) filterCCL(parts []types.AggregatedPart) []types.AggregatedPart {
	var result []types.AggregatedPart
	for _, part := range parts {
		if part.CCL == "Y" {
			result = append(result, part)
		}
	}
	return result
}

// matchesMode checks if the bom_status matches the given mode
// NPI mode: shows I and P
// MP mode: shows I and M
func (f *Filter) matchesMode(bomStatus, mode string) bool {
	switch strings.ToUpper(mode) {
	case "NPI":
		return bomStatus == "I" || bomStatus == "P"
	case "MP":
		return bomStatus == "I" || bomStatus == "M"
	default:
		// Default behavior: show all non-X statuses
		return bomStatus != "X"
	}
}
