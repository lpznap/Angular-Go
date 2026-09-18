package report

import (
	"bytes"
	"dailyworknotes/internal/domain"
	"encoding/csv"
	"strings"
	"testing"
)

func TestCSVFormulaSafetyAndThai(t *testing.T) {
	b, e := CSV([]domain.Note{{Title: "=cmd()", Project: " @SUM(1)", Description: "<p>ภาษาไทย</p>", Minutes: 65}})
	if e != nil {
		t.Fatal(e)
	}
	rows, e := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(b), "\ufeff"))).ReadAll()
	if e != nil {
		t.Fatal(e)
	}
	if rows[1][2] != "'=cmd()" || rows[1][3] != "' @SUM(1)" || rows[1][4] != "ภาษาไทย" {
		t.Fatalf("bad CSV: %v", rows)
	}
}
func TestReportEscapesContent(t *testing.T) {
	b, e := HTML([]domain.Note{{Title: "<script>x</script>", Description: "<p>ภาษาไทย</p><script>bad()</script>", Minutes: 65}})
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Contains(b, []byte("<script>")) || !bytes.Contains(b, []byte("ภาษาไทย")) || !bytes.Contains(b, []byte("1h 05m")) {
		t.Fatal(string(b))
	}
}
