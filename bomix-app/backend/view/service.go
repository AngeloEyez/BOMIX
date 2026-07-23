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
		s.logger.Info(fmt.Sprintf("[ViewService.Query] 執行 View 查詢: RevisionIDs=%v, ViewType=%s, ModeOverride=%s",
			query.RevisionIDs, query.ViewType, query.ModeOverride))
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

	// 建立 ViewRevision 元資料列表
	revisions := buildViewRevisions(rawData)

	// 執行多 revision 聯集合併，建立 ViewPartGroup 列表
	partGroups := s.mergeRevisions(rawData, query)

	// 套用視圖過濾
	filter := NewFilter()
	partGroups = filter.Apply(partGroups, query, rawData)

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
	for _, p := range parts {
		if data, ok := result[p.RevisionID]; ok {
			data.parts = append(data.parts, p)
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
// 輸出列表的順序與 rawData 的迭代順序一致（Go map 順序不固定，
// 但對消費者而言 revision 的前後順序不影響功能正確性）。
//
// 參數：
//   - rawData：從資料庫載入的原始資料映射
//
// 回傳：
//   - []ViewRevision：已組裝的 revision 元資料列表
func buildViewRevisions(rawData map[int64]*rawRevisionData) []ViewRevision {
	revisions := make([]ViewRevision, 0, len(rawData))

	// 按 ID 排序以確保輸出順序一致
	ids := make([]int64, 0, len(rawData))
	for id := range rawData {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, id := range ids {
		data := rawData[id]
		rev := data.revision

		// 建立 ModelNames 列表（排序）與 ModelQty 映射
		modelNames := make([]string, 0, len(data.models))
		modelQty := make(map[string]int, len(data.models))
		for _, m := range data.models {
			modelNames = append(modelNames, m.ModelName)
			modelQty[m.ModelName] = m.Qty
		}
		sort.Strings(modelNames)

		revisions = append(revisions, ViewRevision{
			ID:               id,
			ProjectCode:      data.project.Code,
			Phase:            rev.Phase,
			Version:          rev.Version,
			Description:      rev.Description,
			SchematicVersion: rev.SchematicVersion,
			PCBVersion:       rev.PCBVersion,
			PCAPN:            rev.PCAPN,
			Date:             rev.Date,
			Mode:             rev.Mode,
			ModelNames:       modelNames,
			ModelQty:         modelQty,
		})
	}
	return revisions
}

// groupKey 建立物料群組的識別鍵，格式為 "supplier|supplier_pn"
func groupKey(supplier, supplierPN string) string {
	return supplier + "|" + supplierPN
}

// mergeRevisions 執行多 BOM Revision 的主料與替代料聯集合併，
// 並蒐集 Model 勾選狀態，建立 ViewPartGroup 列表。
//
// 整合演算法步驟：
//  1. 對每個 revision 的 parts，依 (supplier, supplier_pn) 建立群組
//  2. 同一群組鍵出現在多個 revision → 合併為一個 ViewPartGroup，
//     物料屬性取自第一個出現的 revision
//  3. 各 revision 的 SecondSource 以 (supplier, supplier_pn) 去重後取聯集
//  4. 蒐集所有 revision × model 的 MatrixSelection
//
// 注意：此方法不進行視圖過濾，過濾由 Filter.Apply() 負責。
//
// 參數：
//   - rawData：從資料庫載入的原始資料映射
//   - query：查詢參數（用於取得 revision 排序以確保「第一份」的定義一致）
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

	// 建立 model ID → ModelName 的全局映射（跨所有 revision）
	modelIDToName := make(map[int64]string)
	modelIDToQty := make(map[int64]int)
	for _, data := range rawData {
		for _, m := range data.models {
			modelIDToName[m.ID] = m.ModelName
			modelIDToQty[m.ID] = m.Qty
		}
	}

	// 按 query.RevisionIDs 的順序遍歷，確保「第一份 revision」的語意固定
	for _, revID := range query.RevisionIDs {
		data, ok := rawData[revID]
		if !ok {
			continue
		}

		// --- 建立此 revision 的 partID → Part 映射 ---
		partByID := make(map[int64]db.Part, len(data.parts))
		for _, p := range data.parts {
			partByID[p.ID] = p
		}

		// --- 處理此 revision 的 SecondSources ---
		// secondSources 依其關聯的主料 partID → 群組鍵映射
		// db.SecondSource.PartID 為所關聯的主料 Part 的 ID
		ssByMainKey := make(map[string][]db.SecondSource) // 主料 groupKey → []SecondSource
		for _, ss := range data.secondSources {
			mainPart, exists := partByID[ss.PartID]
			if !exists {
				continue
			}
			key := groupKey(mainPart.Supplier, mainPart.SupplierPN)
			ssByMainKey[key] = append(ssByMainKey[key], ss)
		}

		// --- 建立此 revision 的 MatrixSelection 映射 ---
		// selByPartKey: 主料 groupKey → model 名稱 → 被選中的 SupplierPN
		selByGroupKey := make(map[string]map[string]string)
		for _, sel := range data.selections {
			// MatrixSelection.Group 儲存 "main_supplier|main_supplier_pn"
			// 若 Group 欄位非空，直接使用；否則從 PartID 查找
			mainKey := sel.Group
			if mainKey == "" {
				// fallback：嘗試從 PartID 查找主料
				mainPart, exists := partByID[sel.PartID]
				if !exists {
					continue
				}
				mainKey = groupKey(mainPart.Supplier, mainPart.SupplierPN)
			}

			modelName, ok := modelIDToName[sel.ModelID]
			if !ok {
				continue
			}

			if selByGroupKey[mainKey] == nil {
				selByGroupKey[mainKey] = make(map[string]string)
			}
			selByGroupKey[mainKey][modelName] = sel.SelectedSupplierPn
		}

		// --- 遍歷此 revision 的 Parts，合併至 builders ---
		// 先找出每個群組的「代表 Part」（location 非空的最小 ID 的 part）
		// 群組鍵 → 代表 Part
		representativeParts := make(map[string]db.Part)
		// 群組鍵 → 此 revision 下此群組的所有 locations（去重）
		locationsByGroup := make(map[string]map[string]bool)

		for _, p := range data.parts {
			key := groupKey(p.Supplier, p.SupplierPN)
			if locationsByGroup[key] == nil {
				locationsByGroup[key] = make(map[string]bool)
			}
			if p.Location != "" {
				locationsByGroup[key][p.Location] = true
			}
			// 以第一個（ID 最小）有位置的 part 作為代表
			if _, exists := representativeParts[key]; !exists {
				representativeParts[key] = p
			}
		}

		// 將此 revision 的群組合併至 builders
		for key, repPart := range representativeParts {
			if _, exists := builders[key]; !exists {
				// 首次出現此群組：建立新 builder
				locs := sortedLocations(locationsByGroup[key])
				builders[key] = &partGroupBuilder{
					group: ViewPartGroup{
						MainSupplier:   repPart.Supplier,
						MainSupplierPN: repPart.SupplierPN,
						// db.Part 目前尚未有獨立 Item/HHPN 欄位
						// Item 由下游消費者（通常是流水號）產生
						// HHPN 暫時使用 SupplierPN（與現有 loadExportData 行為一致）
						Item:              "",
						HHPN:              repPart.SupplierPN,
						Description:       repPart.Description,
						Type:              repPart.Type, // SMD, PTH, BOTTOM
						BOMStatus:         repPart.BOMStatus,
						CCL:               repPart.CCL,
						Remark:            repPart.Remark,
						Qty:               len(locationsByGroup[key]),
						Locations:         locs,
						SourceRevisionIDs: []int64{revID},
					},
					ssBuilder: make(map[string]*ViewSecondSource),
				}
				keyOrder = append(keyOrder, key)
			} else {
				// 此群組已存在：僅追加 SourceRevisionID
				b := builders[key]
				b.group.SourceRevisionIDs = appendUnique(b.group.SourceRevisionIDs, revID)
			}

			// 合併此 revision 的 SecondSources
			b := builders[key]
			for _, ss := range ssByMainKey[key] {
				ssKey := groupKey(ss.Supplier, ss.SupplierPN)
				if existing, ok := b.ssBuilder[ssKey]; ok {
					// 已存在此替代料：追加 SourceRevisionID
					existing.SourceRevisionIDs = appendUnique(existing.SourceRevisionIDs, revID)
				} else {
					// 首次出現此替代料
					b.ssBuilder[ssKey] = &ViewSecondSource{
						Supplier:          ss.Supplier,
						SupplierPN:        ss.SupplierPN,
						Description:       ss.Description,
						SourceRevisionIDs: []int64{revID},
					}
				}
			}

			// 蒐集此 revision 的 MatrixSelections
			if selMap, ok := selByGroupKey[key]; ok {
				for _, data := range rawData[revID].models {
					modelName := data.ModelName
					selectedPN := selMap[modelName] // 若未勾選則為空字串
					b.group.Selections = append(b.group.Selections, ViewModelSelection{
						RevisionID: revID,
						ModelName:  modelName,
						ModelQty:   data.Qty,
						SelectedPN: selectedPN,
					})
				}
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
