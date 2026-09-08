package excel

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"bomix-app/backend/db"
	"bomix-app/backend/view"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
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

	if _, err := os.Stat(ebomPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: %s not found", ebomPath)
	}
	if _, err := os.Stat(benchmarkPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: %s not found", benchmarkPath)
	}

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

			// 解析主料群組與其轄下的 2nd Sources 與 Location 集合
			type mainGroupData struct {
				MainSupplier   string
				MainSupplierPN string
				HHPN           string
				LocationRaw    string
				LocationSet    map[string]bool
				SecondSources  []string
			}

			parseMainGroups := func(recs []groupRecord) ([]*mainGroupData, map[string]*mainGroupData) {
				var list []*mainGroupData
				gMap := make(map[string]*mainGroupData)
				var current *mainGroupData

				for _, r := range recs {
					if r.IsMain {
						locSet := make(map[string]bool)
						if r.Location != "" {
							parts := strings.Split(r.Location, ",")
							for _, p := range parts {
								p = strings.TrimSpace(p)
								if p != "" {
									locSet[p] = true
								}
							}
						}
						key := r.Supplier + "|" + r.SupplierPN
						current = &mainGroupData{
							MainSupplier:   r.Supplier,
							MainSupplierPN: r.SupplierPN,
							HHPN:           r.HHPN,
							LocationRaw:    r.Location,
							LocationSet:    locSet,
							SecondSources:  []string{},
						}
						list = append(list, current)
						gMap[key] = current
					} else if current != nil {
						if r.Supplier != "" || r.SupplierPN != "" {
							ssKey := r.Supplier + "|" + r.SupplierPN
							current.SecondSources = append(current.SecondSources, ssKey)
						}
					}
				}
				for _, g := range list {
					sort.Strings(g.SecondSources)
				}
				return list, gMap
			}

			expGroups, expGroupMap := parseMainGroups(expRecs)
			benchGroups, _ := parseMainGroups(benchRecs)

			t.Logf("[%s 頁面] 主料群組數量: 導出=%d, 基準=%d", sheet, len(expGroups), len(benchGroups))
			assert.Equal(t, len(benchGroups), len(expGroups), fmt.Sprintf("[%s 頁面] 主料群組數量應完全一致", sheet))

			// 驗證項目 1：每一組 Main Source 所帶的 2nd Source 是否完全相同
			// 驗證項目 2：每一組 Main Source 的 Location 位置 (拆分集合) 是否完全相同
			var ssMismatchCount int
			var locSetMismatchCount int

			for _, bGroup := range benchGroups {
				key := bGroup.MainSupplier + "|" + bGroup.MainSupplierPN
				eGroup, exists := expGroupMap[key]
				if !exists {
					t.Errorf("[%s 頁面] 基準主料群組 %s 在導出檔中未找到", sheet, key)
					continue
				}

				// 1. 比對 2nd Source 列表
				ssMatched := true
				if len(bGroup.SecondSources) != len(eGroup.SecondSources) {
					ssMatched = false
				} else {
					for i := range bGroup.SecondSources {
						if bGroup.SecondSources[i] != eGroup.SecondSources[i] {
							ssMatched = false
							break
						}
					}
				}
				if !ssMatched {
					ssMismatchCount++
					t.Logf("✖ [%s 頁面] 主料 %s 2nd Source 不一致: 基準=%v, 導出=%v",
						sheet, key, bGroup.SecondSources, eGroup.SecondSources)
				}

				// 2. 比對 Location 集合 (以逗號拆分為集合)
				locsMatched := true
				if len(bGroup.LocationSet) != len(eGroup.LocationSet) {
					locsMatched = false
				} else {
					for loc := range bGroup.LocationSet {
						if !eGroup.LocationSet[loc] {
							locsMatched = false
							break
						}
					}
				}
				if !locsMatched {
					locSetMismatchCount++
					t.Logf("✖ [%s 頁面] 主料 %s Location 集合不一致: 基準=%q, 導出=%q",
						sheet, key, bGroup.LocationRaw, eGroup.LocationRaw)
				}
			}

			t.Logf("✔ [%s 頁面] 主料群組 2nd Source 比對結果: %d 筆差異 (共 %d 組)", sheet, ssMismatchCount, len(benchGroups))
			t.Logf("✔ [%s 頁面] 主料群組 Location 集合比對結果: %d 筆差異 (共 %d 組)", sheet, locSetMismatchCount, len(benchGroups))

			assert.Equal(t, 0, ssMismatchCount, fmt.Sprintf("[%s 頁面] 每一組 Main Source 的 2nd Source 應與基準檔 100%% 相同", sheet))
			assert.Equal(t, 0, locSetMismatchCount, fmt.Sprintf("[%s 頁面] 每一組 Main Source 的 Location 集合應與基準檔 100%% 相同", sheet))

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
					var targetMats []db.Material
					database.Where("supplier_pn = ?", targetPN).Find(&targetMats)
					for _, m := range targetMats {
						var comps []db.RevisionComponent
						database.Where("revision_id = ? AND material_id = ?", targetRev.ID, m.ID).Find(&comps)
						for _, c := range comps {
							var locs []db.PartLocation
							database.Where("component_id = ?", c.ID).Find(&locs)
							t.Logf("DB 中 %s (Comp ID %d) 共 %d 個 PartLocation:", targetPN, c.ID, len(locs))
							for _, l := range locs {
								t.Logf("   -> Loc=%s, Type=%s, Status=%s, CCL=%v", l.Location, l.Type, l.BomStatus, l.CCL)
							}
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
							t.Logf("     Selection -> CompID=%d, MainMatID=%d, SelMatID=%d", s.ComponentID, s.MainMaterialID, s.SelectedMaterialID)
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
					if _, exists := expGroupMap[key]; !exists {
						missingInExp++
						var mat db.Material
						if err := database.Where("supplier = ? AND supplier_pn = ?", b.Supplier, b.SupplierPN).First(&mat).Error; err == nil {
							var comp db.RevisionComponent
							if err := database.Where("revision_id = ? AND material_id = ?", targetRev.ID, mat.ID).First(&comp).Error; err == nil {
								var locs []db.PartLocation
								database.Where("component_id = ?", comp.ID).Find(&locs)
								hasCCL := false
								var statuses []string
								for _, l := range locs {
									if l.CCL {
										hasCCL = true
									}
									statuses = append(statuses, l.BomStatus)
								}
								t.Logf("   - 缺少物料: %s | %s (Comp ID %d, Locations %d 個, CCL=%v, Statuses=%v)",
									b.Supplier, b.SupplierPN, comp.ID, len(locs), hasCCL, statuses)
							}
						}
					}
				}
				t.Logf("[%s 頁面] 基準檔存在但導出檔排除的物料共 %d 筆（主要為 DB 中 CCL=false 物料或 PartType 不吻合物料）", sheet, missingInExp)
			}
		})
	}
}
