package cmd

import "testing"

func TestCSVtoEndpoints(t *testing.T) {
	type Case struct {
		csv      string
		err      bool
		expected []string
	}

	cs := []Case{
		Case{
			"8.8.8.8:53,8.8.4.4:8053",
			false,
			[]string{"8.8.8.8:53", "8.8.4.4:8053"},
		},
		Case{
			"8.8.8.8,8.8.4.4:8053",
			false,
			[]string{"8.8.8.8:53", "8.8.4.4:8053"},
		},
		Case{
			"8.8.8.8",
			false,
			[]string{"8.8.8.8:53"},
		},
		Case{
			"",
			false,
			[]string{},
		},
		Case{
			"8.8.8.8:53:54",
			true,
			[]string{},
		},
	}

	for i, c := range cs {
		results, err := CSVtoEndpoints(c.csv)
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
