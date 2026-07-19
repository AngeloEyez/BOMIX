package processor

import (
	"testing"

	"bomix-app/backend/db"
	"bomix-app/backend/types"
	"github.com/stretchr/testify/assert"
)

func TestAggregator_Aggregate(t *testing.T) {
	aggregator := NewAggregator()

	t.Run("merge parts with same supplier and supplier_pn", func(t *testing.T) {
		parts := []db.Part{
			{
				ID:          1,
				Type:        "SMD",
				Supplier:    "Samsung",
				SupplierPN:  "CL05B104KO5NNNC",
				Description: "CAP,22uF,+/-20%,X5R,6.3V,SMD0603",
				Location:    "C1",
				BOMStatus:   "I",
				CCL:         "Y",
			},
			{
				ID:          2,
				Type:        "SMD",
				Supplier:    "Samsung",
				SupplierPN:  "CL05B104KO5NNNC",
				Description: "CAP,22uF,+/-20%,X5R,6.3V,SMD0603",
				Location:    "C2",
				BOMStatus:   "I",
				CCL:         "Y",
			},
			{
				ID:          3,
				Type:        "SMD",
				Supplier:    "Samsung",
				SupplierPN:  "CL05B104KO5NNNC",
				Description: "CAP,22uF,+/-20%,X5R,6.3V,SMD0603",
				Location:    "C3",
				BOMStatus:   "I",
				CCL:         "Y",
			},
		}

		secondSources := []db.SecondSource{}

		result := aggregator.Aggregate(parts, secondSources)

		assert.Len(t, result, 1)
		assert.Equal(t, "Samsung", result[0].MainSupplier)
		assert.Equal(t, "CL05B104KO5NNNC", result[0].MainSupplierPn)
		assert.Equal(t, 3, result[0].Qty)
		assert.Contains(t, result[0].Locations, "C1")
		assert.Contains(t, result[0].Locations, "C2")
		assert.Contains(t, result[0].Locations, "C3")
	})

	t.Run("calculate qty from location count", func(t *testing.T) {
		parts := []db.Part{
			{
				ID:          1,
				Type:        "PTH",
				Supplier:    "Murata",
				SupplierPN:  "GRM155R71H104KA88D",
				Description: "CAP,100nF,+/-10%,X7R,50V,SMD0402",
				Location:    "C10",
				BOMStatus:   "I",
				CCL:         "N",
			},
			{
				ID:          2,
				Type:        "PTH",
				Supplier:    "Murata",
				SupplierPN:  "GRM155R71H104KA88D",
				Description: "CAP,100nF,+/-10%,X7R,50V,SMD0402",
				Location:    "C11",
				BOMStatus:   "I",
				CCL:         "N",
			},
		}

		secondSources := []db.SecondSource{}

		result := aggregator.Aggregate(parts, secondSources)

		assert.Len(t, result, 1)
		assert.Equal(t, 2, result[0].Qty)
	})

	t.Run("merge locations as comma-separated string", func(t *testing.T) {
		parts := []db.Part{
			{
				ID:          1,
				Type:        "SMD",
				Supplier:    "Yageo",
				SupplierPN:  "RC0402FR-0710KL",
				Description: "RES,10K,+/-1%,0603",
				Location:    "R1",
				BOMStatus:   "I",
				CCL:         "N",
			},
			{
				ID:          2,
				Type:        "SMD",
				Supplier:    "Yageo",
				SupplierPN:  "RC0402FR-0710KL",
				Description: "RES,10K,+/-1%,0603",
				Location:    "R2",
				BOMStatus:   "I",
				CCL:         "N",
			},
			{
				ID:          3,
				Type:        "SMD",
				Supplier:    "Yageo",
				SupplierPN:  "RC0402FR-0710KL",
				Description: "RES,10K,+/-1%,0603",
				Location:    "R3",
				BOMStatus:   "I",
				CCL:         "N",
			},
		}

		secondSources := []db.SecondSource{}

		result := aggregator.Aggregate(parts, secondSources)

		assert.Len(t, result, 1)
		locations := result[0].Locations
		assert.Contains(t, locations, "R1")
		assert.Contains(t, locations, "R2")
		assert.Contains(t, locations, "R3")
	})

	t.Run("attach second sources to aggregated part", func(t *testing.T) {
		parts := []db.Part{
			{
				ID:          1,
				Type:        "SMD",
				Supplier:    "Samsung",
				SupplierPN:  "CL05B104KO5NNNC",
				Description: "CAP,22uF,+/-20%,X5R,6.3V,SMD0603",
				Location:    "C1",
				BOMStatus:   "I",
				CCL:         "Y",
			},
		}

		secondSources := []db.SecondSource{
			{
				PartID:      1,
				Supplier:    "Yageo",
				SupplierPN:  "CC0603KRX7R9BB224",
				Description: "CAP,220nF,+/-10%,X7R,10V,SMD0603",
			},
			{
				PartID:      1,
				Supplier:    "Murata",
				SupplierPN:  "GRM188R61A226ME15D",
				Description: "CAP,22uF,+/-20%,X5R,10V,SMD0805",
			},
		}

		result := aggregator.Aggregate(parts, secondSources)

		assert.Len(t, result, 1)
		assert.Len(t, result[0].SecondSources, 2)

		// Check second source 1
		assert.Equal(t, "Yageo", result[0].SecondSources[0].Supplier)
		assert.Equal(t, "CC0603KRX7R9BB224", result[0].SecondSources[0].SupplierPn)

		// Check second source 2
		assert.Equal(t, "Murata", result[0].SecondSources[1].Supplier)
		assert.Equal(t, "GRM188R61A226ME15D", result[0].SecondSources[1].SupplierPn)
	})

	t.Run("multiple groups with second sources", func(t *testing.T) {
		samsungPart := db.Part{
			ID:          1,
			Type:        "SMD",
			Supplier:    "Samsung",
			SupplierPN:  "CL05B104KO5NNNC",
			Description: "CAP,22uF,+/-20%,X5R,6.3V,SMD0603",
			Location:    "C1",
			BOMStatus:   "I",
			CCL:         "Y",
		}
		murataPart := db.Part{
			ID:          2,
			Type:        "SMD",
			Supplier:    "Murata",
			SupplierPN:  "GRM155R71H104KA88D",
			Description: "CAP,100nF,+/-10%,X7R,50V,SMD0402",
			Location:    "C10",
			BOMStatus:   "I",
			CCL:         "N",
		}

		parts := []db.Part{samsungPart, samsungPart, murataPart}
		parts[0].Location = "C1"
		parts[1].Location = "C2"

		secondSources := []db.SecondSource{
			{
				PartID:      1,
				Supplier:    "Yageo",
				SupplierPN:  "CC0603KRX7R9BB224",
				Description: "CAP,220nF,+/-10%,X7R,10V,SMD0603",
			},
			{
				PartID:      2,
				Supplier:    "TDK",
				SupplierPN:  "C1005X7R1H104K",
				Description: "CAP,100nF,+/-10%,X7R,50V,SMD0402",
			},
		}

		result := aggregator.Aggregate(parts, secondSources)

		assert.Len(t, result, 2)

		// Find Samsung group
		var samsungResult *types.AggregatedPart
		var murataResult *types.AggregatedPart
		for i := range result {
			if result[i].MainSupplier == "Samsung" {
				samsungResult = &result[i]
			}
			if result[i].MainSupplier == "Murata" {
				murataResult = &result[i]
			}
		}

		assert.NotNil(t, samsungResult)
		assert.Equal(t, 2, samsungResult.Qty)
		assert.Len(t, samsungResult.SecondSources, 1)
		assert.Equal(t, "Yageo", samsungResult.SecondSources[0].Supplier)

		assert.NotNil(t, murataResult)
		assert.Equal(t, 1, murataResult.Qty)
		assert.Len(t, murataResult.SecondSources, 1)
		assert.Equal(t, "TDK", murataResult.SecondSources[0].Supplier)
	})

	t.Run("handle parts with different bom_status", func(t *testing.T) {
		parts := []db.Part{
			{
				ID:          1,
				Type:        "SMD",
				Supplier:    "Samsung",
				SupplierPN:  "CL05B104KO5NNNC",
				Description: "CAP,22uF,+/-20%,X5R,6.3V,SMD0603",
				Location:    "C1",
				BOMStatus:   "I",
				CCL:         "Y",
			},
			{
				ID:          2,
				Type:        "SMD",
				Supplier:    "Samsung",
				SupplierPN:  "CL05B104KO5NNNC",
				Description: "CAP,22uF,+/-20%,X5R,6.3V,SMD0603",
				Location:    "C2",
				BOMStatus:   "P",
				CCL:         "Y",
			},
		}

		secondSources := []db.SecondSource{}

		result := aggregator.Aggregate(parts, secondSources)

		assert.Len(t, result, 1)
		assert.Equal(t, 2, result[0].Qty)
		// Note: The first part's BOMStatus is used
		assert.Equal(t, "I", result[0].BOMStatus)
	})

	t.Run("handle empty input", func(t *testing.T) {
		parts := []db.Part{}
		secondSources := []db.SecondSource{}

		result := aggregator.Aggregate(parts, secondSources)

		assert.Len(t, result, 0)
	})
}

func TestFormatLocations(t *testing.T) {
	t.Run("format multiple locations", func(t *testing.T) {
		locations := []string{"C1", "C2", "C3", "R1", "R2"}
		result := FormatLocations(locations)
		assert.Equal(t, "C1,C2,C3,R1,R2", result)
	})

	t.Run("format single location", func(t *testing.T) {
		locations := []string{"C1"}
		result := FormatLocations(locations)
		assert.Equal(t, "C1", result)
	})

	t.Run("format empty locations", func(t *testing.T) {
		locations := []string{}
		result := FormatLocations(locations)
		assert.Equal(t, "", result)
	})
}

func TestAggregator_makeGroupKey(t *testing.T) {
	aggregator := NewAggregator()

	t.Run("create group key", func(t *testing.T) {
		key := aggregator.makeGroupKey("Samsung", "CL05B104KO5NNNC")
		assert.Equal(t, "Samsung|CL05B104KO5NNNC", key)
	})
}

func TestAggregator_parseGroupKey(t *testing.T) {
	aggregator := NewAggregator()

	t.Run("parse group key", func(t *testing.T) {
		supplier, supplierPN := aggregator.parseGroupKey("Samsung|CL05B104KO5NNNC")
		assert.Equal(t, "Samsung", supplier)
		assert.Equal(t, "CL05B104KO5NNNC", supplierPN)
	})

	t.Run("parse group key without supplier_pn", func(t *testing.T) {
		supplier, supplierPN := aggregator.parseGroupKey("Samsung")
		assert.Equal(t, "Samsung", supplier)
		assert.Equal(t, "", supplierPN)
	})
}
