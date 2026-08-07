package excelimport

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// newTestXlsx builds an in-memory xlsx from the given rows using excelize.
func newTestXlsx(rows [][]string) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := f.GetSheetName(0)
	if sheet == "" {
		sheet = "Sheet1"
	}
	for r, row := range rows {
		for c, cell := range row {
			col, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				return nil, fmt.Errorf("cell name: %w", err)
			}
			if err := f.SetCellValue(sheet, col, cell); err != nil {
				return nil, err
			}
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
