package domain

import "testing"

func TestValidationAndSanitization(t *testing.T) {
	n := Note{WorkDate: "2026-09-18", Title: "Thai งานวันนี้", Status: "progress", Priority: "medium", Description: `<p>Hello <strong>งาน</strong><script>alert(1)</script><img src=x onerror=alert(1)></p>`}
	if e := n.Validate(); e != nil {
		t.Fatal(e)
	}
	if n.Description != `<p>Hello <strong>งาน</strong></p>` {
		t.Fatalf("unsafe HTML: %s", n.Description)
	}
	n.Minutes = 1441
	if n.Validate() == nil {
		t.Fatal("accepted excessive time")
	}
	n.Minutes = 1
	n.WorkDate = "2026-02-30"
	if n.Validate() == nil {
		t.Fatal("accepted invalid date")
	}
}

func TestStatusMustBeOneEnumValue(t *testing.T) {
	n:=Note{WorkDate:"2026-09-18",Title:"Valid",Status:"todo|progress",Priority:"low"}
	if n.Validate()==nil {t.Fatal("accepted combined status values")}
}
