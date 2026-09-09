package excel

import (
	"fmt"
	"strings"
	"testing"

	"bomix-app/backend/db"
	"bomix-app/backend/logger"
	"bomix-app/backend/types"
	"bomix-app/backend/view"

	"github.com/xuri/excelize/v2"
)

// TestInvestigateBigMatrixIssue 回歸測試：
// 驗證當物料在同一個 BOM Revision 中既充當主料 (Role="M")，又在其他群組中充當替代料 (Role="S") 時，
// Matrix 與 BigMatrix 匯入流程不會因替代料覆蓋主料 Component 映射，而導致 selection 勾選遺失。
//
// 具體測試情境：
// 1. 匯入 EBOM (TARIS 0.4)
// 2. 匯入 Matrix (TARIS 0.4)
// 3. 匯入 BigMatrix (FY27-D6_BigMatrix_SI1_0.4-Deviation_20260904.xlsx)
// 4. 匯出 BigMatrix 並驗證 @U4, LQ1, SQ12/SQ13, XU201,XU801 等位置的勾選完整保留
func TestInvestigateBigMatrixIssue(t *testing.T) {
	lg := logger.NewLogger(100) // 必須 > 0
	gdb := setupMatrixTestDB(t)
	reader := NewReader(gdb, nil)

	ebomPath := "testdata/TARIS_EZBOM_SI1_0.4_BOM_20260827_1100.WP(compared).xls"
	matrixPath := "testdata/TARIS_EZBOM_SI1_0.4_MatrixBOM_20260827_1100.WP.xlsx"
	bigmatrixPath := "testdata/FY27-D6_BigMatrix_SI1_0.4-Deviation_20260904.xlsx"

	t.Log("=== Step 1: 匯入 EBOM ===")
	resEBOM, err := reader.ImportExcel([]string{ebomPath})
	if err != nil {
		t.Fatalf("EBOM 匯入失敗: %v", err)
	}
	if len(resEBOM) == 0 || len(resEBOM[0].Errors) > 0 {
		t.Fatalf("EBOM 匯入含有錯誤: %+v", resEBOM)
	}

	t.Log("=== Step 2: 匯入 Matrix ===")
	resMatrix, err := reader.ImportExcel([]string{matrixPath})
	if err != nil {
		t.Fatalf("Matrix 匯入失敗: %v", err)
	}
	if len(resMatrix) == 0 || len(resMatrix[0].Errors) > 0 {
		t.Fatalf("Matrix 匯入含有錯誤: %+v", resMatrix)
	}

	t.Log("=== Step 3: 匯入 BigMatrix ===")
	resBM, err := reader.ImportExcel([]string{bigmatrixPath})
	if err != nil {
		t.Logf("BigMatrix 匯入警告: %v", err)
	}
	if len(resBM) == 0 {
		t.Fatalf("BigMatrix 匯入無結果")
	}

	// 檢查資料庫中這四個關鍵物料的 Selection 筆數皆應大於 0
	// part_locations 表中每個 location 為獨立原子記錄，使用單一 location 代表進行比對
	targetLocs := []string{"@U4", "LQ1", "SQ12", "XU201"}
	for _, loc := range targetLocs {
		var count int64
		err := gdb.Raw(`
			SELECT COUNT(s.id)
			FROM matrix_selections s
			JOIN revision_components c ON s.component_id = c.id
			JOIN part_locations l ON l.component_id = c.id
			WHERE s.revision_id = 1 AND l.location = ?
		`, loc).Scan(&count).Error
		if err != nil {
			t.Fatalf("查詢位置 %s 的 Selections 失敗: %v", loc, err)
		}
		if count == 0 {
			t.Errorf("位置 %s 在 BigMatrix 匯入後 Selections 筆數期望 > 0，實際為 0", loc)
		}
	}

	t.Log("=== Step 4: 匯出 BigMatrix ===")
	var revs []db.BomRevision
	gdb.Find(&revs)
	revIDs := make([]int64, len(revs))
	for i, r := range revs {
		revIDs[i] = r.ID
	}

	viewSvc := view.NewService(gdb, lg)
	viewRes, err := viewSvc.Query(view.ViewQuery{
		RevisionIDs: revIDs,
		ViewType:    view.ViewCCL,
	})
	if err != nil {
		t.Fatalf("View query failed: %v", err)
	}

	mergedPGs := view.MergePartGroupsByMaterial(viewRes.PartGroups)

	outDir := t.TempDir()
	exportPath := outDir + "/Re-exported_BigMatrix.xlsx"
	writer, err := NewWriter(lg)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	var revDataList []RevisionData
	for _, vr := range viewRes.Revisions {
		revDataList = append(revDataList, RevisionData{
			ID:          fmt.Sprintf("%d", vr.ID),
			ProjectCode: vr.ProjectCode,
			Phase:       vr.Phase,
			Version:     vr.Version,
			Date:        vr.Date,
			ModelNames:  vr.ModelNames,
			ModelQty:    vr.ModelQty,
		})
	}

	var partDataList []PartData
	for idx, pg := range mergedPGs {
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

		var ssList []SecondSourceData
		for _, ss := range pg.SecondSources {
			ssList = append(ssList, SecondSourceData{
				HHPN:              ss.HHPN,
				Description:       ss.Description,
				Supplier:          ss.Supplier,
				SupplierPn:        ss.SupplierPN,
				Notes:             ss.Notes,
				SourceRevisionIDs: ss.SourceRevisionIDs,
			})
		}

		partDataList = append(partDataList, PartData{
			Item:                        fmt.Sprintf("%d", idx+1),
			HHPN:                        pg.HHPN,
			Description:                 pg.Description,
			Supplier:                    pg.MainSupplier,
			SupplierPn:                  pg.MainSupplierPN,
			Qty:                         pg.Qty,
			Location:                    pg.Locations,
			BOMStatus:                   pg.BOMStatus,
			Notes:                       pg.Notes,
			SourceRevisionIDs:           pg.SourceRevisionIDs,
			SecondSources:               ssList,
			Selections:                  selections,
			SelectionsByOrder:           selectionsByOrder,
			SelectionsByRevAndOrder:     selectionsByRevAndOrder,
			SelectionsByRevAndName:      selectionsByRevAndName,
			SelectionsByMaterialByOrder: selectionsByMaterialByOrder,
			SelectionsByRevAndMaterial:  selectionsByRevAndMaterial,
		})
	}

	expOpts := ExportOptions{
		Format:     types.FormatBigMatrix,
		OutputPath: exportPath,
		Revisions:  revDataList,
		PartData:   partDataList,
	}
	_, err = writer.ExportExcel(expOpts)
	if err != nil {
		t.Fatalf("匯出 BigMatrix 失敗: %v", err)
	}

	t.Log("=== Step 5: 檢查匯出的 Excel 是否包含勾選 ===")
	fExp, err := excelize.OpenFile(exportPath)
	if err != nil {
		t.Fatalf("開啟匯出檔失敗: %v", err)
	}
	defer fExp.Close()

	expRows, _ := fExp.GetRows("BigMatrix")
	foundTargets := make(map[string]bool)
	for rIdx, r := range expRows {
		rowStr := strings.Join(r, " | ")
		for _, loc := range targetLocs {
			if strings.Contains(rowStr, loc) {
				foundTargets[loc] = true
				// 驗證此列或此群組是否有勾選 "V"
				hasCheck := false
				for c := 7; c < len(r); c++ {
					if strings.EqualFold(strings.TrimSpace(r[c]), "V") {
						hasCheck = true
						break
					}
				}
				if !hasCheck {
					t.Errorf("匯出檔 Row %d (位置 %s) 期望包含 'V' 勾選，但未找到: %v", rIdx+1, loc, r)
				}
			}
		}
	}

	for _, loc := range targetLocs {
		if !foundTargets[loc] {
			t.Errorf("匯出檔中未找到包含位置 %s 的物料列", loc)
		}
	}
}

// TestTARIS_EBOM_NIView_U4 驗證真實 EBOM 匯入後，在 NI view 下能正確撈取到 @U4 (Item 5 FOXCONN)，在 ALL view 下能撈取到 @U4 (Item 6 NAXIN)
func TestTARIS_EBOM_NIView_U4(t *testing.T) {
	lg := logger.NewLogger(100)
	gdb := setupMatrixTestDB(t)
	reader := NewReader(gdb, nil)

	ebomPath := "testdata/TARIS_EZBOM_SI1_0.4_BOM_20260827_1100.WP(compared).xls"
	_, err := reader.ImportExcel([]string{ebomPath})
	if err != nil {
		t.Fatalf("EBOM 匯入失敗: %v", err)
	}

	viewSvc := view.NewService(gdb, lg)

	// 1. 驗證 NI View
	niRes, err := viewSvc.Query(view.ViewQuery{
		RevisionIDs: []int64{1},
		ViewType:    view.ViewNI,
	})
	if err != nil {
		t.Fatalf("查詢 ViewNI 失敗: %v", err)
	}

	if len(niRes.PartGroups) == 0 {
		t.Fatalf("ViewNI 回傳 0 筆物料群組，期望包含 NI 工作表的所有物料")
	}

	var foundNIU4 *view.ViewPartGroup
	for i := range niRes.PartGroups {
		pg := &niRes.PartGroups[i]
		if strings.Contains(pg.Locations, "@U4") && pg.MainSupplierPN == "PHCX01G11012" {
			foundNIU4 = pg
			break
		}
	}

	if foundNIU4 == nil {
		t.Fatalf("ViewNI 中未找到 @U4 (FOXCONN / PHCX01G11012) 物料")
	}

	if foundNIU4.BOMStatus != "X" {
		t.Errorf("ViewNI @U4 BOMStatus 期望為 'X'，實際為 %s", foundNIU4.BOMStatus)
	}
	if foundNIU4.Qty != 1 {
		t.Errorf("ViewNI @U4 Qty 期望為 1，實際為 %d", foundNIU4.Qty)
	}
	if len(foundNIU4.SecondSources) < 2 {
		t.Errorf("ViewNI @U4 期望具有至少 2 個替代料 (AURAS, NAXIN)，實際為 %d 個", len(foundNIU4.SecondSources))
	} else {
		hasNaxin := false
		for _, ss := range foundNIU4.SecondSources {
			if ss.SupplierPN == "101068500" {
				hasNaxin = true
				break
			}
		}
		if !hasNaxin {
			t.Errorf("ViewNI @U4 替代料中未找到 NAXIN (101068500): %+v", foundNIU4.SecondSources)
		}
	}

	// 2. 驗證 ALL View
	allRes, err := viewSvc.Query(view.ViewQuery{
		RevisionIDs: []int64{1},
		ViewType:    view.ViewAll,
	})
	if err != nil {
		t.Fatalf("查詢 ViewAll 失敗: %v", err)
	}

	var foundAllU4 *view.ViewPartGroup
	for i := range allRes.PartGroups {
		pg := &allRes.PartGroups[i]
		if strings.Contains(pg.Locations, "@U4") && pg.MainSupplierPN == "101068500" {
			foundAllU4 = pg
			break
		}
	}

	if foundAllU4 == nil {
		t.Fatalf("ViewAll 中未找到 @U4 (NAXIN / 101068500) 物料")
	}
	if foundAllU4.BOMStatus != "I" {
		t.Errorf("ViewAll @U4 BOMStatus 期望為 'I'，實際為 %s", foundAllU4.BOMStatus)
	}
	if foundAllU4.Type != "PTH" {
		t.Errorf("ViewAll @U4 Type 期望為 'PTH'，實際為 %s", foundAllU4.Type)
	}
}





