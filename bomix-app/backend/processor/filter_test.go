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
			MainSupplierPn: "C1206C105K5RACTU",
			Type:           "PTH",
			BOMStatus:      "I",
			CCL:            "N",
			Locations:      "C10",
			Qty:            1,
		},
		{
			MainSupplier:   "AVX",
			MainSupplierPn: "08055C104KAT2A",
			Type:           "PTH",
			BOMStatus:      "P",
			CCL:            "N",
			Locations:      "C11",
			Qty:            1,
		},
		// BOTTOM parts
		{
			MainSupplier:   "Osram",
			MainSupplierPn: "SFH 4715",
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

	t.Run("ALL view with NPI mode", func(t *testing.T) {
		// ALL view: exclude bom_status = X, mode NPI shows I+P
		result := filter.FilterByView(parts, "ALL", "NPI")

		// Should include: Samsung(I), Murata(P), Kemet(I), AVX(P), Osram(I)
		// Should exclude: Yageo(M), Cree(M), TDK(X)
		assert.Len(t, result, 5)

		// Verify no X status parts
		for _, p := range result {
			assert.NotEqual(t, "X", p.BOMStatus)
		}

		// Verify only I and P status parts
		for _, p := range result {
			assert.Contains(t, []string{"I", "P"}, p.BOMStatus)
		}
	})

	t.Run("ALL view with MP mode", func(t *testing.T) {
		// ALL view: exclude bom_status = X, mode MP shows I+M
		result := filter.FilterByView(parts, "ALL", "MP")

		// Should include: Samsung(I), Yageo(M), Kemet(I), Osram(I), Cree(M)
		// Should exclude: Murata(P), TDK(X)
		assert.Len(t, result, 5)

		// Verify no X status parts
		for _, p := range result {
			assert.NotEqual(t, "X", p.BOMStatus)
		}

		// Verify only I and M status parts
		for _, p := range result {
			assert.Contains(t, []string{"I", "M"}, p.BOMStatus)
		}
	})

	t.Run("SMD view with NPI mode", func(t *testing.T) {
		// SMD view: type = SMD, exclude bom_status = X, mode NPI shows I+P
		result := filter.FilterByView(parts, "SMD", "NPI")

		// Should include: Samsung(SMD, I), Murata(SMD, P)
		// Should exclude: Yageo(M), TDK(X), and all PTH/BOTTOM parts
		assert.Len(t, result, 2)

		for _, p := range result {
			assert.Equal(t, "SMD", p.Type)
			assert.NotEqual(t, "X", p.BOMStatus)
		}
	})

	t.Run("SMD view with MP mode", func(t *testing.T) {
		// SMD view: type = SMD, exclude bom_status = X, mode MP shows I+M
		result := filter.FilterByView(parts, "SMD", "MP")

		// Should include: Samsung(SMD, I), Yageo(SMD, M)
		// Should exclude: Murata(P), TDK(X), and all PTH/BOTTOM parts
		assert.Len(t, result, 2)

		for _, p := range result {
			assert.Equal(t, "SMD", p.Type)
			assert.NotEqual(t, "X", p.BOMStatus)
		}
	})

	t.Run("PTH view with NPI mode", func(t *testing.T) {
		// PTH view: type = PTH, exclude bom_status = X, mode NPI shows I+P
		result := filter.FilterByView(parts, "PTH", "NPI")

		// Should include: Kemet(PTH, I), AVX(PTH, P)
		assert.Len(t, result, 2)

		for _, p := range result {
			assert.Equal(t, "PTH", p.Type)
		}
	})

	t.Run("PTH view with MP mode", func(t *testing.T) {
		// PTH view: type = PTH, exclude bom_status = X, mode MP shows I+M
		result := filter.FilterByView(parts, "PTH", "MP")

		// Should include: Kemet(PTH, I)
		// Should exclude: AVX(PTH, P)
		assert.Len(t, result, 1)
		assert.Equal(t, "Kemet", result[0].MainSupplier)
	})

	t.Run("BOTTOM view with NPI mode", func(t *testing.T) {
		// BOTTOM view: type = BOTTOM, exclude bom_status = X, mode NPI shows I+P
		result := filter.FilterByView(parts, "BOTTOM", "NPI")

		// Should include: Osram(BOTTOM, I)
		// Should exclude: Cree(BOTTOM, M)
		assert.Len(t, result, 1)
		assert.Equal(t, "Osram", result[0].MainSupplier)
	})

	t.Run("BOTTOM view with MP mode", func(t *testing.T) {
		// BOTTOM view: type = BOTTOM, exclude bom_status = X, mode MP shows I+M
		result := filter.FilterByView(parts, "BOTTOM", "MP")

		// Should include: Osram(BOTTOM, I), Cree(BOTTOM, M)
		assert.Len(t, result, 2)

		for _, p := range result {
			assert.Equal(t, "BOTTOM", p.Type)
		}
	})

	t.Run("NI view with NPI mode", func(t *testing.T) {
		// NI view with NPI: bom_status = X OR M
		result := filter.FilterByView(parts, "NI", "NPI")

		// Should include: TDK(X), Yageo(M), Cree(M) - all parts with X or M status
		assert.Len(t, result, 3)

		for _, p := range result {
			assert.Contains(t, []string{"X", "M"}, p.BOMStatus)
		}
	})

	t.Run("NI view with MP mode", func(t *testing.T) {
		// NI view with MP: bom_status = X OR P
		result := filter.FilterByView(parts, "NI", "MP")

		// Should include: TDK(X), Murata(P), AVX(P)
		assert.Len(t, result, 3)

		for _, p := range result {
			assert.Contains(t, []string{"X", "P"}, p.BOMStatus)
		}
	})

	t.Run("PROTO view", func(t *testing.T) {
		// PROTO view: bom_status = P (mode independent)
		result := filter.FilterByView(parts, "PROTO", "")

		// Should include: Murata(P), AVX(P)
		assert.Len(t, result, 2)

		for _, p := range result {
			assert.Equal(t, "P", p.BOMStatus)
		}
	})

	t.Run("MP view", func(t *testing.T) {
		// MP view: bom_status = M (mode independent)
		result := filter.FilterByView(parts, "MP", "")

		// Should include: Yageo(M), Cree(M)
		assert.Len(t, result, 2)

		for _, p := range result {
			assert.Equal(t, "M", p.BOMStatus)
		}
	})

	t.Run("CCL view", func(t *testing.T) {
		// CCL view: ccl = Y (mode independent)
		result := filter.FilterByView(parts, "CCL", "")

		// Should include: Samsung(CCL=Y), Osram(CCL=Y)
		assert.Len(t, result, 2)

		for _, p := range result {
			assert.Equal(t, "Y", p.CCL)
		}
	})

	t.Run("case insensitive view names", func(t *testing.T) {
		// Test that view names are case insensitive
		result1 := filter.FilterByView(parts, "all", "NPI")
		result2 := filter.FilterByView(parts, "ALL", "NPI")
		result3 := filter.FilterByView(parts, "All", "NPI")

		assert.Len(t, result1, len(result2))
		assert.Len(t, result2, len(result3))
	})

	t.Run("case insensitive mode names", func(t *testing.T) {
		// Test that mode names are case insensitive
		result1 := filter.FilterByView(parts, "ALL", "npi")
		result2 := filter.FilterByView(parts, "ALL", "NPI")
		result3 := filter.FilterByView(parts, "ALL", "Npi")

		assert.Len(t, result1, len(result2))
		assert.Len(t, result2, len(result3))
	})

	t.Run("unknown view returns all parts", func(t *testing.T) {
		// Unknown view should return all parts
		result := filter.FilterByView(parts, "UNKNOWN", "")

		assert.Len(t, result, len(parts))
	})
}

func TestFilter_matchesMode(t *testing.T) {
	filter := NewFilter()

	t.Run("NPI mode matches I status", func(t *testing.T) {
		assert.True(t, filter.matchesMode("I", "NPI"))
	})

	t.Run("NPI mode matches P status", func(t *testing.T) {
		assert.True(t, filter.matchesMode("P", "NPI"))
	})

	t.Run("NPI mode does not match M status", func(t *testing.T) {
		assert.False(t, filter.matchesMode("M", "NPI"))
	})

	t.Run("NPI mode does not match X status", func(t *testing.T) {
		assert.False(t, filter.matchesMode("X", "NPI"))
	})

	t.Run("MP mode matches I status", func(t *testing.T) {
		assert.True(t, filter.matchesMode("I", "MP"))
	})

	t.Run("MP mode matches M status", func(t *testing.T) {
		assert.True(t, filter.matchesMode("M", "MP"))
	})

	t.Run("MP mode does not match P status", func(t *testing.T) {
		assert.False(t, filter.matchesMode("P", "MP"))
	})

	t.Run("MP mode does not match X status", func(t *testing.T) {
		assert.False(t, filter.matchesMode("X", "MP"))
	})

	t.Run("unknown mode defaults to non-X", func(t *testing.T) {
		assert.True(t, filter.matchesMode("I", "UNKNOWN"))
		assert.True(t, filter.matchesMode("P", "UNKNOWN"))
		assert.True(t, filter.matchesMode("M", "UNKNOWN"))
		assert.False(t, filter.matchesMode("X", "UNKNOWN"))
	})
}
