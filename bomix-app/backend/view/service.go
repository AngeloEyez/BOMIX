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

// rawRevisionData 是從資料庫取得的單一 revision 原始資料容器（內部使用）
type rawRevisionData struct {
	revision      db.BomRevision
	project       db.Project
	parts         []db.Part
	partLocations []db.PartLocation
	secondSources []db.SecondSource
	models        []db.MatrixModel
	selections    []db.MatrixSelection
}

// Query 執行視圖查詢，是 View 系統的唯一入口。
//
// 支援單一與多 BOM Revision 查詢：
//   - query.RevisionIDs 傳入 1 個 ID → 單一 revision 視圖
//   - query.RevisionIDs 傳入多個 ID → 多 revision 整合聯集視圖
//
// 整合聯集演算法：
//  1. 以 (supplier, supplier_pn) 為群組鍵，建立主料聯集
//  2. 每個群組記錄 SourceRevisionIDs（出現在哪些 revision）
//  3. 替代料也取所有 revision 的聯集（以 supplier+supplier_pn 去重）
//  4. Model 勾選狀態蒐集所有 revision × model 的完整矩陣
//  5. 套用視圖過濾（ViewType）
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

	// 從資料庫載入所有 revision 的原始資料
	rawData, err := s.loadRawData(query.RevisionIDs)
	if err != nil {
		return nil, fmt.Errorf("view: 載入資料失敗: %w", err)
	}

	// 建立 ViewRevision 元資料列表（依 query.RevisionIDs 指定的順序）
	revisions := buildViewRevisions(query.RevisionIDs, rawData)

	// 執行多 revision 聯集合併，建立 ViewPartGroup 列表
	partGroups := s.mergeRevisions(rawData, query)

	// 取得 Mode（優先使用 query.ModeOverride，若無則從第一份 Revision 取得）
	mode := "NPI"
	if query.ModeOverride != "" {
		mode = strings.ToUpper(strings.TrimSpace(query.ModeOverride))
	} else if len(query.RevisionIDs) > 0 {
		if firstData, ok := rawData[query.RevisionIDs[0]]; ok && firstData.revision.Mode != "" {
			mode = strings.ToUpper(strings.TrimSpace(firstData.revision.Mode))
		}
	}

	// 套用視圖過濾
	filter := NewFilter()
	partGroups = filter.Apply(partGroups, query, mode)

	return &ViewResult{
		Query:      query,
		PartGroups: partGroups,
		Revisions:  revisions,
	}, nil
}

// loadRawData 從資料庫批量載入指定 revision 的所有必要資料。
//
// 採用批量查詢策略（IN 子句），避免 N+1 查詢問題。
// 每種資料型別只進行一次資料庫查詢。
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
	// 將 project 回填至 rawRevisionData
	for id, data := range result {
		data.project = projectMap[data.revision.ProjectID]
		result[id] = data
	}

	// 2. 批量查詢 Parts
	var parts []db.Part
	if err := s.db.Where("revision_id IN ?", revisionIDs).Find(&parts).Error; err != nil {
		return nil, fmt.Errorf("查詢 Parts 失敗: %w", err)
	}
	partIDs := make([]int64, 0, len(parts))
	for _, p := range parts {
		if data, ok := result[p.RevisionID]; ok {
			data.parts = append(data.parts, p)
		}
		partIDs = append(partIDs, p.ID)
	}

	// 2.1 批量查詢 PartLocations
	var locations []db.PartLocation
	if len(partIDs) > 0 {
		if err := s.db.Where("part_id IN ?", partIDs).Find(&locations).Error; err != nil {
			return nil, fmt.Errorf("查詢 PartLocations 失敗: %w", err)
		}
		// 按 PartID 重組映射，方便對應到對應 revision 的 rawRevisionData
		partToRev := make(map[int64]int64, len(parts))
		for _, p := range parts {
			partToRev[p.ID] = p.RevisionID
		}
		for _, loc := range locations {
			if revID, ok := partToRev[loc.PartID]; ok {
				if data, ok := result[revID]; ok {
					data.partLocations = append(data.partLocations, loc)
				}
			}
		}
	}

	// 3. 批量查詢 SecondSources
	var secondSources []db.SecondSource
	if err := s.db.Where("revision_id IN ?", revisionIDs).Find(&secondSources).Error; err != nil {
		return nil, fmt.Errorf("查詢 SecondSources 失敗: %w", err)
	}
	for _, ss := range secondSources {
		if data, ok := result[ss.RevisionID]; ok {
			data.secondSources = append(data.secondSources, ss)
		}
	}

	// 4. 批量查詢 MatrixModels
	var models []db.MatrixModel
	if err := s.db.Where("revision_id IN ?", revisionIDs).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("查詢 MatrixModels 失敗: %w", err)
	}
	for _, m := range models {
		if data, ok := result[m.RevisionID]; ok {
			data.models = append(data.models, m)
		}
	}

	// 5. 批量查詢 MatrixSelections
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
//
// 參數：
//   - requestedIDs：期望的 BOM Revision ID 排序列表
//   - rawData：從資料庫載入的原始資料映射
//
// 回傳：
//   - []ViewRevision：已組裝的 revision 元資料列表
func buildViewRevisions(requestedIDs []int64, rawData map[int64]*rawRevisionData) []ViewRevision {
	revisions := make([]ViewRevision, 0, len(rawData))
	processed := make(map[int64]bool, len(rawData))

	// 1. 依 requestedIDs 指定的順序建立 ViewRevision
	for _, id := range requestedIDs {
		data, ok := rawData[id]
		if !ok || processed[id] {
			continue
		}
		revisions = append(revisions, createViewRevisionFromRaw(id, data))
		processed[id] = true
	}

	// 2. 針對未包含在 requestedIDs 中的 Revision，按 ID 升冪補齊
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
//
// 參數：
//   - id：BOM Revision ID
//   - data：從資料庫載入的原始 revision 資料
//
// 回傳：
//   - ViewRevision：組裝完成的 revision 元資料
func createViewRevisionFromRaw(id int64, data *rawRevisionData) ViewRevision {
	rev := data.revision

	// 建立 ModelNames 列表（排序）與 ModelQty 映射
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

// groupKey 建立物料群組的識別鍵，格式為 "supplier|supplier_pn" 或 "supplier|supplier_pn|type"
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
// 整合演算法步驟：
//  1. 依 Mode 判斷有效的 bom_status 條件（預設 bom_status=I，自動擴充 NPI->P, MP->M），先排除非標的物料。
//  2. 依 query.RevisionIDs 順序遍歷 Revision：
//     - Revision 1（及首次出現的主料）：建立 Group 初始基礎（Item, Description, Remark 等）。
//     - 若同 Revision 中有相同 (Supplier, SupplierPN) 但不同 Type 或 CCL 的主料，合併 Location（去重）並重新計算 Qty。
//     - 2nd Source 依序組裝至 Group 中。
//     - 後續 Revision：若主料已存在，增補 RevisionID 並檢查 2nd Source（未存在則新增，已存在則追加 RevisionID）；若主料不存在，依首個 Revision 原則新增。
//  3. 蒐集所有 revision × model 的 MatrixSelection。
//
// 參數：
//   - rawData：從資料庫載入的原始資料映射
//   - query：查詢參數（用於取得 revision 排序與 ModeOverride）
//
// 回傳：
//   - []ViewPartGroup：聯集合併後的物料群組列表
func (s *Service) mergeRevisions(rawData map[int64]*rawRevisionData, query ViewQuery) []ViewPartGroup {
	// partGroupMap: groupKey → ViewPartGroup（使用指標方便累加）
	type partGroupBuilder struct {
		group     ViewPartGroup
		ssBuilder map[string]*ViewSecondSource // ss groupKey → ViewSecondSource
	}

	builders := make(map[string]*partGroupBuilder) // 主料 groupKey → builder
	// 維持插入順序的鍵列表（讓輸出順序可預期）
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

	// 按 query.RevisionIDs 的順序遍歷，確保「第一份 revision」的語意固定
	for _, revID := range query.RevisionIDs {
		data, ok := rawData[revID]
		if !ok {
			continue
		}

		// 建立 partID -> Part 的快速映射
		partByID := make(map[int64]db.Part, len(data.parts))
		for _, p := range data.parts {
			partByID[p.ID] = p
		}

		// --- 1. 按 Part 收集所有 PartLocation（包含 I, X, P, M） ---
		allLocsByPartID := make(map[int64][]db.PartLocation)
		for _, loc := range data.partLocations {
			allLocsByPartID[loc.PartID] = append(allLocsByPartID[loc.PartID], loc)
		}

		// 追蹤是否有 MatrixSelection 的 partID 或 groupKey
		hasSelectionByPartID := make(map[int64]bool)
		hasSelectionByGroupKey := make(map[string]bool)
		for _, sel := range data.selections {
			if sel.PartID != 0 {
				hasSelectionByPartID[sel.PartID] = true
			}
			if sel.Group != "" {
				hasSelectionByGroupKey[sel.Group] = true
			}
		}

		// 預先追蹤每個 part 是否有至少一個「有效」location（bom_status != X）。
		// 此資訊用於判斷主料是否真正「存在」於此 revision（用於 SourceRevisionIDs 標記），
		// 以確保 BigMatrix 匯出灰色判斷的正確性（bom_status=X 的物料不算存在於此 revision）。
		hasEffectiveLocByPartID := make(map[int64]bool, len(data.parts))
		for partID, locs := range allLocsByPartID {
			for _, loc := range locs {
				if strings.ToUpper(strings.TrimSpace(loc.BomStatus)) != "X" {
					hasEffectiveLocByPartID[partID] = true
					break
				}
			}
		}

		// 找出在此 revision 中含有「有效」location（bom_status != X）或含有 MatrixSelection 的 Parts。
		validParts := make([]db.Part, 0, len(data.parts))
		for _, p := range data.parts {
			key := groupKey(p.Supplier, p.SupplierPN)
			if hasEffectiveLocByPartID[p.ID] || hasSelectionByPartID[p.ID] || hasSelectionByGroupKey[key] {
				validParts = append(validParts, p)
			}
		}

		// --- 2. 處理此 revision 主料對應的 SecondSources ---
		ssByMainKey := make(map[string][]db.SecondSource) // 主料 groupKey → []SecondSource
		for _, ss := range data.secondSources {
			mainPart, exists := partByID[ss.PartID]
			if !exists {
				continue
			}
			// 只有主料在此 revision 有效上件或含有 Selection，才採納其替代料
			if hasEffectiveLocByPartID[mainPart.ID] || hasSelectionByPartID[mainPart.ID] {
				key := groupKey(mainPart.Supplier, mainPart.SupplierPN)
				ssByMainKey[key] = append(ssByMainKey[key], ss)
			}
		}

		// --- 3. 建立此 revision 的 MatrixSelection 映射 (優先以 SortOrder 順序匹配) ---
		selByGroupKeyByOrder := make(map[string]map[int]string)
		selByGroupKeyByName := make(map[string]map[string]string)
		selMaterialByGroupKeyByOrder := make(map[string]map[int]string)
		selMaterialByGroupKeyByName := make(map[string]map[string]string)

		for _, sel := range data.selections {
			var mainKey string
			if mainPart, exists := partByID[sel.PartID]; exists {
				mainKey = groupKey(mainPart.Supplier, mainPart.SupplierPN)
			} else if sel.Group != "" {
				mainKey = sel.Group
			} else {
				continue
			}

			sortOrder, okOrder := modelIDToSortOrder[sel.ModelID]
			modelName, okName := modelIDToName[sel.ModelID]
			if !okOrder && !okName {
				continue
			}

			matKey := sel.Material
			if matKey == "" && sel.SelectedSupplierPn != "" {
				if sel.SelectedSupplier != "" {
					matKey = groupKey(sel.SelectedSupplier, sel.SelectedSupplierPn)
				} else {
					matKey = sel.SelectedSupplierPn
				}
			}

			if okOrder {
				if selByGroupKeyByOrder[mainKey] == nil {
					selByGroupKeyByOrder[mainKey] = make(map[int]string)
				}
				selByGroupKeyByOrder[mainKey][sortOrder] = sel.SelectedSupplierPn

				if selMaterialByGroupKeyByOrder[mainKey] == nil {
					selMaterialByGroupKeyByOrder[mainKey] = make(map[int]string)
				}
				selMaterialByGroupKeyByOrder[mainKey][sortOrder] = matKey
			}

			if okName && modelName != "" {
				if selByGroupKeyByName[mainKey] == nil {
					selByGroupKeyByName[mainKey] = make(map[string]string)
				}
				selByGroupKeyByName[mainKey][modelName] = sel.SelectedSupplierPn

				if selMaterialByGroupKeyByName[mainKey] == nil {
					selMaterialByGroupKeyByName[mainKey] = make(map[string]string)
				}
				selMaterialByGroupKeyByName[mainKey][modelName] = matKey
			}
		}

		// --- 4. 處理此 revision 的 Parts與 PartLocations（按 (Supplier, SupplierPN, Location.Type) 歸類） ---
		representativeParts := make(map[string]db.Part)
		typeByGroup := make(map[string]string)
		locationsByGroup := make(map[string]map[string]bool)
		cclByGroup := make(map[string]bool)
		statusByGroup := make(map[string]string)
		validLocCountByGroup := make(map[string]int)
		pCountByGroup := make(map[string]int)
		mCountByGroup := make(map[string]int)

		for _, p := range validParts {
			locs := allLocsByPartID[p.ID]
			hasEffectiveLoc := false

			for _, loc := range locs {
				statusUpper := strings.ToUpper(strings.TrimSpace(loc.BomStatus))
				if statusUpper == "X" {
					continue // 跳過不上件 location，不計入位置聚合與 BOMStatus 統計
				}
				hasEffectiveLoc = true
				key := groupKey(p.Supplier, p.SupplierPN, loc.Type)
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
				}
				if _, exists := representativeParts[key]; !exists {
					representativeParts[key] = p
				}
			}

			// 若物料沒有任何有效上件 location（如只有 X 或僅有 selection），使用無 type 鍵歸類
			if !hasEffectiveLoc {
				baseKey := groupKey(p.Supplier, p.SupplierPN)
				if _, exists := representativeParts[baseKey]; !exists {
					representativeParts[baseKey] = p
				}
			}
		}

		// 計算此 revision 中各群組的 BOMStatus：
		// 規則：只有當群組內所有有效 location 全為 P 時才為 P，全為 M 時才為 M，否則歸為 I
		for key := range representativeParts {
			totalLocs := validLocCountByGroup[key]
			if totalLocs > 0 && pCountByGroup[key] == totalLocs {
				statusByGroup[key] = "P"
			} else if totalLocs > 0 && mCountByGroup[key] == totalLocs {
				statusByGroup[key] = "M"
			} else {
				statusByGroup[key] = "I"
			}
		}

		// --- 5. 將此 revision 的群組合併至 builders ---
		for key, repPart := range representativeParts {
			b, exists := builders[key]
			if !exists {
				// 主料不存在於 group 中：首次出現此群組
				locs := sortedLocations(locationsByGroup[key])
				bomStat := statusByGroup[key]
				if bomStat == "" {
					bomStat = "I"
				}
				partType := typeByGroup[key]
				b = &partGroupBuilder{
					group: ViewPartGroup{
						MainSupplier:          repPart.Supplier,
						MainSupplierPN:        repPart.SupplierPN,
						Item:                  repPart.Item,
						HHPN:                  repPart.HHPN,
						Description:           repPart.Description,
						Type:                  partType, // SMD, PTH, BOTTOM
						BOMStatus:             bomStat,
						CCL:                   cclByGroup[key],
						Remark:                repPart.Remark,
						Qty:                   len(locationsByGroup[key]),
						Locations:             locs,
						SourceRevisionIDs:     []int64{revID},
						MainSelectionsByOrder: make(map[int]bool),
					},
					ssBuilder: make(map[string]*ViewSecondSource),
				}
				builders[key] = b
				keyOrder = append(keyOrder, key)
			} else {
				// 主料已存在於 group 中：追加 SourceRevisionID
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
				// 跨 Revision 合併：若當前 Revision 該群組狀態與已存在的狀態不一致，退回 "I"
				if b.group.BOMStatus != statusByGroup[key] {
					b.group.BOMStatus = "I"
				}
			}

			baseKey := groupKey(repPart.Supplier, repPart.SupplierPN)
			// 更新主料於此 revision 中的 Selection 狀態
			if selMatOrderMap := selMaterialByGroupKeyByOrder[baseKey]; selMatOrderMap != nil {
				for sortOrder, selectedMat := range selMatOrderMap {
					if strings.EqualFold(selectedMat, baseKey) || strings.EqualFold(selectedMat, key) {
						b.group.MainSelectionsByOrder[sortOrder] = true
					}
				}
			}

			// 檢察與組裝此 revision 屬於該主料的 2nd Source (替代料)
			for _, ss := range ssByMainKey[baseKey] {
				ssKey := groupKey(ss.Supplier, ss.SupplierPN)
				existing, ok := b.ssBuilder[ssKey]
				if !ok {
					existing = &ViewSecondSource{
						HHPN:              ss.HHPN,
						Supplier:          ss.Supplier,
						SupplierPN:        ss.SupplierPN,
						Description:       ss.Description,
						Remark:            ss.Remark,
						SourceRevisionIDs: []int64{revID},
						SelectionsByOrder: make(map[int]bool),
					}
					b.ssBuilder[ssKey] = existing
				} else {
					existing.SourceRevisionIDs = appendUnique(existing.SourceRevisionIDs, revID)
					if existing.SelectionsByOrder == nil {
						existing.SelectionsByOrder = make(map[int]bool)
					}
				}

				if selMatOrderMap := selMaterialByGroupKeyByOrder[baseKey]; selMatOrderMap != nil {
					for sortOrder, selectedMat := range selMatOrderMap {
						if strings.EqualFold(selectedMat, ssKey) {
							existing.SelectionsByOrder[sortOrder] = true
						}
					}
				}
			}

			// 蒐集此 revision 的 MatrixSelections (優先以 SortOrder 順序匹配)
			selOrderMap := selByGroupKeyByOrder[baseKey]
			selNameMap := selByGroupKeyByName[baseKey]
			selMatOrderMap := selMaterialByGroupKeyByOrder[baseKey]
			selMatNameMap := selMaterialByGroupKeyByName[baseKey]

			for _, mData := range rawData[revID].models {
				sortOrder := mData.SortOrder
				var selectedPN string
				var selectedMaterial string
				var selectedSupplier string

				if selOrderMap != nil {
					selectedPN = selOrderMap[sortOrder]
				}
				if selectedPN == "" && selNameMap != nil && mData.ModelName != "" {
					selectedPN = selNameMap[mData.ModelName]
				}

				if selMatOrderMap != nil {
					selectedMaterial = selMatOrderMap[sortOrder]
				}
				if selectedMaterial == "" && selMatNameMap != nil && mData.ModelName != "" {
					selectedMaterial = selMatNameMap[mData.ModelName]
				}

				if selectedMaterial != "" {
					parts := strings.SplitN(selectedMaterial, "|", 2)
					if len(parts) == 2 {
						selectedSupplier = parts[0]
					}
				}

				b.group.Selections = append(b.group.Selections, ViewModelSelection{
					RevisionID:       revID,
					SortOrder:        sortOrder,
					ModelName:        mData.ModelName,
					ModelQty:         mData.Qty,
					SelectedPN:       selectedPN,
					SelectedSupplier: selectedSupplier,
					SelectedMaterial: selectedMaterial,
				})
			}
		}
	}

	// 從 builders 組裝最終的 ViewPartGroup 列表（依 keyOrder 維持順序）
	result := make([]ViewPartGroup, 0, len(builders))
	for _, key := range keyOrder {
		b := builders[key]

		// 組裝 SecondSources（按 supplier+pn 排序）
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

		result = append(result, b.group)
	}

	// 依物料群組的聚合後 Location 欄位排序（以群組為單位，主料+2nd sources 一起移動）。
	// 採自然排序（Natural Sort）正確處理 C1 < C2 < C10 的字母數字混合格式。
	sort.SliceStable(result, func(i, j int) bool {
		return compareLocStr(result[i].Locations, result[j].Locations)
	})

	return result
}

// sortedLocations 將 location set 轉換為排序後的逗號分隔字串
//
// 參數：
//   - locSet：location 的集合（map[string]bool）
//
// 回傳：
//   - string：逗號分隔的位置編號字串
func sortedLocations(locSet map[string]bool) string {
	locs := make([]string, 0, len(locSet))
	for loc := range locSet {
		locs = append(locs, loc)
	}
	sort.Strings(locs)
	return strings.Join(locs, ",")
}

// appendUnique 將 id 附加至 ids slice（若不重複）
//
// 參數：
//   - ids：現有的 ID 列表
//   - id：要追加的 ID
//
// 回傳：
//   - []int64：附加後的 ID 列表
func appendUnique(ids []int64, id int64) []int64 {
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
}

// compareLocStr 比較兩個逗號分隔的 Location 字串大小，用於物料群組排序。
//
// 排序邏輯：
//  1. 取各字串的第一個 token（第一個位置編號）作為排序鍵。
//  2. 對每個 token 套用自然排序（naturalSortKey），正確處理字母數字混合
//     的位置格式（例如 C1 < C2 < C10，而非字典序的 C1 < C10 < C2）。
//  3. 空字串排在最後。
//
// 參數：
//   - a, b：逗號分隔的 Location 字串（例如 "C1,C2,R3" 或 "R10"）
//
// 回傳：
//   - bool：若 a 應排在 b 之前則回傳 true
func compareLocStr(a, b string) bool {
	// 空字串排在最後
	if a == "" && b == "" {
		return false
	}
	if a == "" {
		return false
	}
	if b == "" {
		return true
	}

	// 取第一個 token（第一個逗號前的位置編號）作為主要排序鍵
	firstA := a
	if idx := strings.IndexByte(a, ','); idx >= 0 {
		firstA = a[:idx]
	}
	firstB := b
	if idx := strings.IndexByte(b, ','); idx >= 0 {
		firstB = b[:idx]
	}

	// 套用自然排序鍵比較
	keysA := naturalSortKey(firstA)
	keysB := naturalSortKey(firstB)

	for i := 0; i < len(keysA) && i < len(keysB); i++ {
		if keysA[i] != keysB[i] {
			return keysA[i] < keysB[i]
		}
	}
	return len(keysA) < len(keysB)
}

// naturalSortKey 將位置字串拆解為字母與數字交替的 token 片段，
// 用於自然排序比較（例如 "C10" → ["C", "000000010"]）。
//
// 數字部分以零填充至固定寬度（9位），確保字典序與數值序一致。
//
// 參數：
//   - s：位置編號字串（例如 "C1"、"R10"、"U5A"）
//
// 回傳：
//   - []string：字母與數字交替的 token 片段列表
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
			// 類型切換：儲存當前 token，開始新 token
			if isDigit {
				// 數字 token：零填充至 9 位確保字典序與數值序一致
				tokens = append(tokens, fmt.Sprintf("%09s", string(cur)))
			} else {
				tokens = append(tokens, string(cur))
			}
			cur = cur[:0]
			isDigit = curIsDigit
		}
		cur = append(cur, c)
	}

	// 儲存最後一個 token
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
// 依據 (MainSupplier, MainSupplierPN) 進行二階物料合併。
//
// 適用場景：
//   - BigMatrix 匯出：單一工作表呈現全體物料，同一物料跨製程（如 SMD + BOTTOM）需合併為單一 Main Source。
//
// 合併規則：
//   - 群組識別鍵：(MainSupplier, MainSupplierPN)
//   - Item / HHPN / Description / Remark：保留首個非空資訊
//   - Type：保留首個非空 Type（例如 "SMD"）
//   - Locations：將所有被合併群組的 locations 拆分、去重、自然排序後重新組合為逗號分隔字串
//   - Qty：重新計算為合併後 locations 的數量
//   - CCL：若任一群組為 CCL=true，則合併後為 true
//   - BOMStatus：若所有被合併群組的 BOMStatus 均為 "P" 則為 "P"，全為 "M" 則為 "M"，否則歸為 "I"
//   - SourceRevisionIDs：取所有群組 SourceRevisionIDs 的聯集去重
//   - SecondSources：依 (Supplier, SupplierPN) 取聯集去重，合併 SourceRevisionIDs 與 SelectionsByOrder
//   - Selections：合併各 Revision 各 Model 的選擇狀態
//   - MainSelectionsByOrder：合併主料各 Model 的勾選狀態（OR 運算）
//   - 輸出排序：依合併後的 Locations 自然排序
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
		matKey := groupKey(g.MainSupplier, g.MainSupplierPN)

		b, exists := builders[matKey]
		if !exists {
			locSet := make(map[string]bool)
			for _, loc := range atomizeLocations(g.Locations) {
				locSet[loc] = true
			}

			ssMap := make(map[string]*ViewSecondSource, len(g.SecondSources))
			for _, ss := range g.SecondSources {
				ssKey := groupKey(ss.Supplier, ss.SupplierPN)
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
					MainSupplier:          g.MainSupplier,
					MainSupplierPN:        g.MainSupplierPN,
					Item:                  g.Item,
					HHPN:                  g.HHPN,
					Description:           g.Description,
					Type:                  g.Type,
					BOMStatus:             g.BOMStatus,
					CCL:                   g.CCL,
					Remark:                g.Remark,
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
			// 1. 基本資訊補充（若既有為空則補齊）
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

			// 2. Locations 聯集
			for _, loc := range atomizeLocations(g.Locations) {
				b.locSet[loc] = true
			}

			// 3. CCL 聯集（OR）
			if g.CCL {
				b.group.CCL = true
			}

			// 4. BOMStatus 判斷（全 P 則 P，全 M 則 M，否則 I）
			statusUpper := strings.ToUpper(strings.TrimSpace(g.BOMStatus))
			if statusUpper != "P" {
				b.allProto = false
			}
			if statusUpper != "M" {
				b.allMP = false
			}

			// 5. SourceRevisionIDs 聯集
			for _, revID := range g.SourceRevisionIDs {
				b.group.SourceRevisionIDs = appendUnique(b.group.SourceRevisionIDs, revID)
			}

			// 6. MainSelectionsByOrder 合併（OR）
			for order, isSel := range g.MainSelectionsByOrder {
				if isSel {
					b.group.MainSelectionsByOrder[order] = true
				}
			}

			// 7. SecondSources 聯集
			for _, ss := range g.SecondSources {
				ssKey := groupKey(ss.Supplier, ss.SupplierPN)
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
					for order, isSel := range ss.SelectionsByOrder {
						if isSel {
							existingSS.SelectionsByOrder[order] = true
						}
					}
				}
			}

			// 8. Selections 合併
			for _, sel := range g.Selections {
				selKey := fmt.Sprintf("%d_%d", sel.RevisionID, sel.SortOrder)
				existingSel, ok := b.selectionsMap[selKey]
				if !ok || existingSel.SelectedPN == "" {
					b.selectionsMap[selKey] = sel
				}
			}
		}
	}

	// 組合最終結果
	result := make([]ViewPartGroup, 0, len(builders))
	for _, matKey := range keyOrder {
		b := builders[matKey]

		// 重新計算 Locations 與 Qty
		b.group.Locations = sortedLocations(b.locSet)
		b.group.Qty = len(b.locSet)

		// 決定最終 BOMStatus
		if b.allProto {
			b.group.BOMStatus = "P"
		} else if b.allMP {
			b.group.BOMStatus = "M"
		} else {
			b.group.BOMStatus = "I"
		}

		// 組合 SecondSources（排序）
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

		// 組合 Selections（依 RevisionID, SortOrder 排序）
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

	// 依 Location 自然排序
	sort.SliceStable(result, func(i, j int) bool {
		return compareLocStr(result[i].Locations, result[j].Locations)
	})

	return result
}

// atomizeLocations 將逗號分隔的 location 字串拆分為獨立的 location 陣列（若尚未在當前 package 宣告）
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

