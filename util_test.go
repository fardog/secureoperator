package main

import "testing"

func TestRandSeq(t *testing.T) {
	var s string

	s = GenerateUrlSafeString(10)

	if len(s) != 10 {
		t.Errorf("expected string length 10, got %v", len(s))
	}

	s = GenerateUrlSafeString(20)

	if len(s) != 20 {
		t.Errorf("expected strig length 20, got %v", len(s))
	}
}

func TestCSVtoEndpoints(t *testing.T) {
	type Case struct {
		csv      string
		err      bool
		expected []string
	}
}

func TestCSVtoIPs(t *testing.T) {
	type Case struct {
		csv      string
		err      bool
		expected []string
	}

	cs := []Case{
		Case{
			"8.8.8.8,8.8.4.4",
			false,
			[]string{"8.8.8.8", "8.8.4.4"},
		},
		Case{
			"8.8.8.8,8.8.4.4",
			false,
			[]string{"8.8.8.8", "8.8.4.4"},
		},
		Case{
			"8.8.8.8",
			false,
			[]string{"8.8.8.8"},
		},
		Case{
			"",
			false,
			[]string{},
		},
		Case{
			"8.8.8.8:53",
			true,
			[]string{},
		},
	}

	for i, c := range cs {
		results, err := CSVtoIPs(c.csv)
		if c.err && err == nil {
			t.Errorf("%v: expected err, got none", i)
		} else if !c.err && err != nil {
			t.Errorf("%v: did not expect error, got: %v", i, err)
		}

		if e, r := len(c.expected), len(results); e != r {
			t.Errorf("%v: expected %v results, got %v", i, e, r)
			continue
		}

		for j, r := range results {
			if r.String() != c.expected[j] {
				t.Errorf("%v,%v: expected %v, got %v", i, j, r, c.expected[j])
			}
		}
	}
}

func TestKeyValue(t *testing.T) {
	kv := make(KeyValue)

	kv.Set("key1=value=value")
	kv.Set("key2=value2-1")
	kv.Set("key2=value2-2")

	if vs, ok := kv["key1"]; ok {
		if len(vs) == 1 {
			if vs[0] != "value=value" {
				t.Errorf("unexpected value %v", vs[0])
			}
		} else {
			t.Errorf("expected key1 value to be length 1, saw %v", vs)
		}
	} else {
		t.Error("did not get value for key1")
	}

	if vs, ok := kv["key2"]; ok {
		if len(vs) == 2 {
			if vs[0] != "value2-1" {
				t.Errorf("unexpected value %v", vs[0])
			}
			if vs[1] != "value2-2" {
				t.Errorf("unexpected value %v", vs[1])
			}
		} else {
			t.Errorf("unexpected length for %v", vs)
		}
	} else {
		t.Error("did not get value for key2")
	}
}