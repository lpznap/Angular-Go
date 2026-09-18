package storage

import "testing"

func TestFileValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		b    []byte
		ok   bool
	}{{"งาน.txt", []byte("งานวันนี้"), true}, {"fake.png", []byte("text"), false}, {"../file.txt", []byte("abc"), false}, {"script.html", []byte("<script>"), false}, {"empty.txt", nil, false}, {"too-big.txt", make([]byte, 11), false}} {
		_, e := Validate(tc.name, tc.b, 10)
		if tc.name == "งาน.txt" {
			_, e = Validate(tc.name, tc.b, 100)
		}
		if (e == nil) != tc.ok {
			t.Errorf("%s: %v", tc.name, e)
		}
	}
}
func TestPersistence(t *testing.T) {
	s := Store{Root: t.TempDir(), Limit: 100}
	key := "0123456789abcdef0123456789abcdef"
	if e := s.Put(key, []byte("hello")); e != nil {
		t.Fatal(e)
	}
	b, e := s.Read(key)
	if e != nil || string(b) != "hello" {
		t.Fatal("stored file mismatch")
	}
	if e = s.Remove(key); e != nil {
		t.Fatal(e)
	}
	if e = s.Remove(key); e != nil {
		t.Fatal("delete must be idempotent")
	}
}
