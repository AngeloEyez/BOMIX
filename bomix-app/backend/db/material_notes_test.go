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
