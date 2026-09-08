package processor

import (
	"sort"
	"strings"

	"bomix-app/backend/db"
	"bomix-app/backend/types"
)

// Aggregator 負責將單一 Revision 的 RevisionComponent 與 PartLocation 聚合為 types.AggregatedPart 列表。
type Aggregator struct{}

// NewAggregator 建立一個新的聚合器實例。
func NewAggregator() *Aggregator {
	return &Aggregator{}
}

// Aggregate 依 (supplier, supplier_pn) 聚合主料與其替代料，並整合 PartLocations 計算數量與狀態。
//
// 參數：
//   - components: Revision 包含的所有零件（主料 Role="M"、替代料 Role="S"）
//   - materials: 物料資料庫快取（用於查詢物料屬性）
//   - locations: 零件位置資訊（關聯至 ComponentID）
//
// 回傳：
//   - []types.AggregatedPart: 聚合後的零件視圖 DTO 列表
func (a *Aggregator) Aggregate(components []db.RevisionComponent, materials []db.Material, locations []db.PartLocation) []types.AggregatedPart {
	// 建立 MaterialID -> Material 映射
	materialByID := make(map[int64]db.Material, len(materials))
	for _, m := range materials {
		materialByID[m.ID] = m
	}

	// 建立 ComponentID -> Locations 映射
	locsByCompID := make(map[int64][]db.PartLocation, len(locations))
	for _, loc := range locations {
		locsByCompID[loc.ComponentID] = append(locsByCompID[loc.ComponentID], loc)
	}

	// 分離主料與替代料
	var mainComponents []db.RevisionComponent
	secondSourcesByParentID := make(map[int64][]db.RevisionComponent)

	for _, c := range components {
		if c.Role == "M" {
			mainComponents = append(mainComponents, c)
		} else if c.Role == "S" {
			secondSourcesByParentID[c.ParentComponentID] = append(secondSourcesByParentID[c.ParentComponentID], c)
		}
	}

	// 依 (Supplier, SupplierPN) 分組主料
	groups := make(map[string][]db.RevisionComponent)
	for _, comp := range mainComponents {
		mat := materialByID[comp.MaterialID]
		key := a.makeGroupKey(mat.Supplier, mat.SupplierPN)
		groups[key] = append(groups[key], comp)
	}

	var result []types.AggregatedPart

	for key, groupComps := range groups {
		locationSet := make(map[string]bool)
		var firstComp db.RevisionComponent
		hasCCL := false
		validLocCount := 0
		pCount := 0
		mCount := 0
		partType := ""

		var allSecondSources []db.RevisionComponent

		for _, c := range groupComps {
			if firstComp.ID == 0 {
				firstComp = c
			}
			compLocs := locsByCompID[c.ID]
			for _, loc := range compLocs {
				if loc.Type != "" && partType == "" {
					partType = loc.Type
				}
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

			// 收集該主料底下的替代料
			if sss, ok := secondSourcesByParentID[c.ID]; ok {
				allSecondSources = append(allSecondSources, sss...)
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

		locList := make([]string, 0, len(locationSet))
		for loc := range locationSet {
			locList = append(locList, loc)
		}
		sort.Strings(locList)

		supplier, supplierPN := a.parseGroupKey(key)
		firstMat := materialByID[firstComp.MaterialID]

		// 組裝 SecondSourceDTO
		ssDTOs := make([]types.SecondSourceDTO, 0, len(allSecondSources))
		for _, ssComp := range allSecondSources {
			ssMat := materialByID[ssComp.MaterialID]
			ssDTOs = append(ssDTOs, types.SecondSourceDTO{
				Hhpn:        ssMat.HHPN,
				Supplier:    ssMat.Supplier,
				SupplierPn:  ssMat.SupplierPN,
				Description: ssMat.Description,
			})
		}

		aggregated := types.AggregatedPart{
			Item:           firstComp.Item,
			MainSupplier:   supplier,
			MainSupplierPn: supplierPN,
			Hhpn:           firstMat.HHPN,
			Description:    firstMat.Description,
			Type:           partType,
			Qty:            len(locList),
			Locations:      strings.Join(locList, ","),
			BOMStatus:      bomStatus,
			CCL:            hasCCL,
			Remark:         firstMat.Remark,
			SecondSources:  ssDTOs,
		}

		result = append(result, aggregated)
	}

	return result
}

// makeGroupKey 產生分組鍵：supplier|supplier_pn
func (a *Aggregator) makeGroupKey(supplier, supplierPN string) string {
	return supplier + "|" + supplierPN
}

// parseGroupKey 解析分組鍵
func (a *Aggregator) parseGroupKey(key string) (string, string) {
	parts := strings.SplitN(key, "|", 2)
	supplier := parts[0]
	supplierPN := ""
	if len(parts) > 1 {
		supplierPN = parts[1]
	}
	return supplier, supplierPN
}

// FormatLocations 將 location slice 格式化為逗號分隔字串
func FormatLocations(locations []string) string {
	return strings.Join(locations, ",")
}
