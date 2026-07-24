package processor

import (
	"testing"

	"bomix-app/backend/types"
	"github.com/stretchr/testify/assert"
)

func TestFilter_FilterByView(t *testing.T) {
	filter := NewFilter()

	// Test data setup
	parts := []types.AggregatedPart{
		// SMD parts with different BOM statuses
		{
			MainSupplier:   "Samsung",
			MainSupplierPn: "CL05B104KO5NNNC",
			Type:           "SMD",
			BOMStatus:      "I",
			CCL:            "Y",
			Locations:      "C1,C2",
			Qty:            2,
		},
		{
			MainSupplier:   "Murata",
			MainSupplierPn: "GRM155R71H104KA88D",
			Type:           "SMD",
			BOMStatus:      "P",
			CCL:            "N",
			Locations:      "C3",
			Qty:            1,
		},
		{
			MainSupplier:   "Yageo",
			MainSupplierPn: "CC0603KRX7R9BB224",
			Type:           "SMD",
			BOMStatus:      "M",
			CCL:            "N",
			Locations:      "C4",
			Qty:            1,
		},
		{
			MainSupplier:   "TDK",
			MainSupplierPn: "C1005X7R1H104K",
			Type:           "SMD",
			BOMStatus:      "X",
			CCL:            "N",
			Locations:      "C5",
			Qty:            1,
		},
		// PTH parts
		{
			MainSupplier:   "Kemet",
			MainSupplierPn: "T491A105K016AT",
			Type:           "PTH",
			BOMStatus:      "I",
			CCL:            "N",
			Locations:      "C6",
			Qty:            1,
		},
		{
			MainSupplier:   "AVX",
			MainSupplierPn: "TAJA105K016RNJ",
			Type:           "PTH",
			BOMStatus:      "P",
			CCL:            "N",
			Locations:      "C7",
			Qty:            1,
		},
		// BOTTOM parts
		{
			MainSupplier:   "Osram",
			MainSupplierPn: "LUW W5AM",
			Type:           "BOTTOM",
			BOMStatus:      "I",
			CCL:            "Y",
			Locations:      "LED1",
			Qty:            1,
		},
		{
			MainSupplier:   "Cree",
			MainSupplierPn: "XLamp XP-G",
			Type:           "BOTTOM",
			BOMStatus:      "M",
			CCL:            "N",
			Locations:      "LED2",
			Qty:            1,
		},
	}

	t.Run("ALL view", func(t *testing.T) {
		// ALL view: exclude bom_status = X (shows I, P, M)
		result := filter.FilterByView(parts, "ALL")

		// Should include: Samsung(I), Murata(P), Yageo(M), Kemet(I), AVX(P), Osram(I), Cree(M) -> 7 parts
		// Should exclude: TDK(X) -> 1 part
		assert.Len(t, result, 7)

		for _, p := range result {
			assert.NotEqual(t, "X", p.BOMStatus)
		}
	})

	t.Run("SMD view", func(t *testing.T) {
		// SMD view: type = SMD, exclude bom_status = X
		result := filter.FilterByView(parts, "SMD")

		// Should include: Samsung(I), Murata(P), Yageo(M) -> 3 parts
		// Should exclude: TDK(X) and all PTH/BOTTOM parts
		assert.Len(t, result, 3)

		for _, p := range result {
			assert.Equal(t, "SMD", p.Type)
			assert.NotEqual(t, "X", p.BOMStatus)
		}
	})

	t.Run("PTH view", func(t *testing.T) {
		// PTH view: type = PTH, exclude bom_status = X
		result := filter.FilterByView(parts, "PTH")

		// Should include: Kemet(I), AVX(P) -> 2 parts
		assert.Len(t, result, 2)

		for _, p := range result {
			assert.Equal(t, "PTH", p.Type)
			assert.NotEqual(t, "X", p.BOMStatus)
		}
	})

	t.Run("BOTTOM view", func(t *testing.T) {
		// BOTTOM view: type = BOTTOM, exclude bom_status = X
		result := filter.FilterByView(parts, "BOTTOM")

		// Should include: Osram(I), Cree(M) -> 2 parts
		assert.Len(t, result, 2)

		for _, p := range result {
			assert.Equal(t, "BOTTOM", p.Type)
			assert.NotEqual(t, "X", p.BOMStatus)
		}
	})

	t.Run("NI view", func(t *testing.T) {
		// NI view: bom_status = X
		result := filter.FilterByView(parts, "NI")

		// Should include: TDK(X) -> 1 part
		assert.Len(t, result, 1)
		assert.Equal(t, "TDK", result[0].MainSupplier)
		assert.Equal(t, "X", result[0].BOMStatus)
	})

	t.Run("PROTO view", func(t *testing.T) {
		// PROTO view: bom_status = P
		result := filter.FilterByView(parts, "PROTO")

		// Should include: Murata(P), AVX(P) -> 2 parts
		assert.Len(t, result, 2)

		for _, p := range result {
			assert.Equal(t, "P", p.BOMStatus)
		}
	})

	t.Run("MP view", func(t *testing.T) {
		// MP view: bom_status = M
		result := filter.FilterByView(parts, "MP")

		// Should include: Yageo(M), Cree(M) -> 2 parts
		assert.Len(t, result, 2)

		for _, p := range result {
			assert.Equal(t, "M", p.BOMStatus)
		}
	})

	t.Run("CCL view", func(t *testing.T) {
		// CCL view: ccl = Y and bom_status != X
		result := filter.FilterByView(parts, "CCL")

		// Should include: Samsung(CCL=Y, I), Osram(CCL=Y, I) -> 2 parts
		assert.Len(t, result, 2)

		for _, p := range result {
			assert.Equal(t, "Y", p.CCL)
			assert.NotEqual(t, "X", p.BOMStatus)
		}
	})

	t.Run("case insensitive view names", func(t *testing.T) {
		result1 := filter.FilterByView(parts, "all")
		result2 := filter.FilterByView(parts, "ALL")
		result3 := filter.FilterByView(parts, "All")

		assert.Len(t, result1, len(result2))
		assert.Len(t, result2, len(result3))
	})

	t.Run("unknown view returns all parts", func(t *testing.T) {
		result := filter.FilterByView(parts, "UNKNOWN")
		assert.Len(t, result, len(parts))
	})
}
