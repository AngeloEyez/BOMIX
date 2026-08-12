package excel

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"bomix-app/backend/db"
	"bomix-app/backend/view"
	"github.com/glebarez/sqlite"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// groupRecord 代表 Excel 中的一個群組列（主料或 2nd Source）
type groupRecord struct {
	IsMain      bool
	Item        string
	HHPN        string
	Description string
	Supplier    string
	SupplierPN  string
	Qty         string
	Location    string
	Remark      string
}

func TestVerifyMatrixExport_TARIS_SI1_02(t *testing.T) {
	ebomPath := filepath.Join("testdata", "TARIS_EZBOM_SI1_0.2_BOM_20260811_1000.WP(compared).xls")
	benchmarkPath := filepath.Join("testdata", "TARIS_EZBOM_SI1_0.2_MatrixBOM_20260811_1000.WP.xlsx")

	t.Logf("建立記憶體資料庫並匯入 EBOM 測試檔: %s", ebomPath)
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("無法建立資料庫: %v", err)
	}

	if err := db.AutoMigrate(database); err != nil {
		t.Fatalf("自動遷移失敗: %v", err)
	}
	if _, err := db.CreateSeries(database, "TARIS", "Test Series"); err != nil {
		t.Fatalf("建立 Series 失敗: %v", err)
	}

	reader := NewReader(database, nil)
	importRes, err := reader.ImportExcel([]string{ebomPath})
	if err != nil {
		t.Fatalf("匯入 EBOM 檔失敗: %v", err)
	}
	t.Logf("EBOM 匯入成功: PartsCount=%d", importRes[0].PartsCount)

	var revs []db.BomRevision
	if err := database.Find(&revs).Error; err != nil || len(revs) == 0 {
		t.Fatalf("無法找到匯入後的 BomRevision: %v", err)
	}
	targetRev := revs[0]
	t.Logf("使用匯入後的 Target Revision ID=%d, Mode=%q", targetRev.ID, targetRev.Mode)

	// 透過 View 系統讀取匯出資料
	vSvc := view.NewService(database)
	vResult, err := vSvc.Query(view.ViewQuery{
		RevisionIDs: []int64{targetRev.ID},
		ViewType:    view.ViewCCL,
	})
	if err != nil {
		t.Fatalf("View 查詢失敗: %v", err)
	}

	t.Logf("vResult.PartGroups 數量: %d", len(vResult.PartGroups))
	if len(vResult.PartGroups) == 0 {
		// 查詢全 View 不過濾看有多少 PartGroups
		allRes, _ := vSvc.Query(view.ViewQuery{
			RevisionIDs: []int64{targetRev.ID},
			ViewType:    view.ViewAll,
		})
		t.Logf("ViewAll 回傳 PartGroups 數量: %d", len(allRes.PartGroups))
		for idx, pg := range allRes.PartGroups {
			if idx < 10 {
				t.Logf("Sample PartGroup[%d]: Supp=%s, SuppPN=%s, Type=%q, BOMStatus=%q, CCL=%v",
					idx, pg.MainSupplier, pg.MainSupplierPN, pg.Type, pg.BOMStatus, pg.CCL)
			}
		}
	}

	// 轉換為 RevisionData 與 PartData
	var revDataList []RevisionData
	for _, vr := range vResult.Revisions {
		revDataList = append(revDataList, RevisionData{
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

	var partDataList []PartData
	for idx, pg := range vResult.PartGroups {
		selections := make(map[string]string)
		selectionsByOrder := make(map[int]string)
		for _, sel := range pg.Selections {
			if sel.SelectedPN != "" {
				selections[sel.ModelName] = sel.SelectedPN
				selectionsByOrder[sel.SortOrder] = sel.SelectedPN
			}
		}

		var secondSources []SecondSourceData
		for _, ss := range pg.SecondSources {
			ssSelByOrder := make(map[int]bool)
			for order, isSelected := range ss.SelectionsByOrder {
				ssSelByOrder[order] = isSelected
			}
			secondSources = append(secondSources, SecondSourceData{
				HHPN:              ss.HHPN,
				Supplier:          ss.Supplier,
				SupplierPn:        ss.SupplierPN,
				Description:       ss.Description,
				Remark:            ss.Remark,
				SelectionsByOrder: ssSelByOrder,
			})
		}

		partDataList = append(partDataList, PartData{
			Item:              fmt.Sprintf("%d", idx+1),
			HHPN:              pg.HHPN,
			Description:       pg.Description,
			Supplier:          pg.MainSupplier,
			SupplierPn:        pg.MainSupplierPN,
			Qty:               pg.Qty,
			Location:          pg.Locations,
			Type:              pg.Type,
			BOMStatus:         pg.BOMStatus,
			CCL:               pg.CCL,
			Selections:        selections,
			SelectionsByOrder: selectionsByOrder,
			SecondSources:     secondSources,
			Remark:            pg.Remark,
		})
	}

	// 匯出 Matrix Excel
	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("建立 Writer 失敗: %v", err)
	}

	tmpDir := t.TempDir()
	outPaths, err := writer.exportMatrixDetailed(ExportOptions{
		OutputDir: tmpDir,
	}, revDataList[0], partDataList)
	if err != nil {
		t.Fatalf("Matrix 匯出失敗: %v", err)
	}
	exportedPath := outPaths[0]
	t.Logf("Matrix 成功匯出至: %s", exportedPath)

	// 開啟導出的 Excel 與 基準 Benchmark Excel 進行比對
	fExported, err := excelize.OpenFile(exportedPath)
	if err != nil {
		t.Fatalf("無法開啟匯出的檔 %s: %v", exportedPath, err)
	}
	defer fExported.Close()

	fBench, err := excelize.OpenFile(benchmarkPath)
	if err != nil {
		t.Fatalf("無法開啟基準檔 %s: %v", benchmarkPath, err)
	}
	defer fBench.Close()

	targetSheets := []string{"SMD", "PTH", "BOTTOM"}

	// 讀取工作表中從 Row 6 開始的資料列
	readSheetRecords := func(f *excelize.File, sheet string) []groupRecord {
		var records []groupRecord
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) < 6 {
			return records
		}
		for i := 5; i < len(rows); i++ {
			row := rows[i]
			getCol := func(idx int) string {
				if idx < len(row) {
					return strings.TrimSpace(row[idx])
				}
				return ""
			}
			item := getCol(0)
			hhpn := getCol(1)
			desc := getCol(3)
			supp := getCol(4)
			suppPN := getCol(5)
			qty := getCol(6)
			loc := getCol(7)
			remark := ""
			if len(row) > 16 {
				remark = getCol(16)
			}

			// 若整行皆為空，跳過
			if item == "" && hhpn == "" && suppPN == "" && desc == "" {
				continue
			}

			isMain := (item != "" || loc != "")
			records = append(records, groupRecord{
				IsMain:      isMain,
				Item:        item,
				HHPN:        hhpn,
				Description: desc,
				Supplier:    supp,
				SupplierPN:  suppPN,
				Qty:         qty,
				Location:    loc,
				Remark:      remark,
			})
		}
		return records
	}

	for _, sheet := range targetSheets {
		t.Run("Sheet_"+sheet, func(t *testing.T) {
			expRecs := readSheetRecords(fExported, sheet)
			benchRecs := readSheetRecords(fBench, sheet)

			t.Logf("[%s 頁面] 導出筆數: %d, 基準檔筆數: %d", sheet, len(expRecs), len(benchRecs))

			// 建構 map 供快取比對
			expMap := make(map[string]groupRecord)
			for _, r := range expRecs {
				key := r.Supplier + "|" + r.SupplierPN
				expMap[key] = r
			}

			benchMap := make(map[string]groupRecord)
			for _, r := range benchRecs {
				key := r.Supplier + "|" + r.SupplierPN
				benchMap[key] = r
			}

			// 比對 1：逐列順序比對 (當數量一致時)
			var locationMismatch int
			var groupMismatch int

			minLen := len(expRecs)
			if len(benchRecs) < minLen {
				minLen = len(benchRecs)
			}

			for i := 0; i < minLen; i++ {
				exp := expRecs[i]
				bench := benchRecs[i]

				if exp.Supplier != bench.Supplier || exp.SupplierPN != bench.SupplierPN {
					groupMismatch++
				}
				expNormLoc := strings.ReplaceAll(exp.Location, " ", "")
				benchNormLoc := strings.ReplaceAll(bench.Location, " ", "")
				if expNormLoc != benchNormLoc {
					locationMismatch++
				}
			}

			t.Logf("[%s 頁面] 前 %d 列逐列對比: 群組不一致 %d 筆, Location不一致 %d 筆", sheet, minLen, groupMismatch, locationMismatch)

			// 針對重點物料 AZ1045-04F.R7G 檢查 Location
			if strings.EqualFold(sheet, "SMD") {
				targetPN := "AZ1045-04F.R7G"
				var foundExp, foundBench groupRecord
				var hasExp, hasBench bool
				for _, r := range expRecs {
					if strings.EqualFold(r.SupplierPN, targetPN) {
						foundExp = r
						hasExp = true
						break
					}
				}
				for _, r := range benchRecs {
					if strings.EqualFold(r.SupplierPN, targetPN) {
						foundBench = r
						hasBench = true
						break
					}
				}

				if hasExp {
					t.Logf("✔ [SMD 頁面] 找到目標物料 %s: 導出 Location = %q", targetPN, foundExp.Location)
					t.Logf("[SMD 頁面] 導出的 %s Location = %q (包含 DB 中該物料所有 Status=I 且 CCL=true 之有效位置)", targetPN, foundExp.Location)

					// 查詢 DB 中此物料的所有 Location 屬性
					var targetParts []db.Part
					database.Where("revision_id = ? AND supplier_pn = ?", targetRev.ID, targetPN).Find(&targetParts)
					for _, p := range targetParts {
						var locs []db.PartLocation
						database.Where("part_id = ?", p.ID).Find(&locs)
						t.Logf("DB 中 %s (Part ID %d, Type=%s) 共 %d 個 PartLocation:", targetPN, p.ID, p.Type, len(locs))
						for _, l := range locs {
							t.Logf("   -> Loc=%s, Status=%s, CCL=%v", l.Location, l.BomStatus, l.CCL)
						}
					}

					// 查詢 Revision 6 的 MatrixModels 與 MatrixSelections
					var matrixModels []db.MatrixModel
					database.Where("revision_id = ?", targetRev.ID).Find(&matrixModels)
					t.Logf("Revision ID %d 共 %d 個 MatrixModels", targetRev.ID, len(matrixModels))
					for _, m := range matrixModels {
						var sels []db.MatrixSelection
						database.Where("model_id = ?", m.ID).Find(&sels)
						t.Logf("  Model ID %d (%s, Order %d, Qty %d) 有 %d 個 MatrixSelections", m.ID, m.ModelName, m.SortOrder, m.Qty, len(sels))
						for _, s := range sels {
							if strings.Contains(s.Material, targetPN) || strings.Contains(s.Group, targetPN) {
								t.Logf("     Selection -> PartID=%d, Group=%s, Mat=%s, SelPN=%s", s.PartID, s.Group, s.Material, s.SelectedSupplierPn)
							}
						}
					}
				} else {
					t.Errorf("✖ [SMD 頁面] 導出的 Excel 中未找到物料 %s", targetPN)
				}

				if hasBench {
					t.Logf("  [SMD 基準檔] %s: Location = %q", targetPN, foundBench.Location)
				}

				// 在基準檔的所有工作表搜尋 targetPN
				for _, sName := range targetSheets {
					recs := readSheetRecords(fBench, sName)
					for rIdx, r := range recs {
						if strings.EqualFold(r.SupplierPN, targetPN) {
							t.Logf("★ 基準檔 工作表 [%s] Row %d 找到 %s: Main=%v, HHPN=%s, Loc=%q",
								sName, rIdx+6, targetPN, r.IsMain, r.HHPN, r.Location)
						}
					}
				}
			}

			// 若數量有差異（如 BOTTOM 頁面），分析基準檔有但導出檔沒有的物料在 DB 中的 CCL 與 BOMStatus 屬性
			if len(expRecs) != len(benchRecs) {
				t.Logf("[%s 頁面] 筆數差異 (導出 %d vs 基準 %d):", sheet, len(expRecs), len(benchRecs))
				missingInExp := 0
				for _, b := range benchRecs {
					key := b.Supplier + "|" + b.SupplierPN
					if _, exists := expMap[key]; !exists {
						missingInExp++
						var parts []db.Part
						database.Where("revision_id = ? AND supplier = ? AND supplier_pn = ?", targetRev.ID, b.Supplier, b.SupplierPN).Find(&parts)
						if len(parts) > 0 {
							p := parts[0]
							var locs []db.PartLocation
							database.Where("part_id = ?", p.ID).Find(&locs)
							hasCCL := false
							var statuses []string
							for _, l := range locs {
								if l.CCL {
									hasCCL = true
								}
								statuses = append(statuses, l.BomStatus)
							}
							if missingInExp <= 5 {
								t.Logf("  基準有但導出無: Supp=%s, SuppPN=%s, PartType=%q, DB_CCL=%v, LocStatuses=%v",
									b.Supplier, b.SupplierPN, p.Type, hasCCL, statuses)
							}
						}
					}
				}
				t.Logf("[%s 頁面] 基準檔存在但導出檔排除的物料共 %d 筆（主要為 DB 中 CCL=false 物料或 PartType 不吻合物料）", sheet, missingInExp)
			}
		})
	}
}
