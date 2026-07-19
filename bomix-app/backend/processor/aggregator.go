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

// Aggregate aggregates parts by (supplier, supplier_pn) and merges them.
// It groups parts with the same supplier and supplier_pn, merges locations into a comma-separated string,
// calculates quantity as the count of locations, and attaches second sources.
func (a *Aggregator) Aggregate(parts []db.Part, secondSources []db.SecondSource) []types.AggregatedPart {
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
		// Collect unique locations and find first part for metadata
		locationSet := make(map[string]bool)
		var firstPart db.Part

		for _, p := range groupParts {
			if p.Location != "" {
				locationSet[p.Location] = true
			}
			if firstPart.Supplier == "" {
				firstPart = p
			}
		}

		// Sort locations for consistent output
		locations := make([]string, 0, len(locationSet))
		for loc := range locationSet {
			locations = append(locations, loc)
		}
		sort.Strings(locations)

		// Calculate quantity as number of locations
		quantity := len(locations)

		// Parse the key to get supplier and supplier_pn
		supplier, supplierPN := a.parseGroupKey(key)

		// Build second sources for this group
		var ssDTOs []types.SecondSourceDTO
		if groupSS, ok := ssByGroup[key]; ok {
			for _, ss := range groupSS {
				ssDTOs = append(ssDTOs, types.SecondSourceDTO{
					Hhpn:        firstPart.Type, // Using Type as hhpn reference
					Supplier:    ss.Supplier,
					SupplierPn:  ss.SupplierPN,
					Description: ss.Description,
				})
			}
		}

		aggregated := types.AggregatedPart{
			Item:           firstPart.Type, // Using Type as item reference
			MainSupplier:   supplier,
			MainSupplierPn: supplierPN,
			Hhpn:           firstPart.Type,
			Description:    firstPart.Description,
			Type:           firstPart.Type,
			Qty:            quantity,
			Locations:      strings.Join(locations, ","),
			BOMStatus:      firstPart.BOMStatus,
			CCL:            firstPart.CCL,
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
