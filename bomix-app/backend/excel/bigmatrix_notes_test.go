package excel

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"bomix-app/backend/db"
	"bomix-app/backend/types"
	"github.com/xuri/excelize/v2"
)

// TestIsNotesColumn 驗證 isNotesColumn 判斷函式的正確性：
// 1. Row 2="Notes" 且 Row 3~5 皆為空時，認定為 true
// 2. Row 2 不為 "Notes"（如專案代號）時，認定為 false
// 3. Row 3 有值（如 Phase-Version）時，認定為 false
// 4. Row 5 有值（如 Qty）時，認定為 false
func TestIsNotesColumn(t *testing.T) {
	f := excelize.NewFile()
	sheet := "Sheet1"

	// 欄 H (7): 正確的 Notes 欄位
	f.SetCellValue(sheet, "H2", "Notes")
	f.SetCellValue(sheet, "H3", "")
	f.SetCellValue(sheet, "H4", "")
	f.SetCellValue(sheet, "H5", "")

	// 欄 I (8): BOM Revision 第一欄
	f.SetCellValue(sheet, "I2", "PROJ_A")
	f.SetCellValue(sheet, "I3", "EVT-0.1")
	f.SetCellValue(sheet, "I4", "A")
	f.SetCellValue(sheet, "I5", 10)

	// 欄 J (9): Row 2 為 Notes 但 Row 3 有值
	f.SetCellValue(sheet, "J2", "Notes")
	f.SetCellValue(sheet, "J3", "SomeValue")

	// 欄 K (10): 空白欄位
	f.SetCellValue(sheet, "K2", "")

	// 欄 L (11): 模擬 Excelize 合併儲存格讀取情境（Row 2~5 皆回傳 Master Cell 內容 "Notes"）
	f.SetCellValue(sheet, "L2", "Notes")
	f.SetCellValue(sheet, "L3", "Notes")
	f.SetCellValue(sheet, "L4", "Notes")
	f.SetCellValue(sheet, "L5", "Notes")

	wb := &ExcelizeWorkbook{f: f}

	if !isNotesColumn(wb, sheet, 7) {
		t.Errorf("欄 H 應被識別為 Notes 欄位")
	}
	if isNotesColumn(wb, sheet, 8) {
		t.Errorf("欄 I 不應被識別為 Notes 欄位")
	}
	if isNotesColumn(wb, sheet, 9) {
		t.Errorf("欄 J 不應被識別為 Notes 欄位（Row 3 非空）")
	}
	if isNotesColumn(wb, sheet, 10) {
		t.Errorf("欄 K 不應被識別為 Notes 欄位（Row 2 為空）")
	}
	if !isNotesColumn(wb, sheet, 11) {
		t.Errorf("欄 L 應被識別為 Notes 欄位（合併儲存格情境）")
	}
}

// TestBigMatrixExportAndImport_Notes 端到端測試：
// 1. 驗證 BigMatrix 匯出時最末欄為 Notes 欄位
// 2. 驗證 Row 2~5 合併且標題為 "Notes"
// 3. 驗證 Row 6+ 包含主料與替代料的 notes
// 4. 驗證 BigMatrix 匯入時自動辨識該欄並更新 Material 表的 notes
func TestBigMatrixExportAndImport_Notes(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_bm_notes.db")

	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("開啟資料庫失敗: %v", err)
	}
	defer db.Close(database)

	if err := db.AutoMigrate(database); err != nil {
		t.Fatalf("AutoMigrate 失敗: %v", err)
	}

	// 1. 建立測試專案與 BOM Revision
	series, err := db.CreateSeries(database, "SeriesA", "Test Series")
	if err != nil {
		t.Fatalf("建立 Series 失敗: %v", err)
	}

	project, err := db.GetOrCreateProject(database, series.ID, "PROJ1", "Test Project")
	if err != nil {
		t.Fatalf("建立 Project 失敗: %v", err)
	}

	revision, err := db.CreateRevision(database, project.ID, "EVT", "0.1", "Initial Revision")
	if err != nil {
		t.Fatalf("建立 BomRevision 失敗: %v", err)
	}

	// 2. 建立測試物料（初始 notes 均為舊值或空）
	materials := []db.Material{
		{
			Supplier:    "Murata",
			SupplierPN:  "GRM155R71C104KA88D",
			HHPN:        "HH-001",
			Description: "Capacitor 0.1uF",
			Remark:      "Main Remark",
			Notes:       "Old Main Note",
		},
		{
			Supplier:    "Samsung",
			SupplierPN:  "CL05B104KO5NNNC",
			HHPN:        "HH-002",
			Description: "Capacitor 0.1uF 2nd",
			Remark:      "SS Remark",
			Notes:       "Old SS Note",
		},
	}
	_, _, err = db.UpsertMaterials(database, materials, nil)
	if err != nil {
		t.Fatalf("UpsertMaterials 失敗: %v", err)
	}

	matMurata, _ := db.GetMaterialBySupplierPN(database, "Murata", "GRM155R71C104KA88D")
	matSamsung, _ := db.GetMaterialBySupplierPN(database, "Samsung", "CL05B104KO5NNNC")

	// 建立主料與替代料 Component
	mainComp := db.RevisionComponent{
		RevisionID: revision.ID,
		MaterialID: matMurata.ID,
		Role:       "M",
		Item:       "1",
	}
	if err := database.Create(&mainComp).Error; err != nil {
		t.Fatalf("建立主料 Component 失敗: %v", err)
	}

	secondComp := db.RevisionComponent{
		RevisionID:        revision.ID,
		MaterialID:        matSamsung.ID,
		Role:              "S",
		ParentComponentID: mainComp.ID,
	}
	if err := database.Create(&secondComp).Error; err != nil {
		t.Fatalf("建立替代料 Component 失敗: %v", err)
	}

	// 3. 準備 BigMatrix 匯出資料
	exportPath := filepath.Join(tmpDir, "exported_bigmatrix.xlsx")
	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("建立 Writer 失敗: %v", err)
	}

	options := ExportOptions{
		Format:      types.FormatBigMatrix,
		RevisionIDs: []string{fmt.Sprintf("%d", revision.ID)},
		OutputPath:  exportPath,
	}

	revisionsData := []RevisionData{
		{
			ID:          fmt.Sprintf("%d", revision.ID),
			ProjectCode: "PROJ1",
			Phase:       "EVT",
			Version:     "0.1",
			ModelNames:  []string{"A", "B"},
			ModelQty:    map[string]int{"A": 5, "B": 10},
		},
	}

	partData := []PartData{
		{
			Item:        "1",
			HHPN:        matMurata.HHPN,
			Description: matMurata.Description,
			Supplier:    matMurata.Supplier,
			SupplierPn:  matMurata.SupplierPN,
			Qty:         1,
			Location:    "C1",
			Type:        "SMD",
			BOMStatus:   "I",
			CCL:         true,
			Remark:      matMurata.Remark,
			Notes:       "Exported Note from Murata",
			SecondSources: []SecondSourceData{
				{
					HHPN:        matSamsung.HHPN,
					Supplier:    matSamsung.Supplier,
					SupplierPn:  matSamsung.SupplierPN,
					Description: matSamsung.Description,
					Remark:      matSamsung.Remark,
					Notes:       "Exported Note from Samsung 2nd",
				},
			},
			SourceRevisionIDs: []int64{revision.ID},
		},
	}

	exportedPaths, err := writer.exportBigMatrixDetailed(options, revisionsData, partData)
	if err != nil {
		t.Fatalf("匯出 BigMatrix 失敗: %v", err)
	}
	if len(exportedPaths) == 0 {
		t.Fatalf("未回傳匯出路徑")
	}

	// 4. 驗證匯出的 Excel 檔案格式與 Notes 欄位
	f, err := excelize.OpenFile(exportPath)
	if err != nil {
		t.Fatalf("開啟匯出的 BigMatrix 失敗: %v", err)
	}
	defer f.Close()

	// 驗證 Model 欄位：Revision 1 有 2 個 Model（欄 H=7, I=8）
	// Notes 欄位應為第 3 欄，即欄 J (索引 9)
	notesColName := "J"
	titleVal, err := f.GetCellValue("BigMatrix", notesColName+"2")
	if err != nil {
		t.Fatalf("讀取 J2 失敗: %v", err)
	}
	if titleVal != "Notes" {
		t.Errorf("預期 %s2 標題為 'Notes'，實際為 '%s'", notesColName, titleVal)
	}

	// 驗證 Row 3~5 為空值或回傳合併儲存格 Master 值 "Notes"
	for r := 3; r <= 5; r++ {
		val, _ := f.GetCellValue("BigMatrix", fmt.Sprintf("%s%d", notesColName, r))
		if val != "" && !strings.EqualFold(val, "notes") {
			t.Errorf("預期 %s%d 為空字串或 'Notes'，實際為 '%s'", notesColName, r, val)
		}
	}

	// 驗證 Row 2~5 是否已合併
	mergeCells, err := f.GetMergeCells("BigMatrix")
	if err != nil {
		t.Fatalf("GetMergeCells 失敗: %v", err)
	}
	hasNotesMerge := false
	for _, mc := range mergeCells {
		if strings.EqualFold(mc.GetStartAxis(), notesColName+"2") && strings.EqualFold(mc.GetEndAxis(), notesColName+"5") {
			hasNotesMerge = true
			break
		}
	}
	if !hasNotesMerge {
		t.Errorf("未找到 %s2:%s5 的儲存格合併設定", notesColName, notesColName)
	}

	// 驗證 Row 6 (主料列) 之 Notes 內容
	row6Notes, err := f.GetCellValue("BigMatrix", notesColName+"6")
	if err != nil {
		t.Fatalf("讀取 Row 6 Notes 失敗: %v", err)
	}
	if row6Notes != "Exported Note from Murata" {
		t.Errorf("Row 6 Notes 預期 'Exported Note from Murata'，實際為 '%s'", row6Notes)
	}

	// 驗證 Row 7 (替代料列) 之 Notes 內容
	row7Notes, err := f.GetCellValue("BigMatrix", notesColName+"7")
	if err != nil {
		t.Fatalf("讀取 Row 7 Notes 失敗: %v", err)
	}
	if row7Notes != "Exported Note from Samsung 2nd" {
		t.Errorf("Row 7 Notes 預期 'Exported Note from Samsung 2nd'，實際為 '%s'", row7Notes)
	}

	// 驗證 Row 1 無公式
	row1Formula, _ := f.GetCellFormula("BigMatrix", notesColName+"1")
	if row1Formula != "" {
		t.Errorf("預期 %s1 無公式，實際包含公式 '%s'", notesColName, row1Formula)
	}

	// 驗證 Notes 欄位樣式是否繼承該 row 左側格式（字體、斑馬紋底色、框線、Locked: false）
	for _, r := range []int{6, 7} {
		notesCell := fmt.Sprintf("%s%d", notesColName, r)
		leftCell := fmt.Sprintf("I%d", r) // Revision 1 最後一個 Model 欄位為 I

		notesSid, err := f.GetCellStyle("BigMatrix", notesCell)
		if err != nil {
			t.Fatalf("取得 %s 樣式失敗: %v", notesCell, err)
		}
		leftSid, err := f.GetCellStyle("BigMatrix", leftCell)
		if err != nil {
			t.Fatalf("取得 %s 樣式失敗: %v", leftCell, err)
		}

		notesStyle, err := f.GetStyle(notesSid)
		if err != nil {
			t.Fatalf("解析 %s 樣式定義失敗: %v", notesCell, err)
		}
		leftStyle, err := f.GetStyle(leftSid)
		if err != nil {
			t.Fatalf("解析 %s 樣式定義失敗: %v", leftCell, err)
		}

		// 1. 驗證儲存格是否已解鎖 (Protection.Locked == false)
		if notesStyle.Protection == nil || notesStyle.Protection.Locked {
			t.Errorf("預期 %s 儲存格為解鎖 (Locked: false)，實際為鎖定", notesCell)
		}

		// 2. 驗證字體樣式（字體名稱、大小）是否繼承
		if leftStyle.Font != nil && notesStyle.Font != nil {
			if notesStyle.Font.Size != leftStyle.Font.Size || notesStyle.Font.Family != leftStyle.Font.Family {
				t.Errorf("Row %d Notes 字體未繼承左側: 預期 %v，實際 %v", r, leftStyle.Font, notesStyle.Font)
			}
		}

		// 3. 驗證斑馬紋底色（Fill Pattern & Color）是否繼承
		if leftStyle.Fill.Type != notesStyle.Fill.Type || leftStyle.Fill.Pattern != notesStyle.Fill.Pattern {
			t.Errorf("Row %d Notes 底色類型未繼承左側", r)
		}
		if len(leftStyle.Fill.Color) > 0 && len(notesStyle.Fill.Color) > 0 {
			if leftStyle.Fill.Color[0] != notesStyle.Fill.Color[0] {
				t.Errorf("Row %d Notes 底色未繼承左側: 預期 %v，實際 %v", r, leftStyle.Fill.Color, notesStyle.Fill.Color)
			}
		}

		// 4. 驗證框線設定（Border）
		if len(notesStyle.Border) == 0 && len(leftStyle.Border) > 0 {
			t.Errorf("Row %d Notes 缺少框線設定", r)
		}

		// 5. 驗證對齊方式：文字靠左對齊 (Horizontal == "left")，自動換行 (WrapText == true)
		if notesStyle.Alignment == nil {
			t.Fatalf("Row %d Notes 缺少 Alignment 設定", r)
		}
		if notesStyle.Alignment.Horizontal != "left" {
			t.Errorf("Row %d Notes 預期靠左對齊 (Horizontal: 'left')，實際為 '%s'", r, notesStyle.Alignment.Horizontal)
		}
		if !notesStyle.Alignment.WrapText {
			t.Errorf("Row %d Notes 預期自動換行 (WrapText: true)，實際為 false", r)
		}
	}

	// 驗證 Notes 欄寬為 500 像素（換算約 70.71 字元單位）
	colWidth, err := f.GetColWidth("BigMatrix", notesColName)
	if err != nil {
		t.Fatalf("取得 %s 欄寬失敗: %v", notesColName, err)
	}
	if colWidth < 70.0 || colWidth > 71.0 {
		t.Errorf("Notes 欄寬預期約 70.71 (500 像素)，實際為 %f", colWidth)
	}

	// 5. 測試修改 Excel 中的 Notes 後進行 BigMatrix 匯入
	_ = f.SetCellValue("BigMatrix", notesColName+"6", "Imported New Note for Murata")
	_ = f.SetCellValue("BigMatrix", notesColName+"7", "Imported New Note for Samsung")
	if err := f.Save(); err != nil {
		t.Fatalf("儲存修改後的 Excel 失敗: %v", err)
	}

	// 執行匯入
	wb := &ExcelizeWorkbook{f: f}
	bmReader := &BigMatrixReader{
		db:     database,
		result: &types.ImportResult{},
	}
	if err := bmReader.Import(wb); err != nil {
		t.Fatalf("BigMatrix 匯入失敗: %v", err)
	}

	// 6. 驗證資料庫中的 Material.notes 是否成功更新
	murataAfter, err := db.GetMaterialBySupplierPN(database, "Murata", "GRM155R71C104KA88D")
	if err != nil {
		t.Fatalf("查詢更新後的 Murata 失敗: %v", err)
	}
	if murataAfter.Notes != "Imported New Note for Murata" {
		t.Errorf("Murata Notes 匯入後更新錯誤，預期 'Imported New Note for Murata'，實際 '%s'", murataAfter.Notes)
	}
	if murataAfter.Remark != "Main Remark" {
		t.Errorf("Murata Remark 遭意外覆寫: %s", murataAfter.Remark)
	}

	samsungAfter, err := db.GetMaterialBySupplierPN(database, "Samsung", "CL05B104KO5NNNC")
	if err != nil {
		t.Fatalf("查詢更新後的 Samsung 失敗: %v", err)
	}
	if samsungAfter.Notes != "Imported New Note for Samsung" {
		t.Errorf("Samsung Notes 匯入後更新錯誤，預期 'Imported New Note for Samsung'，實際 '%s'", samsungAfter.Notes)
	}
	if samsungAfter.Remark != "SS Remark" {
		t.Errorf("Samsung Remark 遭意外覆寫: %s", samsungAfter.Remark)
	}

	// 7. 驗證 ImportResult
	if bmReader.result.MaterialsUpdated != 2 {
		t.Errorf("預期 MaterialsUpdated 為 2，實際為 %d", bmReader.result.MaterialsUpdated)
	}
}
