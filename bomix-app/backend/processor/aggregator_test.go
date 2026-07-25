package processor

import (
	"testing"

	"bomix-app/backend/db"
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
			},
		}

		locations := []db.PartLocation{
			{PartID: 1, Location: "C1", BomStatus: "I", CCL: true},
			{PartID: 1, Location: "C2", BomStatus: "I", CCL: true},
			{PartID: 1, Location: "C3", BomStatus: "I", CCL: true},
		}

		secondSources := []db.SecondSource{}

		result := aggregator.Aggregate(parts, locations, secondSources)

		assert.Len(t, result, 1)
		assert.Equal(t, "Samsung", result[0].MainSupplier)
		assert.Equal(t, "CL05B104KO5NNNC", result[0].MainSupplierPn)
		assert.Equal(t, 3, result[0].Qty)
		assert.Contains(t, result[0].Locations, "C1")
		assert.Contains(t, result[0].Locations, "C2")
		assert.Contains(t, result[0].Locations, "C3")
		assert.True(t, result[0].CCL)
	})

	t.Run("calculate qty from location count", func(t *testing.T) {
		parts := []db.Part{
			{
				ID:          1,
				Type:        "PTH",
				Supplier:    "Murata",
				SupplierPN:  "GRM155R71H104KA88D",
				Description: "CAP,100nF,+/-10%,X7R,50V,SMD0402",
			},
		}

		locations := []db.PartLocation{
			{PartID: 1, Location: "C10", BomStatus: "I", CCL: false},
			{PartID: 1, Location: "C11", BomStatus: "I", CCL: false},
		}

		secondSources := []db.SecondSource{}

		result := aggregator.Aggregate(parts, locations, secondSources)

		assert.Len(t, result, 1)
		assert.Equal(t, 2, result[0].Qty)
		assert.False(t, result[0].CCL)
	})

	t.Run("merge locations as comma-separated string", func(t *testing.T) {
		parts := []db.Part{
			{
				ID:          1,
				Type:        "SMD",
				Supplier:    "Yageo",
				SupplierPN:  "RC0402FR-0710KL",
				Description: "RES,10K,+/-1%,0603",
			},
		}

		locations := []db.PartLocation{
			{PartID: 1, Location: "R1", BomStatus: "I", CCL: false},
			{PartID: 1, Location: "R2", BomStatus: "I", CCL: false},
			{PartID: 1, Location: "R3", BomStatus: "I", CCL: false},
		}

		secondSources := []db.SecondSource{}

		result := aggregator.Aggregate(parts, locations, secondSources)

		assert.Len(t, result, 1)
		locs := result[0].Locations
		assert.Contains(t, locs, "R1")
		assert.Contains(t, locs, "R2")
		assert.Contains(t, locs, "R3")
	})

	t.Run("attach second sources to aggregated part", func(t *testing.T) {
		parts := []db.Part{
			{
				ID:          1,
				Type:        "SMD",
				Supplier:    "Samsung",
				SupplierPN:  "CL05B104KO5NNNC",
				Description: "CAP,22uF,+/-20%,X5R,6.3V,SMD0603",
			},
		}

		locations := []db.PartLocation{
			{PartID: 1, Location: "C1", BomStatus: "I", CCL: true},
		}

		secondSources := []db.SecondSource{
			{
				PartID:      1,
				Supplier:    "Yageo",
				SupplierPN:  "CC0603KRX7R9BB224",
				Description: "CAP,220nF,+/-10%,X7R,10V,SMD0603",
			},
		}

		result := aggregator.Aggregate(parts, locations, secondSources)

		assert.Len(t, result, 1)
		assert.Len(t, result[0].SecondSources, 1)
		assert.Equal(t, "Yageo", result[0].SecondSources[0].Supplier)
	})
}
