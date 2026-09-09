package view

import (
	"fmt"
	"sort"
	"strings"

	"bomix-app/backend/db"
	"bomix-app/backend/logger"

	"gorm.io/gorm"
)

// Service 是 View 系統的核心服務。
//
// 設計為完全無狀態（stateless）：Service 本身不持有任何查詢中間狀態，
// 每次 Query() 呼叫皆獨立進行，可安全地被多個 goroutine 同時呼叫
// （例如前端顯示與後端匯出同時以不同條件查詢）。
//
// Service 依賴注入 *gorm.DB，但僅用於讀取操作，不進行任何寫入。
type Service struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewService 建立一個新的 View 服務實例。
//
// 參數：
//   - db：GORM 資料庫連線，View 服務將使用此連線進行唯讀查詢。
//   - lg：選填的 Logger 實例（可用於紀錄 debug log）
//
// 回傳：
//   - *Service：View 服務實例
func NewService(db *gorm.DB, lg ...*logger.Logger) *Service {
	svc := &Service{db: db}
	if len(lg) > 0 && lg[0] != nil {
		svc.logger = lg[0]
	}
	return svc
}

// rawRevisionData 是從資料庫取得的單一 revision 原始資料容器（內部使用）。
// 採用延遲載入（Late-Binding）設計，此階段絕不載入 materials 全域表。
type rawRevisionData struct {
	revision      db.BomRevision
	project       db.Project
	components    []db.RevisionComponent
	partLocations []db.PartLocation
	models        []db.MatrixModel
	selections    []db.MatrixSelection
}

// Query 執行視圖查詢，是 View 系統的唯一入口。
//
// 支援單一與多 BOM Revision 查詢：
//   - query.RevisionIDs 傳入 1 個 ID → 單一 revision 視圖
//   - query.RevisionIDs 傳入多個 ID → 多 revision 整合聯集視圖
//
// 整合聯集與延遲載入演算法：
//  1. loadRawData：批量查詢 revision, components, locations, selections，絕不查詢 materials。
//  2. mergeRevisions：以 MaterialID 為核心鍵，進行 locations, BOMStatus, selections 聚合。
//  3. 套用視圖過濾（ViewType）。
//  4. hydratePartGroups：過濾完成後，針對最終需要呈現的物料群組，單次批量查詢 materials 表回填字串屬性。
//
// 此函數為無狀態同步呼叫，可並發執行。
//
// 參數：
//   - query：查詢條件，包含 RevisionIDs、ViewType、ModeOverride
//
// 回傳：
//   - *ViewResult：查詢結果，包含聚合物料群組與 revision 元資料
//   - error：若資料庫查詢失敗則回傳錯誤
func (s *Service) Query(query ViewQuery) (*ViewResult, error) {
	if s.logger != nil {
		s.logger.Debug(fmt.Sprintf("[ViewService.Query] 執行 View 查詢: RevisionIDs=%v, ViewType=%s",
			query.RevisionIDs, query.ViewType))
	}

	if len(query.RevisionIDs) == 0 {
		return &ViewResult{
			Query:      query,
			PartGroups: []ViewPartGroup{},
			Revisions:  []ViewRevision{},
		}, nil
	}

	// 1. 從資料庫載入所有 revision 的原始資料（延遲載入：不查 materials）
	rawData, err := s.loadRawData(query.RevisionIDs)
	if err != nil {
		return nil, fmt.Errorf("view: 載入資料失敗: %w", err)
	}

	// 2. 建立 ViewRevision 元資料列表（依 query.RevisionIDs 指定的順序）
	revisions := buildViewRevisions(query.RevisionIDs, rawData)

	// 3. 執行多 revision 聯集合併，以 MaterialID 聚合
	partGroups := s.mergeRevisions(rawData, query)

	// 4. 取得 Mode（優先使用 query.ModeOverride，若無則從第一份 Revision 取得）
	mode := "NPI"
	if query.ModeOverride != "" {
		mode = strings.ToUpper(strings.TrimSpace(query.ModeOverride))
	} else if len(query.RevisionIDs) > 0 {
		if firstData, ok := rawData[query.RevisionIDs[0]]; ok && firstData.revision.Mode != "" {
			mode = strings.ToUpper(strings.TrimSpace(firstData.revision.Mode))
		}
	}

	// 5. 套用視圖過濾
	filter := NewFilter()
	partGroups = filter.Apply(partGroups, query, mode)

	// 6. 最後一步：批次回填物料字串資料（Late-Binding）
	if err := s.hydratePartGroups(partGroups); err != nil {
		return nil, fmt.Errorf("view: 回填物料資訊失敗: %w", err)
	}

	return &ViewResult{
		Query:      query,
		PartGroups: partGroups,
		Revisions:  revisions,
	}, nil
}

// loadRawData 從資料庫批量載入指定 revision 的所有必要資料。
//
// 採用批量查詢策略（IN 子句），避免 N+1 查詢問題。
// 嚴格遵守延遲載入設計：此函式絕不查詢 materials 全域表。
//
// 參數：
//   - revisionIDs：要載入的 BOM Revision ID 列表
//
// 回傳：
//   - map[int64]*rawRevisionData：以 Revision ID 為鍵的原始資料映射
//   - error：若任何必要資料查詢失敗則回傳錯誤
func (s *Service) loadRawData(revisionIDs []int64) (map[int64]*rawRevisionData, error) {
	result := make(map[int64]*rawRevisionData, len(revisionIDs))
	for _, id := range revisionIDs {
		result[id] = &rawRevisionData{}
	}

	// 1. 批量查詢 BomRevisions（含 Project）
	var revs []db.BomRevision
	if err := s.db.Where("id IN ?", revisionIDs).Find(&revs).Error; err != nil {
		return nil, fmt.Errorf("查詢 BomRevisions 失敗: %w", err)
	}

	// 收集所有 ProjectID 以批量查詢
	projectIDs := make([]int64, 0, len(revs))
	for _, r := range revs {
		if data, ok := result[r.ID]; ok {
			data.revision = r
		}
		projectIDs = append(projectIDs, r.ProjectID)
	}

	// 批量查詢 Projects
	var projects []db.Project
	if err := s.db.Where("id IN ?", projectIDs).Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("查詢 Projects 失敗: %w", err)
	}
	projectMap := make(map[int64]db.Project, len(projects))
	for _, p := range projects {
		projectMap[p.ID] = p
	}
	for id, data := range result {
		data.project = projectMap[data.revision.ProjectID]
		result[id] = data
	}

	// 2. 批量查詢 RevisionComponents
	var components []db.RevisionComponent
	if err := s.db.Where("revision_id IN ?", revisionIDs).Find(&components).Error; err != nil {
		return nil, fmt.Errorf("查詢 RevisionComponents 失敗: %w", err)
	}
	componentIDs := make([]int64, 0, len(components))
	for _, c := range components {
		if data, ok := result[c.RevisionID]; ok {
			data.components = append(data.components, c)
		}
		componentIDs = append(componentIDs, c.ID)
	}

	// 2.1 批量查詢 PartLocations
	if len(componentIDs) > 0 {
		var locations []db.PartLocation
		if err := s.db.Where("component_id IN ?", componentIDs).Find(&locations).Error; err != nil {
			return nil, fmt.Errorf("查詢 PartLocations 失敗: %w", err)
		}
		compToRev := make(map[int64]int64, len(components))
		for _, c := range components {
			compToRev[c.ID] = c.RevisionID
		}
		for _, loc := range locations {
			if revID, ok := compToRev[loc.ComponentID]; ok {
				if data, ok := result[revID]; ok {
					data.partLocations = append(data.partLocations, loc)
				}
			}
		}
	}

	// 3. 批量查詢 MatrixModels
	var models []db.MatrixModel
	if err := s.db.Where("revision_id IN ?", revisionIDs).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("查詢 MatrixModels 失敗: %w", err)
	}
	for _, m := range models {
		if data, ok := result[m.RevisionID]; ok {
			data.models = append(data.models, m)
		}
	}

	// 4. 批量查詢 MatrixSelections
	var selections []db.MatrixSelection
	if err := s.db.Where("revision_id IN ?", revisionIDs).Find(&selections).Error; err != nil {
		return nil, fmt.Errorf("查詢 MatrixSelections 失敗: %w", err)
	}
	for _, sel := range selections {
		if data, ok := result[sel.RevisionID]; ok {
			data.selections = append(data.selections, sel)
		}
	}

	return result, nil
}

// buildViewRevisions 從 rawRevisionData 建立 ViewRevision 元資料列表。
//
// 輸出列表的順序優先與 requestedIDs 指定的順序一致。
// 若有未在 requestedIDs 中指定的 Revision，則依 ID 升冪補齊。
func buildViewRevisions(requestedIDs []int64, rawData map[int64]*rawRevisionData) []ViewRevision {
	revisions := make([]ViewRevision, 0, len(rawData))
	processed := make(map[int64]bool, len(rawData))

	for _, id := range requestedIDs {
		data, ok := rawData[id]
		if !ok || processed[id] {
			continue
		}
		revisions = append(revisions, createViewRevisionFromRaw(id, data))
		processed[id] = true
	}

	remainingIDs := make([]int64, 0, len(rawData)-len(processed))
	for id := range rawData {
		if !processed[id] {
			remainingIDs = append(remainingIDs, id)
		}
	}
	sort.Slice(remainingIDs, func(i, j int) bool { return remainingIDs[i] < remainingIDs[j] })

	for _, id := range remainingIDs {
		data := rawData[id]
		revisions = append(revisions, createViewRevisionFromRaw(id, data))
	}

	return revisions
}

// createViewRevisionFromRaw 從 rawRevisionData 建立單一 ViewRevision 元資料物件。
func createViewRevisionFromRaw(id int64, data *rawRevisionData) ViewRevision {
	rev := data.revision

	modelNames := make([]string, 0, len(data.models))
	modelQty := make(map[string]int, len(data.models))
	modelQtyByOrder := make(map[int]int, len(data.models))
	for _, m := range data.models {
		modelNames = append(modelNames, m.ModelName)
		modelQty[m.ModelName] = m.Qty
		modelQtyByOrder[m.SortOrder] = m.Qty
	}
	sort.Strings(modelNames)

	return ViewRevision{
		ID:               id,
		ProjectCode:      data.project.Code,
		Phase:            rev.Phase,
		Version:          rev.Version,
		Description:      rev.Description,
		SchematicVersion: rev.SchematicVersion,
		PCBVersion:       rev.PCBVersion,
		PCAPN:            rev.PCAPN,
		Date:             rev.Date,
		SourceFile:       rev.SourceFile,
		ModelNames:       modelNames,
		ModelQty:         modelQty,
		ModelQtyByOrder:  modelQtyByOrder,
	}
}

// matGroupKey 建立物料群組鍵：以 MaterialID 為核心，可選擇性附加製程類型 (SMD/PTH/BOTTOM)
func matGroupKey(materialID int64, partType ...string) string {
	k := fmt.Sprintf("%d", materialID)
	if len(partType) > 0 && partType[0] != "" {
		k += "|" + strings.ToUpper(strings.TrimSpace(partType[0]))
	}
	return k
}

// groupKey 建立物料群組字串識別鍵，格式為 "supplier|supplier_pn" 或 "supplier|supplier_pn|type"
func groupKey(supplier, supplierPN string, partType ...string) string {
	k := supplier + "|" + supplierPN
	if len(partType) > 0 && partType[0] != "" {
		k += "|" + strings.ToUpper(strings.TrimSpace(partType[0]))
	}
	return k
}

// isEffectiveBOMStatus 判斷指定物料的 bom_status 是否有效（非 X 上件狀態）。
func isEffectiveBOMStatus(bomStatus string) bool {
	status := strings.ToUpper(strings.TrimSpace(bomStatus))
	return status != "X"
}

// mergeRevisions 執行多 BOM Revision 的主料與替代料聯集合併，
// 並蒐集 Model 勾選狀態，建立 ViewPartGroup 列表。
//
// 完全基於 MaterialID 進行記憶體分組與數值運算，不涉及 Material 表查詢。
func (s *Service) mergeRevisions(rawData map[int64]*rawRevisionData, query ViewQuery) []ViewPartGroup {
	type partGroupBuilder struct {
		group     ViewPartGroup
		ssBuilder map[int64]*ViewSecondSource // 以 ss.MaterialID 為鍵
	}

	builders := make(map[string]*partGroupBuilder) // 主料 matGroupKey → builder
	keyOrder := make([]string, 0)

	// 建立 model ID → ModelName 與 ModelID → SortOrder 的全局映射（跨所有 revision）
	modelIDToName := make(map[int64]string)
	modelIDToSortOrder := make(map[int64]int)
	modelIDToQty := make(map[int64]int)
	for _, data := range rawData {
		for _, m := range data.models {
			modelIDToName[m.ID] = m.ModelName
			modelIDToSortOrder[m.ID] = m.SortOrder
			modelIDToQty[m.ID] = m.Qty
		}
	}

	// 按 query.RevisionIDs 的順序遍歷
	for _, revID := range query.RevisionIDs {
		data, ok := rawData[revID]
		if !ok {
			continue
		}

		// 區分主料 ("M") 與替代料 ("S")
		compByID := make(map[int64]db.RevisionComponent, len(data.components))
		mainComps := make([]db.RevisionComponent, 0, len(data.components))
		ssByParentCompID := make(map[int64][]db.RevisionComponent)
		for _, c := range data.components {
			compByID[c.ID] = c
			if c.Role == "M" {
				mainComps = append(mainComps, c)
			} else if c.Role == "S" {
				ssByParentCompID[c.ParentComponentID] = append(ssByParentCompID[c.ParentComponentID], c)
			}
		}

		// 按 ComponentID 收集 PartLocation
		allLocsByCompID := make(map[int64][]db.PartLocation)
		for _, loc := range data.partLocations {
			allLocsByCompID[loc.ComponentID] = append(allLocsByCompID[loc.ComponentID], loc)
		}

		// 追蹤是否有 MatrixSelection
		hasSelectionByCompID := make(map[int64]bool)
		hasSelectionByMaterialID := make(map[int64]bool)
		for _, sel := range data.selections {
			if sel.ComponentID != 0 {
				hasSelectionByCompID[sel.ComponentID] = true
			}
			if sel.MainMaterialID != 0 {
				hasSelectionByMaterialID[sel.MainMaterialID] = true
			}
		}

		// 找出有效主料：具有打件位置（無論上件或不上件）或具有 Matrix Selection 的主料
		validComps := make([]db.RevisionComponent, 0, len(mainComps))
		for _, c := range mainComps {
			if len(allLocsByCompID[c.ID]) > 0 || hasSelectionByCompID[c.ID] || hasSelectionByMaterialID[c.MaterialID] {
				validComps = append(validComps, c)
			}
		}

		// 建立此 revision 的 Selection 映射：MainMaterialID -> SortOrder / ModelName -> SelectedMaterialID
		selMatByMainMatByOrder := make(map[int64]map[int]int64)
		selMatByMainMatByName := make(map[int64]map[string]int64)

		for _, sel := range data.selections {
			mainMatID := sel.MainMaterialID
			if mainMatID == 0 && sel.ComponentID != 0 {
				if c, ok := compByID[sel.ComponentID]; ok {
					mainMatID = c.MaterialID
				}
			}
			if mainMatID == 0 {
				continue
			}

			sortOrder, okOrder := modelIDToSortOrder[sel.ModelID]
			modelName, okName := modelIDToName[sel.ModelID]

			if okOrder {
				if selMatByMainMatByOrder[mainMatID] == nil {
					selMatByMainMatByOrder[mainMatID] = make(map[int]int64)
				}
				selMatByMainMatByOrder[mainMatID][sortOrder] = sel.SelectedMaterialID
			}

			if okName && modelName != "" {
				if selMatByMainMatByName[mainMatID] == nil {
					selMatByMainMatByName[mainMatID] = make(map[string]int64)
				}
				selMatByMainMatByName[mainMatID][modelName] = sel.SelectedMaterialID
			}
		}

		// 處理此 revision 的主料與 PartLocations（按 (MaterialID, Location.Type/NI) 歸類）
		representativeComps := make(map[string]db.RevisionComponent)
		typeByGroup := make(map[string]string)
		locationsByGroup := make(map[string]map[string]bool)
		cclByGroup := make(map[string]bool)
		statusByGroup := make(map[string]string)
		validLocCountByGroup := make(map[string]int)
		pCountByGroup := make(map[string]int)
		mCountByGroup := make(map[string]int)
		xCountByGroup := make(map[string]int)

		for _, c := range validComps {
			locs := allLocsByCompID[c.ID]

			if len(locs) == 0 {
				// 若物料完全沒有任何 location（僅有 selection），使用無 type 鍵歸類
				baseKey := matGroupKey(c.MaterialID)
				if _, exists := representativeComps[baseKey]; !exists {
					representativeComps[baseKey] = c
				}
				continue
			}

			for _, loc := range locs {
				statusUpper := strings.ToUpper(strings.TrimSpace(loc.BomStatus))
				var key string
				if statusUpper == "X" {
					// 不上件位置獨立歸類，避免與同料之上件位置混合
					key = matGroupKey(c.MaterialID, "NI")
				} else {
					key = matGroupKey(c.MaterialID, loc.Type)
				}

				if locationsByGroup[key] == nil {
					locationsByGroup[key] = make(map[string]bool)
				}

				locationsByGroup[key][loc.Location] = true
				typeByGroup[key] = loc.Type
				if loc.CCL {
					cclByGroup[key] = true
				}
				validLocCountByGroup[key]++
				if statusUpper == "P" {
					pCountByGroup[key]++
				} else if statusUpper == "M" {
					mCountByGroup[key]++
				} else if statusUpper == "X" {
					xCountByGroup[key]++
				}
				if _, exists := representativeComps[key]; !exists {
					representativeComps[key] = c
				}
			}
		}

		// 計算此 revision 中各群組的 BOMStatus
		for key := range representativeComps {
			totalLocs := validLocCountByGroup[key]
			if totalLocs > 0 && xCountByGroup[key] == totalLocs {
				statusByGroup[key] = "X"
			} else if totalLocs > 0 && pCountByGroup[key] == totalLocs {
				statusByGroup[key] = "P"
			} else if totalLocs > 0 && mCountByGroup[key] == totalLocs {
				statusByGroup[key] = "M"
			} else {
				statusByGroup[key] = "I"
			}
		}

		// 將此 revision 的群組合併至 builders
		for key, repComp := range representativeComps {
			b, exists := builders[key]
			if !exists {
				locs := sortedLocations(locationsByGroup[key])
				bomStat := statusByGroup[key]
				if bomStat == "" {
					bomStat = "I"
				}
				partType := typeByGroup[key]
				b = &partGroupBuilder{
					group: ViewPartGroup{
						MaterialID:            repComp.MaterialID,
						Item:                  repComp.Item,
						Type:                  partType,
						BOMStatus:             bomStat,
						CCL:                   cclByGroup[key],
						Qty:                   len(locationsByGroup[key]),
						Locations:             locs,
						SourceRevisionIDs:     []int64{revID},
						MainSelectionsByOrder: make(map[int]bool),
					},
					ssBuilder: make(map[int64]*ViewSecondSource),
				}
				builders[key] = b
				keyOrder = append(keyOrder, key)
			} else {
				b.group.SourceRevisionIDs = appendUnique(b.group.SourceRevisionIDs, revID)
				if b.group.Type == "" && typeByGroup[key] != "" {
					b.group.Type = typeByGroup[key]
				}
				if cclByGroup[key] {
					b.group.CCL = true
				}
				if b.group.MainSelectionsByOrder == nil {
					b.group.MainSelectionsByOrder = make(map[int]bool)
				}
				if b.group.BOMStatus != statusByGroup[key] {
					b.group.BOMStatus = "I"
				}
			}

			// 更新主料 Selection 狀態
			if selOrderMap := selMatByMainMatByOrder[repComp.MaterialID]; selOrderMap != nil {
				for sortOrder, selectedMatID := range selOrderMap {
					if selectedMatID == repComp.MaterialID {
						b.group.MainSelectionsByOrder[sortOrder] = true
					}
				}
			}

			// 檢查與組裝此 revision 屬於該主料的 2nd Source
			for _, ss := range ssByParentCompID[repComp.ID] {
				existing, ok := b.ssBuilder[ss.MaterialID]
				if !ok {
					existing = &ViewSecondSource{
						MaterialID:        ss.MaterialID,
						SourceRevisionIDs: []int64{revID},
						SelectionsByOrder: make(map[int]bool),
					}
					b.ssBuilder[ss.MaterialID] = existing
				} else {
					existing.SourceRevisionIDs = appendUnique(existing.SourceRevisionIDs, revID)
					if existing.SelectionsByOrder == nil {
						existing.SelectionsByOrder = make(map[int]bool)
					}
				}

				if selOrderMap := selMatByMainMatByOrder[repComp.MaterialID]; selOrderMap != nil {
					for sortOrder, selectedMatID := range selOrderMap {
						if selectedMatID == ss.MaterialID {
							existing.SelectionsByOrder[sortOrder] = true
						}
					}
				}
			}

			// 蒐集此 revision 的 MatrixSelections
			selOrderMap := selMatByMainMatByOrder[repComp.MaterialID]
			selNameMap := selMatByMainMatByName[repComp.MaterialID]

			for _, mData := range rawData[revID].models {
				sortOrder := mData.SortOrder
				var selectedMatID int64

				if selOrderMap != nil {
					selectedMatID = selOrderMap[sortOrder]
				}
				if selectedMatID == 0 && selNameMap != nil && mData.ModelName != "" {
					selectedMatID = selNameMap[mData.ModelName]
				}

				b.group.Selections = append(b.group.Selections, ViewModelSelection{
					RevisionID:         revID,
					SortOrder:          sortOrder,
					ModelName:          mData.ModelName,
					ModelQty:           mData.Qty,
					SelectedMaterialID: selectedMatID,
				})
			}
		}
	}

	// 組裝結果列表
	result := make([]ViewPartGroup, 0, len(builders))
	for _, key := range keyOrder {
		b := builders[key]

		ssList := make([]ViewSecondSource, 0, len(b.ssBuilder))
		for _, ss := range b.ssBuilder {
			ssList = append(ssList, *ss)
		}
		b.group.SecondSources = ssList

		result = append(result, b.group)
	}

	// 依 Location 自然排序
	sort.SliceStable(result, func(i, j int) bool {
		return compareLocStr(result[i].Locations, result[j].Locations)
	})

	return result
}

// hydratePartGroups 批次回填過濾後之物料詳細屬性（Late-Binding）。
//
// 僅對最終留下來的 PartGroups 進行單次批量查詢 materials 表，回填字串欄位。
func (s *Service) hydratePartGroups(partGroups []ViewPartGroup) error {
	if len(partGroups) == 0 {
		return nil
	}

	// 1. 收集所有出現的不重複 MaterialID
	matIDMap := make(map[int64]bool)
	for _, g := range partGroups {
		if g.MaterialID != 0 {
			matIDMap[g.MaterialID] = true
		}
		for _, ss := range g.SecondSources {
			if ss.MaterialID != 0 {
				matIDMap[ss.MaterialID] = true
			}
		}
		for _, sel := range g.Selections {
			if sel.SelectedMaterialID != 0 {
				matIDMap[sel.SelectedMaterialID] = true
			}
		}
	}

	if len(matIDMap) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(matIDMap))
	for id := range matIDMap {
		ids = append(ids, id)
	}

	// 2. 單次批量查詢 materials 表
	var materials []db.Material
	if err := s.db.Where("id IN ?", ids).Find(&materials).Error; err != nil {
		return fmt.Errorf("hydratePartGroups: 查詢 materials 失敗: %w", err)
	}

	materialMap := make(map[int64]db.Material, len(materials))
	for _, m := range materials {
		materialMap[m.ID] = m
	}

	// 3. 回填字串屬性
	for i := range partGroups {
		g := &partGroups[i]
		if mat, ok := materialMap[g.MaterialID]; ok {
			g.MainSupplier = mat.Supplier
			g.MainSupplierPN = mat.SupplierPN
			g.HHPN = mat.HHPN
			g.Description = mat.Description
			g.Remark = mat.Remark
			g.Notes = mat.Notes
		}

		for j := range g.SecondSources {
			ss := &g.SecondSources[j]
			if mat, ok := materialMap[ss.MaterialID]; ok {
				ss.Supplier = mat.Supplier
				ss.SupplierPN = mat.SupplierPN
				ss.HHPN = mat.HHPN
				ss.Description = mat.Description
				ss.Remark = mat.Remark
				ss.Notes = mat.Notes
			}
		}

		// 依 (Supplier, SupplierPN) 排序 SecondSources
		sort.Slice(g.SecondSources, func(a, b int) bool {
			ka := g.SecondSources[a].Supplier + "|" + g.SecondSources[a].SupplierPN
			kb := g.SecondSources[b].Supplier + "|" + g.SecondSources[b].SupplierPN
			return ka < kb
		})

		for k := range g.Selections {
			sel := &g.Selections[k]
			if sel.SelectedMaterialID != 0 {
				if mat, ok := materialMap[sel.SelectedMaterialID]; ok {
					sel.SelectedSupplier = mat.Supplier
					sel.SelectedPN = mat.SupplierPN
					sel.SelectedMaterial = mat.Supplier + "|" + mat.SupplierPN
				}
			}
		}
	}

	return nil
}

// sortedLocations 將 location set 轉換為排序後的逗號分隔字串
func sortedLocations(locSet map[string]bool) string {
	locs := make([]string, 0, len(locSet))
	for loc := range locSet {
		locs = append(locs, loc)
	}
	sort.Strings(locs)
	return strings.Join(locs, ",")
}

// appendUnique 將 id 附加至 ids slice（若不重複）
func appendUnique(ids []int64, id int64) []int64 {
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
}

// compareLocStr 比較兩個逗號分隔的 Location 字串大小，用於物料群組排序。
func compareLocStr(a, b string) bool {
	if a == "" && b == "" {
		return false
	}
	if a == "" {
		return false
	}
	if b == "" {
		return true
	}

	firstA := a
	if idx := strings.IndexByte(a, ','); idx >= 0 {
		firstA = a[:idx]
	}
	firstB := b
	if idx := strings.IndexByte(b, ','); idx >= 0 {
		firstB = b[:idx]
	}

	keysA := naturalSortKey(firstA)
	keysB := naturalSortKey(firstB)

	for i := 0; i < len(keysA) && i < len(keysB); i++ {
		if keysA[i] != keysB[i] {
			return keysA[i] < keysB[i]
		}
	}
	return len(keysA) < len(keysB)
}

// naturalSortKey 將位置字串拆解為字母與數字交替的 token 片段，用於自然排序比較。
func naturalSortKey(s string) []string {
	var tokens []string
	var cur []byte
	isDigit := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		curIsDigit := c >= '0' && c <= '9'
		if i == 0 {
			isDigit = curIsDigit
		}
		if curIsDigit != isDigit {
			if isDigit {
				tokens = append(tokens, fmt.Sprintf("%09s", string(cur)))
			} else {
				tokens = append(tokens, string(cur))
			}
			cur = cur[:0]
			isDigit = curIsDigit
		}
		cur = append(cur, c)
	}

	if len(cur) > 0 {
		if isDigit {
			tokens = append(tokens, fmt.Sprintf("%09s", string(cur)))
		} else {
			tokens = append(tokens, string(cur))
		}
	}
	return tokens
}

// MergePartGroupsByMaterial 將包含 Type 維度的物料群組列表，進一步去除 Type 維度，
// 依據 MaterialID 或 (MainSupplier, MainSupplierPN) 進行二階物料合併。
//
// 適用場景：
//   - BigMatrix 匯出：單一工作表呈現全體物料，同一物料跨製程（如 SMD + BOTTOM）需合併為單一 Main Source。
func MergePartGroupsByMaterial(groups []ViewPartGroup) []ViewPartGroup {
	if len(groups) == 0 {
		return []ViewPartGroup{}
	}

	type materialGroupBuilder struct {
		group         ViewPartGroup
		locSet        map[string]bool
		allProto      bool
		allMP         bool
		hasStatus     bool
		ssBuilder     map[string]*ViewSecondSource
		selectionsMap map[string]ViewModelSelection // key: revID_sortOrder
	}

	builders := make(map[string]*materialGroupBuilder)
	keyOrder := make([]string, 0)

	for _, g := range groups {
		var matKey string
		if g.MaterialID != 0 {
			matKey = fmt.Sprintf("MAT_%d", g.MaterialID)
		} else {
			matKey = g.MainSupplier + "|" + g.MainSupplierPN
		}

		b, exists := builders[matKey]
		if !exists {
			locSet := make(map[string]bool)
			for _, loc := range atomizeLocations(g.Locations) {
				locSet[loc] = true
			}

			ssMap := make(map[string]*ViewSecondSource, len(g.SecondSources))
			for _, ss := range g.SecondSources {
				var ssKey string
				if ss.MaterialID != 0 {
					ssKey = fmt.Sprintf("MAT_%d", ss.MaterialID)
				} else {
					ssKey = ss.Supplier + "|" + ss.SupplierPN
				}
				ssCopy := ss
				ssCopy.SourceRevisionIDs = append([]int64(nil), ss.SourceRevisionIDs...)
				ssCopy.SelectionsByOrder = make(map[int]bool, len(ss.SelectionsByOrder))
				for k, v := range ss.SelectionsByOrder {
					ssCopy.SelectionsByOrder[k] = v
				}
				ssMap[ssKey] = &ssCopy
			}

			selMap := make(map[string]ViewModelSelection, len(g.Selections))
			for _, sel := range g.Selections {
				selKey := fmt.Sprintf("%d_%d", sel.RevisionID, sel.SortOrder)
				selMap[selKey] = sel
			}

			mainSelMap := make(map[int]bool, len(g.MainSelectionsByOrder))
			for k, v := range g.MainSelectionsByOrder {
				mainSelMap[k] = v
			}

			statusUpper := strings.ToUpper(strings.TrimSpace(g.BOMStatus))
			isP := (statusUpper == "P")
			isM := (statusUpper == "M")

			b = &materialGroupBuilder{
				group: ViewPartGroup{
					MaterialID:            g.MaterialID,
					MainSupplier:          g.MainSupplier,
					MainSupplierPN:        g.MainSupplierPN,
					Item:                  g.Item,
					HHPN:                  g.HHPN,
					Description:           g.Description,
					Type:                  g.Type,
					BOMStatus:             g.BOMStatus,
					CCL:                   g.CCL,
					Remark:                g.Remark,
					Notes:                 g.Notes,
					SourceRevisionIDs:     append([]int64(nil), g.SourceRevisionIDs...),
					MainSelectionsByOrder: mainSelMap,
				},
				locSet:        locSet,
				allProto:      isP,
				allMP:         isM,
				hasStatus:     true,
				ssBuilder:     ssMap,
				selectionsMap: selMap,
			}
			builders[matKey] = b
			keyOrder = append(keyOrder, matKey)
		} else {
			// 主料已存在：進行屬性合併
			if b.group.Item == "" && g.Item != "" {
				b.group.Item = g.Item
			}
			if b.group.HHPN == "" && g.HHPN != "" {
				b.group.HHPN = g.HHPN
			}
			if b.group.Description == "" && g.Description != "" {
				b.group.Description = g.Description
			}
			if b.group.Type == "" && g.Type != "" {
				b.group.Type = g.Type
			}
			if b.group.Remark == "" && g.Remark != "" {
				b.group.Remark = g.Remark
			}
			if b.group.Notes == "" && g.Notes != "" {
				b.group.Notes = g.Notes
			}

			// Locations 聯集
			for _, loc := range atomizeLocations(g.Locations) {
				b.locSet[loc] = true
			}

			// CCL 聯集
			if g.CCL {
				b.group.CCL = true
			}

			// BOMStatus 判斷
			statusUpper := strings.ToUpper(strings.TrimSpace(g.BOMStatus))
			if statusUpper != "P" {
				b.allProto = false
			}
			if statusUpper != "M" {
				b.allMP = false
			}

			// SourceRevisionIDs 聯集
			for _, revID := range g.SourceRevisionIDs {
				b.group.SourceRevisionIDs = appendUnique(b.group.SourceRevisionIDs, revID)
			}

			// MainSelectionsByOrder 合併
			for order, isSel := range g.MainSelectionsByOrder {
				if isSel {
					b.group.MainSelectionsByOrder[order] = true
				}
			}

			// SecondSources 聯集
			for _, ss := range g.SecondSources {
				var ssKey string
				if ss.MaterialID != 0 {
					ssKey = fmt.Sprintf("MAT_%d", ss.MaterialID)
				} else {
					ssKey = ss.Supplier + "|" + ss.SupplierPN
				}
				existingSS, ok := b.ssBuilder[ssKey]
				if !ok {
					ssCopy := ss
					ssCopy.SourceRevisionIDs = append([]int64(nil), ss.SourceRevisionIDs...)
					ssCopy.SelectionsByOrder = make(map[int]bool, len(ss.SelectionsByOrder))
					for k, v := range ss.SelectionsByOrder {
						ssCopy.SelectionsByOrder[k] = v
					}
					b.ssBuilder[ssKey] = &ssCopy
				} else {
					for _, rID := range ss.SourceRevisionIDs {
						existingSS.SourceRevisionIDs = appendUnique(existingSS.SourceRevisionIDs, rID)
					}
					if existingSS.Notes == "" && ss.Notes != "" {
						existingSS.Notes = ss.Notes
					}
					for order, isSel := range ss.SelectionsByOrder {
						if isSel {
							existingSS.SelectionsByOrder[order] = true
						}
					}
				}
			}

			// Selections 合併
			for _, sel := range g.Selections {
				selKey := fmt.Sprintf("%d_%d", sel.RevisionID, sel.SortOrder)
				existingSel, ok := b.selectionsMap[selKey]
				if !ok || (existingSel.SelectedPN == "" && existingSel.SelectedMaterialID == 0) {
					b.selectionsMap[selKey] = sel
				}
			}
		}
	}

	// 組合最終結果
	result := make([]ViewPartGroup, 0, len(builders))
	for _, matKey := range keyOrder {
		b := builders[matKey]

		b.group.Locations = sortedLocations(b.locSet)
		b.group.Qty = len(b.locSet)

		if b.allProto {
			b.group.BOMStatus = "P"
		} else if b.allMP {
			b.group.BOMStatus = "M"
		} else {
			b.group.BOMStatus = "I"
		}

		ssList := make([]ViewSecondSource, 0, len(b.ssBuilder))
		for _, ss := range b.ssBuilder {
			ssList = append(ssList, *ss)
		}
		sort.Slice(ssList, func(i, j int) bool {
			ki := ssList[i].Supplier + "|" + ssList[i].SupplierPN
			kj := ssList[j].Supplier + "|" + ssList[j].SupplierPN
			return ki < kj
		})
		b.group.SecondSources = ssList

		selections := make([]ViewModelSelection, 0, len(b.selectionsMap))
		for _, sel := range b.selectionsMap {
			selections = append(selections, sel)
		}
		sort.Slice(selections, func(i, j int) bool {
			if selections[i].RevisionID != selections[j].RevisionID {
				return selections[i].RevisionID < selections[j].RevisionID
			}
			return selections[i].SortOrder < selections[j].SortOrder
		})
		b.group.Selections = selections

		result = append(result, b.group)
	}

	sort.SliceStable(result, func(i, j int) bool {
		return compareLocStr(result[i].Locations, result[j].Locations)
	})

	return result
}

// atomizeLocations 將逗號分隔的 location 字串拆分為獨立的 location 陣列
func atomizeLocations(locationStr string) []string {
	raw := strings.Split(locationStr, ",")
	result := make([]string, 0, len(raw))
	for _, p := range raw {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
