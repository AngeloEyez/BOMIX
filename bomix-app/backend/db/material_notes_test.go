package db

import (
	"path/filepath"
	"testing"
)

// TestUpdateMaterialNotes 驗證 UpdateMaterialNotes 批次更新 Material.notes 的行為：
// 1. 既有物料的 notes 成功更新
// 2. 其餘欄位 (HHPN, Description, Remark 等) 保持不變
// 3. 不存在的物料被忽略，不執行新增
// 4. 允許將 notes 清空為空字串
func TestUpdateMaterialNotes(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_material_notes.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("開啟資料庫失敗: %v", err)
	}
	defer Close(database)

	if err := AutoMigrate(database); err != nil {
		t.Fatalf("AutoMigrate 失敗: %v", err)
	}

	// 1. 建立初始物料
	initialMats := []Material{
		{
			Supplier:    "Murata",
			SupplierPN:  "GRM155R71C104KA88D",
			HHPN:        "HH-001",
			Description: "Capacitor 0.1uF",
			Remark:      "Original Remark",
			Notes:       "Old Note 1",
		},
		{
			Supplier:    "Samsung",
			SupplierPN:  "CL05B104KO5NNNC",
			HHPN:        "HH-002",
			Description: "Capacitor 0.1uF Samsung",
			Remark:      "Samsung Remark",
			Notes:       "Old Note 2",
		},
		{
			Supplier:    "Yageo",
			SupplierPN:  "CC0402KRX7R9BB104",
			HHPN:        "HH-003",
			Description: "Capacitor 0.1uF Yageo",
			Remark:      "Yageo Remark",
			Notes:       "To be cleared",
		},
	}

	inserted, _, err := UpsertMaterials(database, initialMats, nil)
	if err != nil {
		t.Fatalf("初始化 UpsertMaterials 失敗: %v", err)
	}
	if inserted != 3 {
		t.Fatalf("預期新增 3 筆，實際新增 %d 筆", inserted)
	}

	// 2. 測試批次更新 Notes
	notesMap := map[string]string{
		"Murata|GRM155R71C104KA88D": "Updated Note for Murata",
		"Samsung|CL05B104KO5NNNC":   "Updated Note for Samsung",
		"Yageo|CC0402KRX7R9BB104":   "", // 測試清空為空字串
		"Unknown|PN999":             "Non-existent part note", // 測試不存在之物料
	}

	updated, err := UpdateMaterialNotes(database, notesMap, nil)
	if err != nil {
		t.Fatalf("UpdateMaterialNotes 失敗: %v", err)
	}

	// 應更新 3 筆既有物料，Unknown 應被略過
	if updated != 3 {
		t.Errorf("預期更新 3 筆物料，實際更新 %d 筆", updated)
	}

	// 3. 驗證 Murata 物料更新結果
	murata, err := GetMaterialBySupplierPN(database, "Murata", "GRM155R71C104KA88D")
	if err != nil {
		t.Fatalf("查詢 Murata 失敗: %v", err)
	}
	if murata.Notes != "Updated Note for Murata" {
		t.Errorf("Murata Notes 錯誤，預期 'Updated Note for Murata'，實際 '%s'", murata.Notes)
	}
	// 驗證其餘欄位未被覆蓋
	if murata.HHPN != "HH-001" || murata.Description != "Capacitor 0.1uF" || murata.Remark != "Original Remark" {
		t.Errorf("Murata 其餘欄位遭意外修改: %+v", murata)
	}

	// 4. 驗證 Yageo 物料 Notes 是否清空
	yageo, err := GetMaterialBySupplierPN(database, "Yageo", "CC0402KRX7R9BB104")
	if err != nil {
		t.Fatalf("查詢 Yageo 失敗: %v", err)
	}
	if yageo.Notes != "" {
		t.Errorf("Yageo Notes 預期清空為空字串，實際為 '%s'", yageo.Notes)
	}
	if yageo.Remark != "Yageo Remark" {
		t.Errorf("Yageo Remark 遭意外修改: %s", yageo.Remark)
	}

	// 5. 驗證 Unknown 物料未被新增
	_, err = GetMaterialBySupplierPN(database, "Unknown", "PN999")
	if err == nil {
		t.Errorf("預期查無 Unknown 物料，但查詢成功")
	}

	// 6. 重複執行相同更新，因內容未改變，updated 應為 0
	updated2, err := UpdateMaterialNotes(database, notesMap, nil)
	if err != nil {
		t.Fatalf("第二次 UpdateMaterialNotes 失敗: %v", err)
	}
	if updated2 != 0 {
		t.Errorf("無內容變更時預期更新 0 筆，實際更新 %d 筆", updated2)
	}
}

// TestUpdateMaterialNote_Single 驗證 UpdateMaterialNote 依 MaterialID 單獨更新 Notes 的行為：
// 1. 成功更新單筆物料的 notes（支援多行文字，含 \n）
// 2. 成功將 notes 清空為空字串
// 3. 其餘欄位 (HHPN, Description 等) 保持不變
// 4. 當傳入無效 ID 或不存在的 ID 時，正確回傳錯誤
func TestUpdateMaterialNote_Single(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_single_note.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("開啟資料庫失敗: %v", err)
	}
	defer Close(database)

	if err := AutoMigrate(database); err != nil {
		t.Fatalf("AutoMigrate 失敗: %v", err)
	}

	// 1. 建立測試物料
	m := Material{
		Supplier:    "TDK",
		SupplierPN:  "C1005X7R1H104K",
		HHPN:        "HH-TDK-01",
		Description: "TDK Cap 0.1uF",
		Remark:      "Initial Remark",
		Notes:       "Initial Note",
	}
	if err := database.Create(&m).Error; err != nil {
		t.Fatalf("建立測試物料失敗: %v", err)
	}

	// 2. 測試更新為多行文字 (包含 Shift-Enter 產生的換行符號)
	multiLineNote := "Line 1: Special notice\nLine 2: Shift-Enter new line\nLine 3: Finished"
	if err := UpdateMaterialNote(database, m.ID, multiLineNote); err != nil {
		t.Fatalf("UpdateMaterialNote (多行) 失敗: %v", err)
	}

	var checked Material
	if err := database.First(&checked, m.ID).Error; err != nil {
		t.Fatalf("讀取更新後的 Material 失敗: %v", err)
	}
	if checked.Notes != multiLineNote {
		t.Errorf("Notes 更新不符，預期:\n%s\n實際:\n%s", multiLineNote, checked.Notes)
	}
	if checked.HHPN != "HH-TDK-01" || checked.Description != "TDK Cap 0.1uF" || checked.Remark != "Initial Remark" {
		t.Errorf("其他屬性遭意外修改: %+v", checked)
	}

	// 3. 測試清空 Notes
	if err := UpdateMaterialNote(database, m.ID, ""); err != nil {
		t.Fatalf("UpdateMaterialNote (清空) 失敗: %v", err)
	}
	if err := database.First(&checked, m.ID).Error; err != nil {
		t.Fatalf("讀取清空後的 Material 失敗: %v", err)
	}
	if checked.Notes != "" {
		t.Errorf("Notes 預期為空字串，實際為 '%s'", checked.Notes)
	}

	// 4. 測試無效 ID 與不存在的 ID
	if err := UpdateMaterialNote(database, 0, "fail"); err == nil {
		t.Errorf("傳入 ID=0 預期回傳錯誤，實際無錯誤")
	}
	if err := UpdateMaterialNote(database, -1, "fail"); err == nil {
		t.Errorf("傳入 ID=-1 預期回傳錯誤，實際無錯誤")
	}
	if err := UpdateMaterialNote(database, 999999, "fail"); err == nil {
		t.Errorf("傳入不存在的 ID 預期回傳錯誤，實際無錯誤")
	}
}
