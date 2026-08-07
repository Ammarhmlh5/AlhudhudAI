package excelimport

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
)

func TestParseCSV(t *testing.T) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"رقم الهاتف", "نص الرسالة"})
	_ = w.Write([]string{"0551234567", "مرحبا"})
	_ = w.Write([]string{"", ""})
	_ = w.Write([]string{"966512345678", "رسالة ثانية"})
	_ = w.Write([]string{"بيانات خاطئة", ""})
	w.Flush()

	rows, err := Parse(&buf, ".csv")
	if err != nil {
		t.Fatalf("Parse csv: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 data rows, got %d: %+v", len(rows), rows)
	}
	if rows[0].Phone != "0551234567" || rows[0].Content != "مرحبا" {
		t.Errorf("row0 mismatch: %+v", rows[0])
	}
	if rows[1].Phone != "966512345678" {
		t.Errorf("row1 mismatch: %+v", rows[1])
	}
	if rows[2].Phone != "بيانات خاطئة" {
		t.Errorf("row2 mismatch: %+v", rows[2])
	}
}

func TestParseCSV_BOM(t *testing.T) {
	data := append([]byte{0xEF, 0xBB, 0xBF}, []byte("0551234567,مرحبا\n966512345678,ثانية\n")...)
	rows, err := Parse(bytes.NewReader(data), "csv")
	if err != nil {
		t.Fatalf("Parse csv with BOM: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Phone != "0551234567" {
		t.Errorf("BOM row0 mismatch: %+v", rows[0])
	}
}

func TestParseUnsupported(t *testing.T) {
	if _, err := Parse(strings.NewReader("x"), ".xls"); err == nil {
		t.Fatal("expected error for .xls")
	}
}

func TestParseXlsx(t *testing.T) {
	f, err := newTestXlsx([][]string{
		{"رقم الهاتف", "الرسالة"},
		{"0551234567", "نص تجريبي ١"},
		{"966512345678", "نص تجريبي ٢"},
		{"invalid", "نص بدون رقم"},
	})
	if err != nil {
		t.Fatalf("build xlsx: %v", err)
	}
	rows, err := Parse(bytes.NewReader(f), ".xlsx")
	if err != nil {
		t.Fatalf("Parse xlsx: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d: %+v", len(rows), rows)
	}
	if rows[0].Phone != "0551234567" || rows[1].Phone != "966512345678" {
		t.Errorf("xlsx rows mismatch: %+v", rows)
	}
}

func TestParseCSV_ThreeColumnsWithName(t *testing.T) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"الاسم", "رقم الجوال", "الرسالة"})
	_ = w.Write([]string{"أحمد", "0501234567", "مرحبا أحمد"})
	_ = w.Write([]string{"سارة", "966509876543", "مرحبا سارة"})
	_ = w.Write([]string{"خالد", "0581122334", "مرحبا خالد"})
	w.Flush()

	rows, err := Parse(&buf, ".csv")
	if err != nil {
		t.Fatalf("Parse csv: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 data rows, got %d: %+v", len(rows), rows)
	}
	if rows[0].Phone != "0501234567" || rows[0].Content != "مرحبا أحمد" {
		t.Errorf("row0 mismatch: %+v", rows[0])
	}
	if rows[1].Phone != "966509876543" || rows[1].Content != "مرحبا سارة" {
		t.Errorf("row1 mismatch: %+v", rows[1])
	}
	if rows[2].Phone != "0581122334" || rows[2].Content != "مرحبا خالد" {
		t.Errorf("row2 mismatch: %+v", rows[2])
	}
}

func TestParseCSV_PhoneInSecondColumn(t *testing.T) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"اسم العميل", "mobile", "نص"})
	_ = w.Write([]string{"علي", "0505556666", "رسالة علي"})
	w.Flush()

	rows, err := Parse(&buf, ".csv")
	if err != nil {
		t.Fatalf("Parse csv: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d: %+v", len(rows), rows)
	}
	if rows[0].Phone != "0505556666" || rows[0].Content != "رسالة علي" {
		t.Errorf("row0 mismatch: %+v", rows[0])
	}
}

func TestParseCSV_TitleThenHeader(t *testing.T) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"قائمة عملاء شركة الهدهد"})
	_ = w.Write([]string{"الاسم", "الجوال", "الرسالة"})
	_ = w.Write([]string{"محمد", "0551112223", "أهلا محمد"})
	w.Flush()

	rows, err := Parse(&buf, ".csv")
	if err != nil {
		t.Fatalf("Parse csv: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 data row, got %d: %+v", len(rows), rows)
	}
	if rows[0].Phone != "0551112223" || rows[0].Content != "أهلا محمد" {
		t.Errorf("row0 mismatch: %+v", rows[0])
	}
}

func TestParseXlsx_ThreeColumns(t *testing.T) {
	f, err := newTestXlsx([][]string{
		{"الاسم", "رقم الجوال", "الرسالة"},
		{"أحمد", "0551234567", "نص تجريبي ١"},
		{"سارة", "966512345678", "نص تجريبي ٢"},
	})
	if err != nil {
		t.Fatalf("build xlsx: %v", err)
	}
	rows, err := Parse(bytes.NewReader(f), ".xlsx")
	if err != nil {
		t.Fatalf("Parse xlsx: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d: %+v", len(rows), rows)
	}
	if rows[0].Phone != "0551234567" || rows[0].Content != "نص تجريبي ١" {
		t.Errorf("row0 mismatch: %+v", rows[0])
	}
	if rows[1].Phone != "966512345678" || rows[1].Content != "نص تجريبي ٢" {
		t.Errorf("row1 mismatch: %+v", rows[1])
	}
}

func TestStoreSetGetExpiry(t *testing.T) {
	s := NewStore()
	p := s.Set(Preview{FileName: "a.csv", Valid: 1, Rows: []PreviewRow{{Phone: "+9665", Content: "x", Valid: true}}})
	if p.Token == "" || !strings.HasPrefix(p.Token, "ex_") {
		t.Fatalf("bad token: %q", p.Token)
	}
	got, ok := s.Get(p.Token)
	if !ok || got.FileName != "a.csv" {
		t.Fatalf("Get failed: %+v", got)
	}
	s.Delete(p.Token)
	if _, ok := s.Get(p.Token); ok {
		t.Fatal("expected deleted preview to be gone")
	}
	if _, ok := s.Get("ex_missing"); ok {
		t.Fatal("expected missing token to be gone")
	}
}
