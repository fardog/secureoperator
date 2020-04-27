package main

import (
	"net"
	"testing"
)

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

func TestCSVtoIPs(t *testing.T) {
	type Case struct {
		csv      string
		err      bool
		expected []string
	}

	cs := []Case{
		{
			"8.8.8.8,8.8.4.4",
			false,
			[]string{"8.8.8.8", "8.8.4.4"},
		},
		{
			"8.8.8.8,8.8.4.4",
			false,
			[]string{"8.8.8.8", "8.8.4.4"},
		},
		{
			"8.8.8.8",
			false,
			[]string{"8.8.8.8"},
		},
		{
			"",
			false,
			[]string{},
		},
		{
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

	_ = kv.Set("key1=value=value")
	_ = kv.Set("key2=value2-1")
	_ = kv.Set("key2=value2-2")

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

func TestIpv46Cast(t *testing.T) {

	ip4Str := "192.168.31.1"
	ip16Str := "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
	ip16StrExpect := "2001:db8:85a3::8a2e:370:7334"

	ip4 := net.ParseIP(ip4Str)
	ip16 := net.ParseIP(ip16Str)

	ip4to16 := ip4.To16()
	ip16to4 := ip16.To4()

	if ip4Str != ip4.String() {
		t.Errorf("unexpected convert ip4: %v > %v", ip4Str, ip4)
	}

	if ip16StrExpect != ip16.String() {
		t.Errorf("unexpected convert ip16: %v > %v", ip16StrExpect, ip16)
	}

	if ip4to16.String() != ip4.String() {
		t.Errorf("unexpected convert ip4 to ip16: %v > %v", ip4to16, ip4)
	}

	if ip16to4 != nil {
		t.Errorf("unexpected convert ip16 to ip4: %v > %v", ip16to4, ip4)
	}
}

func TestIfLocalAddr(t *testing.T){
	addrs := []string{
		"127.0.0.1",
		"0.0.0.0",
		"::1",
		"::",
		"localhost",
		"192.168.31.1",
	}

	ports := []string {
		"0",
		"53",
		"",
	}

	for _, addr := range addrs{
		for _, port := range ports{
			network := net.JoinHostPort(addr, port)
			result := IsLocalListen(network)
			if addr == "192.168.31.1" && result != false {
				t.Errorf("unexpected result for %v : %v", network, result)
			}
		}
	}
}