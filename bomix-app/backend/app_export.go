package backend

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"bomix-app/backend/db"
	"bomix-app/backend/excel"
	"bomix-app/backend/logger"
	"bomix-app/backend/task"
	"bomix-app/backend/types"
	"bomix-app/backend/view"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ==================== Excel 匯出工作流 (Excel Export Workflow) ====================

// ExportExcel 依指定格式將資料庫資料匯出為 Excel 檔案
//
// 支援格式：
//   - Matrix: 針對每個選取的 Revision 分別獨立匯出一個 Matrix 檔案
//   - BigMatrix: 將多個選取的 Revision 橫向展開合併於單一 BigMatrix 檔案
//
// 參數：
//   - options: 匯出選項參數（包含格式、選取的 RevisionIDs、覆蓋 Model 數量、輸出目錄等）
//
// 回傳：
//   - []string: 建立的背景 Task ID 列表
//   - error: 若未開啟系列資料庫則回傳錯誤
func (a *App) ExportExcel(options *ExportOptions) ([]string, error) {
	formatStr := strings.TrimSpace(options.Format)
	var bomFormat types.BOMFormat
	if strings.EqualFold(formatStr, string(types.FormatBigMatrix)) {
		bomFormat = types.FormatBigMatrix
	} else if strings.EqualFold(formatStr, string(types.FormatMatrix)) {
		bomFormat = types.FormatMatrix
	} else {
		bomFormat = types.BOMFormat(formatStr)
	}

	a.logger.Info(fmt.Sprintf("[ExportExcel] 開始進行 Excel 匯出作業 (Format: %s, 選取 Revisions 數量: %d)", bomFormat, len(options.RevisionIDs)))
	a.logger.Debug(fmt.Sprintf("[ExportExcel] 匯出詳細參數: RevisionIDs=%v, ModelCountOverrides=%+v, OutputDir=%s", options.RevisionIDs, options.ModelCountOverrides, options.OutputDir))

	a.mu.RLock()
	dbConn := a.db
	a.mu.RUnlock()

	if dbConn == nil {
		a.logger.Error("[ExportExcel] 失敗: 未開啟 Series 資料庫")
		return nil, fmt.Errorf("no series is currently open")
	}

	// Update the series LastExportPath
	if options.OutputDir != "" {
		if err := dbConn.Model(&db.Series{}).Where("id = ?", 1).Update("last_export_path", options.OutputDir).Error; err != nil {
			a.logger.Warn(fmt.Sprintf("Failed to update last_export_path: %v", err))
		}
	}

	// 矩陣 (Matrix) 格式：每一個 selected BOM Revision 需各自匯出為獨立的 Matrix 檔案與任務
	if bomFormat == types.FormatMatrix {
		var taskIDs []string
		for _, revIDVal := range options.RevisionIDs {
			revIDStr := fmt.Sprintf("%d", revIDVal)

			// 嘗試讀取 Revision 基本資訊，建立具備專案/Phase/Version識別度的 Task 名稱
			var taskName string
			var revRecord db.BomRevision
			if err := dbConn.First(&revRecord, revIDVal).Error; err == nil {
				var projRecord db.Project
				projCode := "BOMIX"
				if err := dbConn.First(&projRecord, revRecord.ProjectID).Error; err == nil && projRecord.Code != "" {
					projCode = projRecord.Code
				}
				taskName = fmt.Sprintf("Export: Matrix (%s %s-%s)", projCode, revRecord.Phase, revRecord.Version)
			} else {
				taskName = fmt.Sprintf("Export: Matrix (ID: %d)", revIDVal)
			}

			// 多個 Revision 匯出時，若指定了 OutputPath 則取其目錄為 OutputDir
			outDir := options.OutputDir
			outPath := options.OutputPath
			if len(options.RevisionIDs) > 1 && outDir == "" && outPath != "" {
				outDir = filepath.Dir(outPath)
				outPath = ""
			}

			tID := a.taskMgr.Submit(
				taskName,
				"Export",
				func(ctx context.Context, progress func(float64, string), taskLogger *logger.Logger) error {
					progress(0.1, fmt.Sprintf("Preparing export data for Revision %d...", revIDVal))

					// 僅載入此單一 Revision 的 DB View 資料（Matrix 匯出：保留 Type 維度）
					revisions, parts, err := loadExportData(taskLogger, dbConn, []int64{revIDVal}, false)
					if err != nil {
						taskLogger.Warn(fmt.Sprintf("[ExportExcel] 載入 Revision %d 的 View 資料失敗/警告: %v", revIDVal, err))
					} else {
						taskLogger.Info(fmt.Sprintf("[ExportExcel] 成功載入 Revision %d 的 DB 資料: Parts=%d", revIDVal, len(parts)))
					}

					exportOptions := excel.ExportOptions{
						Format:              bomFormat,
						ProjectIDs:          options.ProjectIDs,
						RevisionIDs:         []string{revIDStr},
						Description:         options.Description,
						OutputPath:          outPath,
						OutputDir:           outDir,
						ModelCountOverrides: options.ModelCountOverrides,
						Revisions:           revisions,
						PartData:            parts,
					}

					excelWriter, err := excel.NewWriter(taskLogger)
					if err != nil {
						taskLogger.Error(fmt.Sprintf("[ExportExcel] 建立 Excel Writer 失敗: %v", err))
						return fmt.Errorf("failed to create excel writer: %w", err)
					}

					outputPaths, err := excelWriter.ExportExcel(exportOptions)
					if err != nil {
						if errors.Is(err, excel.ErrInvalidOutputPath) || strings.Contains(err.Error(), "invalid export output path") {
							if taskLogger != nil {
								taskLogger.Warn(fmt.Sprintf("[ExportExcel] 匯出路徑無效或無法寫入: %v", err))
							}
							return task.NewWarningError(fmt.Errorf("無效的匯出路徑: %w", err))
						}
						return fmt.Errorf("failed to export: %w", err)
					}

					if len(outputPaths) > 0 && taskLogger != nil {
						taskLogger.Info(fmt.Sprintf("[ExportExcel] 成功匯出 Matrix 檔案: %s", filepath.Base(outputPaths[0])))
					}

					progress(1.0, fmt.Sprintf("Exported %d files", len(outputPaths)))
					return nil
				},
			)
			taskIDs = append(taskIDs, tID)
		}
		return taskIDs, nil
	}

	// BigMatrix 格式：所有選取的 Revisions 橫向合併於單一 BigMatrix 檔案中
	revisionIDsStr := make([]string, len(options.RevisionIDs))
	for i, id := range options.RevisionIDs {
		revisionIDsStr[i] = fmt.Sprintf("%d", id)
	}

	// Export in a task
	taskID := uuid.New().String()
	taskID = a.taskMgr.Submit(
		fmt.Sprintf("Export: %s", options.Format),
		"Export",
		func(ctx context.Context, progress func(float64, string), taskLogger *logger.Logger) error {
			progress(0.1, "Preparing export data...")

			// Load revisions and part data from DB via View System using taskLogger
			// BigMatrix 格式：需進一步去除 Type 維度，依 (Supplier, SupplierPN) 合併物料
			revisions, parts, err := loadExportData(taskLogger, dbConn, options.RevisionIDs, true)
			if err != nil {
				taskLogger.Warn(fmt.Sprintf("[ExportExcel] 從資料庫載入 View 資料失敗/警告: %v", err))
			} else {
				taskLogger.Info(fmt.Sprintf("[ExportExcel] 成功透過 View 系統載入 DB 資料: Revisions=%d, Parts=%d", len(revisions), len(parts)))
			}

			exportOptions := excel.ExportOptions{
				Format:              bomFormat,
				ProjectIDs:          options.ProjectIDs,
				RevisionIDs:         revisionIDsStr,
				Description:         options.Description,
				OutputPath:          options.OutputPath,
				OutputDir:           options.OutputDir,
				ModelCountOverrides: options.ModelCountOverrides,
				Revisions:           revisions,
				PartData:            parts,
			}

			// Create Excel writer with taskLogger
			excelWriter, err := excel.NewWriter(taskLogger)
			if err != nil {
				taskLogger.Error(fmt.Sprintf("[ExportExcel] 建立 Excel Writer 失敗: %v", err))
				return fmt.Errorf("failed to create excel writer: %w", err)
			}

			// Export to Excel
			outputPaths, err := excelWriter.ExportExcel(exportOptions)
			if err != nil {
				if errors.Is(err, excel.ErrInvalidOutputPath) || strings.Contains(err.Error(), "invalid export output path") {
					if taskLogger != nil {
						taskLogger.Warn(fmt.Sprintf("[ExportExcel] 匯出路徑無效或無法寫入: %v", err))
					}
					return task.NewWarningError(fmt.Errorf("無效的匯出路徑: %w", err))
				}
				return fmt.Errorf("failed to export: %w", err)
			}

			progress(1.0, fmt.Sprintf("Exported %d files", len(outputPaths)))
			return nil
		},
	)

	return []string{taskID}, nil
}

// loadExportData 透過 View 系統從資料庫讀取匯出所需的 Revisions 與 Parts 資料。
//
// 此函數透過 View 系統的 Query() 取得資料，
// 依據 product-spec 8.1.6 規定使用 ViewCCL 視圖條件過濾：
// 包含 CCL=true (依模式過濾 bom_status=I + P/M) 或在當次匯出之 revisions 中有任何 matrix selection 勾選之物料群組。
//
// 匯出時使用「整合聯集」視圖（多 revision 時取聯集），
// ViewPartGroup 中的 SourceRevisionIDs 會被傳遞至 PartData，
// 供 BigMatrix Writer 判斷哪些儲存格需要填灰色底色。
//
// 參數：
//   - lg: Logger 實例
//   - dbConn: GORM 資料庫連線
//   - revisionIDs: 要匯出的 BOM Revision ID 列表
//   - groupByMaterial: 是否進一步去除 Type 維度，依 (Supplier, SupplierPN) 合併物料（BigMatrix 匯出為 true，Matrix 匯出為 false）
//
// 回傳：
//   - []excel.RevisionData: revision 元資料列表
//   - []excel.PartData: 物料資料列表（包含 SourceRevisionIDs）
//   - error: 若查詢失敗則回傳錯誤
func loadExportData(lg *logger.Logger, dbConn *gorm.DB, revisionIDs []int64, groupByMaterial bool) ([]excel.RevisionData, []excel.PartData, error) {
	if len(revisionIDs) == 0 {
		return nil, nil, nil
	}

	query := view.ViewQuery{
		RevisionIDs: revisionIDs,
		ViewType:    view.ViewCCL, // BigMatrix/Matrix 匯出依 product-spec 8.1.6 需使用 CCL 視圖過濾 (CCL=Y 或有 Matrix Selection 勾選)
	}

	if lg != nil {
		lg.Info(fmt.Sprintf("[loadExportData] 建立 View 條件: RevisionIDs=%v, ViewType=%s, GroupByMaterial=%v",
			query.RevisionIDs, query.ViewType, groupByMaterial))
	}

	svc := view.NewService(dbConn, lg)
	viewResult, err := svc.Query(query)
	if err != nil {
		return nil, nil, fmt.Errorf("view query failed: %w", err)
	}

	// 若啟用 groupByMaterial (如 BigMatrix 匯出)，進一步去除 Type 維度，依 (Supplier, SupplierPN) 進行二階物料合併
	partGroups := viewResult.PartGroups
	if groupByMaterial {
		partGroups = view.MergePartGroupsByMaterial(viewResult.PartGroups)
		if lg != nil {
			lg.Info(fmt.Sprintf("[loadExportData] 套用二階物料合併 (去除 Type 維度): 原始群組數=%d, 合併後物料數=%d",
				len(viewResult.PartGroups), len(partGroups)))
		}
	}

	// 將 ViewRevision 轉換為 excel.RevisionData
	revDataList := make([]excel.RevisionData, 0, len(viewResult.Revisions))
	for _, vr := range viewResult.Revisions {
		revDataList = append(revDataList, excel.RevisionData{
			ID:               fmt.Sprintf("%d", vr.ID),
			ProjectCode:      vr.ProjectCode,
			Description:      vr.Description,
			SchematicVersion: vr.SchematicVersion,
			PCBVersion:       vr.PCBVersion,
			PCAPN:            vr.PCAPN,
			Phase:            vr.Phase,
			Version:          vr.Version,
			Date:             vr.Date,
			SourceFile:       vr.SourceFile,
			ModelNames:       vr.ModelNames,
			ModelQty:         vr.ModelQty,
			ModelQtyByOrder:  vr.ModelQtyByOrder,
		})
	}

	// 將 ViewPartGroup 轉換為 excel.PartData
	// 物料群組已由 View 系統聚合完畢（locations 已合併、qty 已計算）
	partDataList := make([]excel.PartData, 0, len(partGroups))
	for idx, pg := range partGroups {
		// 整合此群組的 Model 勾選狀態：
		// 1. 單一 Revision 相容: map[modelName]selectedPN 與 map[sortOrder]selectedPN
		// 2. BigMatrix 多 Revision 精確: map[revIDStr]map[sortOrder]selectedPN 與 map[revIDStr]map[modelName]selectedPN
		selections := make(map[string]string)
		selectionsByOrder := make(map[int]string)
		selectionsByRevAndOrder := make(map[string]map[int]string)
		selectionsByRevAndName := make(map[string]map[string]string)
		selectionsByMaterialByOrder := make(map[int]string)
		selectionsByRevAndMaterial := make(map[string]map[int]string)

		for _, sel := range pg.Selections {
			if sel.SelectedPN != "" {
				revIDStr := fmt.Sprintf("%d", sel.RevisionID)
				selectedMat := sel.SelectedMaterial
				if selectedMat == "" {
					if sel.SelectedSupplier != "" {
						selectedMat = fmt.Sprintf("%s|%s", strings.TrimSpace(sel.SelectedSupplier), strings.TrimSpace(sel.SelectedPN))
					} else {
						selectedMat = strings.TrimSpace(sel.SelectedPN)
					}
				}

				// 單一 Revision 相容
				if _, exists := selections[sel.ModelName]; !exists {
					selections[sel.ModelName] = sel.SelectedPN
				}
				if _, exists := selectionsByOrder[sel.SortOrder]; !exists {
					selectionsByOrder[sel.SortOrder] = sel.SelectedPN
				}
				if selectedMat != "" {
					if _, exists := selectionsByMaterialByOrder[sel.SortOrder]; !exists {
						selectionsByMaterialByOrder[sel.SortOrder] = selectedMat
					}
				}

				// 多 Revision 精確映射
				if selectionsByRevAndOrder[revIDStr] == nil {
					selectionsByRevAndOrder[revIDStr] = make(map[int]string)
				}
				selectionsByRevAndOrder[revIDStr][sel.SortOrder] = sel.SelectedPN

				if selectionsByRevAndName[revIDStr] == nil {
					selectionsByRevAndName[revIDStr] = make(map[string]string)
				}
				selectionsByRevAndName[revIDStr][sel.ModelName] = sel.SelectedPN

				if selectedMat != "" {
					if selectionsByRevAndMaterial[revIDStr] == nil {
						selectionsByRevAndMaterial[revIDStr] = make(map[int]string)
					}
					selectionsByRevAndMaterial[revIDStr][sel.SortOrder] = selectedMat
				}
			}
		}

		// 整合 SecondSources
		ssData := make([]excel.SecondSourceData, 0, len(pg.SecondSources))
		for _, ss := range pg.SecondSources {
			ssData = append(ssData, excel.SecondSourceData{
				HHPN:              ss.HHPN,
				Supplier:          ss.Supplier,
				SupplierPn:        ss.SupplierPN,
				Description:       ss.Description,
				Remark:            ss.Remark,
				Notes:             ss.Notes,
				SourceRevisionIDs: ss.SourceRevisionIDs,
				SelectionsByOrder: ss.SelectionsByOrder,
			})
		}

		partDataList = append(partDataList, excel.PartData{
			Item:                         fmt.Sprintf("%d", idx+1), // 流水號
			HHPN:                         pg.HHPN,
			Description:                  pg.Description,
			Supplier:                     pg.MainSupplier,
			SupplierPn:                   pg.MainSupplierPN,
			Qty:                          pg.Qty,
			Location:                     pg.Locations,
			Type:                         pg.Type,
			BOMStatus:                    pg.BOMStatus,
			CCL:                          pg.CCL,
			Remark:                       pg.Remark,
			Notes:                        pg.Notes,
			SecondSources:                ssData,
			Selections:                   selections,
			SelectionsByOrder:            selectionsByOrder,
			SelectionsByRevAndOrder:      selectionsByRevAndOrder,
			SelectionsByRevAndName:       selectionsByRevAndName,
			SelectionsByMaterialByOrder:  selectionsByMaterialByOrder,
			SelectionsByRevAndMaterial:   selectionsByRevAndMaterial,
			MainSelectionsByOrder:        pg.MainSelectionsByOrder,
			SourceRevisionIDs:            pg.SourceRevisionIDs, // 傳遞來源歸屬，供 BigMatrix 填灰色
		})
	}

	return revDataList, partDataList, nil
}
