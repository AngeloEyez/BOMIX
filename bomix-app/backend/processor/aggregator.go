package processor

import (
	"sort"
	"strings"

	"bomix-app/backend/db"
	"bomix-app/backend/types"
)

// Aggregator handles data aggregation logic
type Aggregator struct{}

// NewAggregator creates a new aggregator instance
func NewAggregator() *Aggregator {
	return &Aggregator{}
}

// Aggregate aggregates parts by (supplier, supplier_pn) and merges them with partLocations.
func (a *Aggregator) Aggregate(parts []db.Part, locations []db.PartLocation, secondSources []db.SecondSource) []types.AggregatedPart {
	// Build partID -> Part lookup
	partByID := make(map[int64]db.Part)
	for _, p := range parts {
		partByID[p.ID] = p
	}

	// Group locations by partID
	locsByPartID := make(map[int64][]db.PartLocation)
	for _, loc := range locations {
		locsByPartID[loc.PartID] = append(locsByPartID[loc.PartID], loc)
	}

	// Group parts by (supplier, supplier_pn)
	groups := make(map[string][]db.Part)
	for _, part := range parts {
		key := a.makeGroupKey(part.Supplier, part.SupplierPN)
		groups[key] = append(groups[key], part)
	}

	// Build second source lookup by group key
	ssByGroup := a.buildSecondSourceMap(secondSources, parts)

	// Create aggregated parts
	var result []types.AggregatedPart

	for key, groupParts := range groups {
		locationSet := make(map[string]bool)
		var firstPart db.Part
		hasCCL := false
		validLocCount := 0
		pCount := 0
		mCount := 0

		for _, p := range groupParts {
			if firstPart.Supplier == "" {
				firstPart = p
			}
			partLocs := locsByPartID[p.ID]
			for _, loc := range partLocs {
				if loc.Location != "" {
					locationSet[loc.Location] = true
				}
				if loc.CCL {
					hasCCL = true
				}
				statusUpper := strings.ToUpper(strings.TrimSpace(loc.BomStatus))
				if statusUpper != "X" {
					validLocCount++
					if statusUpper == "P" {
						pCount++
					} else if statusUpper == "M" {
						mCount++
					}
				}
			}
		}

		bomStatus := "I"
		if validLocCount > 0 {
			if pCount == validLocCount {
				bomStatus = "P"
			} else if mCount == validLocCount {
				bomStatus = "M"
			}
		}

		// Sort locations for consistent output
		locList := make([]string, 0, len(locationSet))
		for loc := range locationSet {
			locList = append(locList, loc)
		}
		sort.Strings(locList)

		quantity := len(locList)
		supplier, supplierPN := a.parseGroupKey(key)

		var ssDTOs []types.SecondSourceDTO
		if groupSS, ok := ssByGroup[key]; ok {
			for _, ss := range groupSS {
				ssDTOs = append(ssDTOs, types.SecondSourceDTO{
					Hhpn:        ss.HHPN,
					Supplier:    ss.Supplier,
					SupplierPn:  ss.SupplierPN,
					Description: ss.Description,
				})
			}
		}

		aggregated := types.AggregatedPart{
			Item:           firstPart.Item,
			MainSupplier:   supplier,
			MainSupplierPn: supplierPN,
			Hhpn:           firstPart.HHPN,
			Description:    firstPart.Description,
			Type:           firstPart.Type,
			Qty:            quantity,
			Locations:      strings.Join(locList, ","),
			BOMStatus:      bomStatus,
			CCL:            hasCCL,
			Remark:         firstPart.Remark,
			SecondSources:  ssDTOs,
		}

		result = append(result, aggregated)
	}

	return result
}

// makeGroupKey creates a group key from supplier and supplier_pn
func (a *Aggregator) makeGroupKey(supplier, supplierPN string) string {
	return supplier + "|" + supplierPN
}

// parseGroupKey parses a group key into supplier and supplier_pn
func (a *Aggregator) parseGroupKey(key string) (string, string) {
	parts := strings.SplitN(key, "|", 2)
	supplier := parts[0]
	supplierPN := ""
	if len(parts) > 1 {
		supplierPN = parts[1]
	}
	return supplier, supplierPN
}

// buildSecondSourceMap builds a map of second sources by group key
// It uses the PartID to look up the corresponding part and build the group key
func (a *Aggregator) buildSecondSourceMap(secondSources []db.SecondSource, parts []db.Part) map[string][]db.SecondSource {
	// Build a lookup map from PartID to Part
	partByID := make(map[int64]db.Part)
	for _, p := range parts {
		partByID[p.ID] = p
	}

	// Build second source map by group key
	ssByGroup := make(map[string][]db.SecondSource)

	for _, ss := range secondSources {
		if part, ok := partByID[ss.PartID]; ok {
			key := a.makeGroupKey(part.Supplier, part.SupplierPN)
			ssByGroup[key] = append(ssByGroup[key], ss)
		}
	}

	return ssByGroup
}

// FormatLocations formats locations as a comma-separated string
func FormatLocations(locations []string) string {
	return strings.Join(locations, ",")
}
