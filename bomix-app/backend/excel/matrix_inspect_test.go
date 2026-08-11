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

func TestFullMatrixImportExportFlow(t *testing.T) {
	gdb := setupMatrixTestDB(t)
	lg := logger.NewLogger(100)
	reader := NewReader(gdb, lg)

	ebomPath1 := "testdata/TARISN_EZBOM_DB_0.5_BOM_20260622_0800.WP(compared).xls"
	ebomPath2 := "testdata/TARIS_EZBOM_SI1_0.1_BOM_20260722_1400.WP(compared).xls"
	matrixPath := "testdata/TARIS_EZBOM_SI1_0.1_MatrixBOM_20260722_1400.WP.xlsx"

	t.Log("=== 1. Importing EBOMs ===")
	results1, err := reader.ImportExcel([]string{ebomPath1, ebomPath2})
	if err != nil {
		t.Fatalf("Import EBOM failed: %v", err)
	}
	t.Logf("EBOM import results: %+v", results1)

	var revs []db.BomRevision
	gdb.Find(&revs)
	t.Logf("Revisions in DB after EBOM import: %d", len(revs))
	var dbParts []db.Part
	gdb.Where("revision_id = ?", 2).Find(&dbParts)
	t.Logf("Parts in DB for Rev 2: %d", len(dbParts))
	for _, p := range dbParts {
		if strings.Contains(p.SupplierPN, "AZC199") || strings.Contains(p.SupplierPN, "AP22818") || strings.Contains(p.SupplierPN, "LBSS139") {
			t.Logf("DB Part: ID=%d, RevID=%d, Type=%s, Sup=%s, PN=%s, Item=%s",
				p.ID, p.RevisionID, p.Type, p.Supplier, p.SupplierPN, p.Item)
		}
	}

	t.Log("=== 2. Importing Matrix BOM ===")
	results2, err := reader.ImportExcel([]string{matrixPath})
	if err != nil {
		t.Fatalf("Import Matrix failed: %v", err)
	}
	t.Logf("Matrix import results: %+v", results2)

	var models []db.MatrixModel
	gdb.Find(&models)
	t.Logf("MatrixModels in DB: %d", len(models))
	for _, m := range models {
		t.Logf("Model ID=%d, RevID=%d, Name=%s, SortOrder=%d, Qty=%d", m.ID, m.RevisionID, m.ModelName, m.SortOrder, m.Qty)
	}

	var selections []db.MatrixSelection
	gdb.Find(&selections)
	t.Logf("MatrixSelections in DB: %d", len(selections))
	for _, s := range selections {
		if strings.Contains(s.SelectedSupplierPn, "LMBT3906") || strings.Contains(s.SelectedSupplierPn, "MMDT3906") {
			t.Logf("Selection: ID=%d, RevID=%d, ModelID=%d, Group=%s, Material=%s, SelectedPN=%s",
				s.ID, s.RevisionID, s.ModelID, s.Group, s.Material, s.SelectedSupplierPn)
		}
	}

	t.Log("=== 3. Querying View Service ===")
	viewSvc := view.NewService(gdb, lg)
	if len(revs) == 0 {
		t.Fatalf("No revisions in DB!")
	}

	var targetRevID int64
	for _, r := range revs {
		if strings.EqualFold(r.Phase, "SI1") && strings.EqualFold(r.Version, "0.1") {
			targetRevID = r.ID
		}
	}
	if targetRevID == 0 {
		t.Fatalf("Target revision SI1 0.1 not found in DB!")
	}
	t.Logf("Selected targetRevID: %d", targetRevID)
	query := view.ViewQuery{
		RevisionIDs: []int64{targetRevID},
		ViewType:    view.ViewAll,
	}
	viewResult, err := viewSvc.Query(query)
	if err != nil {
		t.Fatalf("View query failed: %v", err)
	}
	t.Logf("View PartGroups count: %d", len(viewResult.PartGroups))
	typeCounts := make(map[string]int)
	for _, pg := range viewResult.PartGroups {
		typeCounts[pg.Type]++
	}
	t.Logf("PartGroups count by Type: %+v", typeCounts)

	t.Log("=== 4. Exporting Matrix ===")
	writer, err := NewWriter(lg)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	revDataList, partDataList := convertViewResultToExportData(viewResult)

	outPath := t.TempDir() + "/exported_matrix.xlsx"
	exportOpts := ExportOptions{
		Format:      types.FormatMatrix,
		RevisionIDs: []string{fmt.Sprintf("%d", targetRevID)},
		OutputPath:  outPath,
		PartData:    partDataList,
		Revisions:   revDataList,
	}

	outFiles, err := writer.ExportExcel(exportOpts)
	if err != nil {
		t.Fatalf("Export matrix failed: %v", err)
	}
	t.Logf("Exported matrix file: %v", outFiles)

	// Compare checkmarks
	compareMatrixCheckmarks(t, matrixPath, outFiles[0])
}

func convertViewResultToExportData(vr *view.ViewResult) ([]RevisionData, []PartData) {
	revDataList := make([]RevisionData, 0, len(vr.Revisions))
	for _, vrev := range vr.Revisions {
		revDataList = append(revDataList, RevisionData{
			ID:               fmt.Sprintf("%d", vrev.ID),
			ProjectCode:      vrev.ProjectCode,
			Description:      vrev.Description,
			SchematicVersion: vrev.SchematicVersion,
			PCBVersion:       vrev.PCBVersion,
			PCAPN:            vrev.PCAPN,
			Phase:            vrev.Phase,
			Version:          vrev.Version,
			Date:             vrev.Date,
			SourceFile:       vrev.SourceFile,
			ModelNames:       vrev.ModelNames,
			ModelQty:         vrev.ModelQty,
			ModelQtyByOrder:  vrev.ModelQtyByOrder,
		})
	}

	partDataList := make([]PartData, 0, len(vr.PartGroups))
	for idx, pg := range vr.PartGroups {
		selections := make(map[string]string)
		selectionsByOrder := make(map[int]string)
		selectionsByRevAndOrder := make(map[string]map[int]string)
		selectionsByRevAndName := make(map[string]map[string]string)
		selectionsByMaterialByOrder := make(map[int]string)
		selectionsByRevAndMaterial := make(map[string]map[int]string)

		for _, sel := range pg.Selections {
			if sel.SelectedPN != "" {
				revIDStr := fmt.Sprintf("%d", sel.RevisionID)
				if _, exists := selections[sel.ModelName]; !exists {
					selections[sel.ModelName] = sel.SelectedPN
				}
				if _, exists := selectionsByOrder[sel.SortOrder]; !exists {
					selectionsByOrder[sel.SortOrder] = sel.SelectedPN
				}
				if sel.SelectedMaterial != "" {
					if _, exists := selectionsByMaterialByOrder[sel.SortOrder]; !exists {
						selectionsByMaterialByOrder[sel.SortOrder] = sel.SelectedMaterial
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
				if sel.SelectedMaterial != "" {
					if selectionsByRevAndMaterial[revIDStr] == nil {
						selectionsByRevAndMaterial[revIDStr] = make(map[int]string)
					}
					selectionsByRevAndMaterial[revIDStr][sel.SortOrder] = sel.SelectedMaterial
				}
			}
		}

		ssData := make([]SecondSourceData, 0, len(pg.SecondSources))
		for _, ss := range pg.SecondSources {
			ssData = append(ssData, SecondSourceData{
				HHPN:              ss.HHPN,
				Supplier:          ss.Supplier,
				SupplierPn:        ss.SupplierPN,
				Description:       ss.Description,
				Remark:            ss.Remark,
				SourceRevisionIDs: ss.SourceRevisionIDs,
				SelectionsByOrder: ss.SelectionsByOrder,
			})
		}

		partDataList = append(partDataList, PartData{
			Item:                         fmt.Sprintf("%d", idx+1),
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
			SecondSources:                ssData,
			Selections:                   selections,
			SelectionsByOrder:            selectionsByOrder,
			SelectionsByRevAndOrder:      selectionsByRevAndOrder,
			SelectionsByRevAndName:       selectionsByRevAndName,
			SelectionsByMaterialByOrder:  selectionsByMaterialByOrder,
			SelectionsByRevAndMaterial:   selectionsByRevAndMaterial,
			MainSelectionsByOrder:        pg.MainSelectionsByOrder,
			SourceRevisionIDs:            pg.SourceRevisionIDs,
		})
	}

	return revDataList, partDataList
}

func compareMatrixCheckmarks(t *testing.T, origPath, exportPath string) {
	origF, err := excelize.OpenFile(origPath)
	if err != nil {
		t.Fatalf("Failed to open original matrix file: %v", err)
	}
	defer origF.Close()

	expF, err := excelize.OpenFile(exportPath)
	if err != nil {
		t.Fatalf("Failed to open exported matrix file: %v", err)
	}
	defer expF.Close()

	t.Log("=== Comparing Original vs Exported Checkmarks ===")
	sheets := []string{"SMD", "PTH", "BOTTOM"}

	mismatches := 0
	totalOrigChecks := 0
	totalExpChecks := 0

	origGlobalMap := make(map[string]map[int]string)
	expGlobalMap := make(map[string]map[int]string)

	for _, sheet := range sheets {
		origRows, err1 := origF.GetRows(sheet)
		expRows, err2 := expF.GetRows(sheet)
		if err1 == nil {
			for r := 5; r < len(origRows); r++ {
				row := origRows[r]
				sup := safeGetCol(row, 4)
				pn := safeGetCol(row, 5)
				if sup == "" && pn == "" {
					continue
				}
				key := fmt.Sprintf("%s|%s", strings.TrimSpace(sup), strings.TrimSpace(pn))
				if origGlobalMap[key] == nil {
					origGlobalMap[key] = make(map[int]string)
				}
				for c := 10; c <= 16; c++ {
					val := safeGetCol(row, c)
					if strings.EqualFold(val, "V") {
						origGlobalMap[key][c] = "V"
						totalOrigChecks++
					}
				}
			}
		}

		if err2 == nil {
			for r := 5; r < len(expRows); r++ {
				row := expRows[r]
				sup := safeGetCol(row, 4)
				pn := safeGetCol(row, 5)
				if sup == "" && pn == "" {
					continue
				}
				key := fmt.Sprintf("%s|%s", strings.TrimSpace(sup), strings.TrimSpace(pn))
				if expGlobalMap[key] == nil {
					expGlobalMap[key] = make(map[int]string)
				}
				for c := 10; c <= 16; c++ {
					val := safeGetCol(row, c)
					if strings.EqualFold(val, "V") {
						expGlobalMap[key][c] = "V"
						totalExpChecks++
					}
				}
			}
		}
	}

	for key, origCols := range origGlobalMap {
		expCols := expGlobalMap[key]
		for col := 10; col <= 16; col++ {
			origV := origCols[col]
			expV := ""
			if expCols != nil {
				expV = expCols[col]
			}
			if origV != expV {
				colName, _ := excelize.ColumnNumberToName(col + 1)
				t.Errorf("MISMATCH Material [%s] Col [%s]: Original='%s', Exported='%s'",
					key, colName, origV, expV)
				mismatches++
			}
		}
	}

	t.Logf("Total Original V checks: %d, Total Exported V checks: %d, Total mismatches: %d",
		totalOrigChecks, totalExpChecks, mismatches)
}
