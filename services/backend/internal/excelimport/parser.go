package excelimport

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Limits enforced by this feature. Keep them as package constants so they are
// easy to tune and fully isolated from the rest of the codebase.
const (
	MaxFileBytes  = 5 * 1024 * 1024
	MaxRows       = 2000
	MaxContentLen = 1000
	PreviewSample = 20
)

var (
	ErrUnsupportedFormat = errors.New("unsupported file format: use xlsx or csv")
	ErrInvalidFile       = errors.New("invalid Excel/CSV file")
	ErrNoDataRows        = errors.New("no data rows found in file")
	ErrTooManyRows       = errors.New("file exceeds the maximum of 2000 rows")
)

// RawRow is one parsed data row from the source file.
// SourceRow is 1-based and matches the original spreadsheet row for reporting.
type RawRow struct {
	Phone     string
	Content   string
	SourceRow int
}

// headerWords are common column titles used to detect and skip header rows.
var headerWords = []string{
	"phone", "number", "mobile", "tel", "cell", "name",
	"رقم", "الهاتف", "الموبايل", "الجوال", "الاسم",
	"content", "message", "msg", "text", "نص", "الرسالة", "رسالة",
}

// phoneHeaderWords identify the phone/mobile column in a header row.
var phoneHeaderWords = []string{
	"phone", "number", "mobile", "tel", "cell",
	"رقم", "الهاتف", "الموبايل", "الجوال", "هاتف", "جوال",
}

// contentHeaderWords identify the message/content column in a header row.
var contentHeaderWords = []string{
	"content", "message", "msg", "text", "sms",
	"نص", "الرسالة", "رسالة",
}

// colMap holds the detected indices of the phone and content columns.
type colMap struct {
	phoneIdx   int
	contentIdx int
	hasPhone   bool
	hasContent bool
}

// Parse reads an .xlsx or .csv stream and returns data rows (headers and empty
// rows excluded). The phone and content columns are detected from the header
// row; when no header is present the first two columns are assumed, with a
// per-row fallback that scans the row for a phone-like cell.
func Parse(r io.Reader, ext string) ([]RawRow, error) {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "xlsx":
		return parseXlsx(r)
	case "csv", "txt":
		return parseCsv(r)
	default:
		return nil, ErrUnsupportedFormat
	}
}

func parseXlsx(r io.Reader) ([]RawRow, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, ErrInvalidFile
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalidFile
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, ErrInvalidFile
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, ErrInvalidFile
	}
	return rowsToRaw(rows)
}

func parseCsv(r io.Reader) ([]RawRow, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, ErrInvalidFile
	}
	// Strip UTF-8 BOM if present.
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, ErrInvalidFile
	}
	return rowsToRaw(rows)
}

func rowsToRaw(rows [][]string) ([]RawRow, error) {
	var out []RawRow
	if len(rows) == 0 {
		return out, nil
	}

	headerIdx, cm := findHeader(rows)

	phoneCol, contentCol := 0, 1
	if headerIdx >= 0 {
		if cm.hasPhone {
			phoneCol = cm.phoneIdx
		}
		if cm.hasContent {
			contentCol = cm.contentIdx
		} else if cm.hasPhone {
			contentCol = firstNonPhoneCol(rows[headerIdx], phoneCol)
		}
	}

	for i, row := range rows {
		if headerIdx >= 0 && i <= headerIdx {
			continue
		}
		if isBlankRow(row) {
			continue
		}
		phone := strings.TrimSpace(cellAt(row, phoneCol))
		content := strings.TrimSpace(cellAt(row, contentCol))
		if !isPhoneLike(phone) {
			if scanned, ok := scanPhone(row, contentCol); ok {
				if content == "" {
					content = strings.TrimSpace(cellAt(row, contentCol))
				}
				phone = scanned
			}
		}
		if content == "" {
			content = firstNonPhoneCell(row, phoneCol)
		}
		if phone == "" && content == "" {
			continue
		}
		out = append(out, RawRow{
			Phone:     phone,
			Content:   content,
			SourceRow: i + 1,
		})
	}
	return out, nil
}

// findHeader locates the header row and the phone/content columns. It skips
// leading blank rows and title rows until it finds a row that either carries a
// phone+content header or matches the known header words.
func findHeader(rows [][]string) (int, colMap) {
	var candidateIdx = -1
	var candidate colMap
	for i, row := range rows {
		if isBlankRow(row) {
			continue
		}
		cm := detectCols(row)
		if cm.hasPhone && cm.hasContent {
			return i, cm
		}
		if isHeaderRow(row) {
			if candidateIdx == -1 {
				candidateIdx, candidate = i, cm
			}
			continue
		}
		// First non-blank, non-header row: data begins here.
		return candidateIdx, candidate
	}
	return candidateIdx, candidate
}

// detectCols inspects a header row and returns the indices of the phone and
// content columns based on the column titles.
func detectCols(header []string) colMap {
	var cm colMap
	for i, cell := range header {
		s := strings.ToLower(strings.TrimSpace(cell))
		if s == "" {
			continue
		}
		if !cm.hasPhone && containsAnyWord(s, phoneHeaderWords) {
			cm.phoneIdx, cm.hasPhone = i, true
		}
		if !cm.hasContent && containsAnyWord(s, contentHeaderWords) {
			cm.contentIdx, cm.hasContent = i, true
		}
	}
	return cm
}

// firstNonPhoneCol returns the first column (other than phoneCol) whose header
// cell is not phone-like, or phoneCol+1 when every other column looks numeric.
func firstNonPhoneCol(header []string, phoneCol int) int {
	for j, cell := range header {
		if j == phoneCol {
			continue
		}
		if !isPhoneLike(strings.TrimSpace(cell)) {
			return j
		}
	}
	if phoneCol != 1 && len(header) > 1 {
		return 1
	}
	if len(header) > phoneCol+1 {
		return phoneCol + 1
	}
	return phoneCol
}

// firstNonPhoneCell returns the first cell in the row (other than phoneCol)
// that is non-empty and not phone-like; empty when none exists.
func firstNonPhoneCell(row []string, phoneCol int) string {
	for j, cell := range row {
		if j == phoneCol {
			continue
		}
		s := strings.TrimSpace(cell)
		if s != "" && !isPhoneLike(s) {
			return s
		}
	}
	return ""
}

// scanPhone returns the first phone-like cell in the row, skipping contentCol.
func scanPhone(row []string, contentCol int) (string, bool) {
	for j, cell := range row {
		if j == contentCol {
			continue
		}
		s := strings.TrimSpace(cell)
		if isPhoneLike(s) {
			return s, true
		}
	}
	return "", false
}

func cellAt(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func containsAnyWord(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func isBlankRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func isHeaderRow(row []string) bool {
	for _, cell := range row {
		s := strings.TrimSpace(cell)
		if s == "" {
			continue
		}
		low := strings.ToLower(s)
		for _, w := range headerWords {
			if low == w || strings.Contains(low, w) {
				return true
			}
		}
	}
	// A first-cell that contains letters and is not a phone-like value is a title.
	if len(row) > 0 {
		s := strings.TrimSpace(row[0])
		if s != "" && !isPhoneLike(s) {
			return true
		}
	}
	return false
}

// isPhoneLike reports whether a cell looks like a phone number rather than text.
func isPhoneLike(s string) bool {
	hasDigit := false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case r == '+' || r == '-' || r == ' ' || r == '(' || r == ')':
		default:
			return false
		}
	}
	return hasDigit
}
