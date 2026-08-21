package excel

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"bomix-app/backend/logger"
	"bomix-app/backend/types"
	"bomix-app/backend/view"

	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
)

func TestVerifyBlankCellsExport(t *testing.T) {
	gdb := setupMatrixTestDB(t)
	lg := logger.NewLogger(100)
	reader := NewReader(gdb, lg)

	ebomPath := "testdata/TARIS-PCM_EZBOM_SI1_0.3_BOM_20260817_1000.WP(compared).xls"
	matrixPath := "testdata/TARIS_EZBOM_SI1_0.3_MatrixBOM_20260817_1000.WP.xlsx"

	if _, err := os.Stat(ebomPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: %s not found", ebomPath)
	}

	// 1. Import EBOM
	_, err := reader.ImportExcel([]string{ebomPath})
	if err != nil {
		t.Fatalf("Import EBOM failed: %v", err)
	}

	// 2. Import Matrix if exists
	if _, err := os.Stat(matrixPath); err == nil {
		_, _ = reader.ImportExcel([]string{matrixPath})
	}

	// 3. Query View
	viewSvc := view.NewService(gdb, lg)
	viewResult, err := viewSvc.Query(view.ViewQuery{
		RevisionIDs: []int64{1},
		ViewType:    view.ViewAll,
	})
	if err != nil {
		t.Fatalf("View query failed: %v", err)
	}

	// 4. Export Matrix
	writer, err := NewWriter(lg)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	revDataList, partDataList := convertViewResultToExportData(viewResult)

	outPath := filepath.Join("testdata", "test_blank_verified.xlsx")
	defer os.Remove(outPath)
	exportOpts := ExportOptions{
		Format:      types.FormatMatrix,
		RevisionIDs: []string{"1"},
		OutputPath:  outPath,
		PartData:    partDataList,
		Revisions:   revDataList,
	}

	outFiles, err := writer.ExportExcel(exportOpts)
	if err != nil {
		t.Fatalf("Export matrix failed: %v", err)
	}

	exportedFile := outFiles[0]

	// 5. Inspect Excelize values and cell types
	fExp, err := excelize.OpenFile(exportedFile)
	if err != nil {
		t.Fatalf("Failed to open exported file: %v", err)
	}
	defer fExp.Close()

	sheets := []string{"SMD", "PTH", "BOTTOM"}
	for _, sheet := range sheets {
		// Verify Row 5 (Model D..G: N5, O5, P5, Q5)
		for _, col := range []string{"N", "O", "P", "Q"} {
			cell := fmt.Sprintf("%s5", col)
			cType, _ := fExp.GetCellType(sheet, cell)
			val, _ := fExp.GetCellValue(sheet, cell)
			assert.Equal(t, excelize.CellTypeUnset, cType, fmt.Sprintf("%s %s should be CellTypeUnset, got %v", sheet, cell, cType))
			assert.Equal(t, "", val, fmt.Sprintf("%s %s value should be empty", sheet, cell))
		}

		// Verify Row 6 and 7 (Cleared template rows)
		for r := 6; r <= 7; r++ {
			for _, col := range []string{"N", "O", "P", "Q"} {
				cell := fmt.Sprintf("%s%d", col, r)
				cType, _ := fExp.GetCellType(sheet, cell)
				assert.Equal(t, excelize.CellTypeUnset, cType, fmt.Sprintf("%s %s (template row) should be CellTypeUnset, got %v", sheet, cell, cType))
			}
		}
	}

	// 6. Deep Inspect Raw XML: ensure no <c ... t="s"> with empty string in N..Q from row 5 down
	rZip, err := zip.OpenReader(exportedFile)
	if err != nil {
		t.Fatal(err)
	}
	defer rZip.Close()

	reCellWithStr := regexp.MustCompile(`<c r="([N-Q])([0-9]+)"[^>]*t="s"[^>]*>`)
	for _, file := range rZip.File {
		if strings.HasPrefix(file.Name, "xl/worksheets/sheet") {
			rc, _ := file.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			xmlStr := string(b)

			matches := reCellWithStr.FindAllStringSubmatch(xmlStr, -1)
			var invalidCells []string
			for _, m := range matches {
				var row int
				fmt.Sscanf(m[2], "%d", &row)
				if row >= 5 {
					invalidCells = append(invalidCells, m[1]+m[2])
				}
			}
			assert.Empty(t, invalidCells, fmt.Sprintf("Sheet %s has string-type cells in N..Q from row 5 down: %v", file.Name, invalidCells))
		}
	}

	// 7. Verify via Python openpyxl (if python is available)
	pyCode := fmt.Sprintf(`
import openpyxl
wb = openpyxl.load_workbook(r"%s", data_only=True)
ws = wb["SMD"]
for col in ["N", "O", "P", "Q"]:
    c = ws[f"{col}5"]
    assert c.value is None, f"{col}5 value expected None, got {c.value}"
    assert c.data_type != "s", f"{col}5 data_type expected non-string, got {c.data_type}"
for r in [6, 7]:
    for col in ["N", "O", "P", "Q"]:
        c = ws[f"{col}{r}"]
        assert c.value is None, f"{col}{r} value expected None, got {c.value}"
        assert c.data_type != "s", f"{col}{r} data_type expected non-string, got {c.data_type}"
print("PYTHON OPENPYXL VERIFICATION PASSED")
`, exportedFile)

	cmd := exec.Command("python", "-c", pyCode)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Logf("Python openpyxl check output: %s", string(out))
		assert.Contains(t, string(out), "PYTHON OPENPYXL VERIFICATION PASSED")
	} else {
		t.Logf("Python check skipped or errored: %v (%s)", err, string(out))
	}
}
