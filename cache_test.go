package main

import (
	"fmt"
	"github.com/miekg/dns"
	"testing"
	"time"
)

func BenchmarkGetQueryStringForCache(b *testing.B) {
	msg := new(dns.Msg)
	msg.SetQuestion("google.com", dns.TypeAAAA)
	ednsSubnet := &dns.EDNS0_SUBNET{
		Address:       parseIPv4("114.114.114.114"),
		SourceNetmask: 24,
	}
	msg.SetEdns0(ednsSubnet.Option(), true)
	for i := 0; i < b.N; i++ {
		getQueryStringForCache(msg)
	}
}

func TestCache_Get(t *testing.T) {
	msg := new(dns.Msg)
	nonce := time.Now().UnixNano()
	ednsSubnet := &dns.EDNS0_SUBNET{
		Address:       parseIPv4("114.114.114.114"),
		SourceNetmask: 24,
	}
	msg.SetEdns0(ednsSubnet.Option(), true)
	msg.SetQuestion(dns.CanonicalName(fmt.Sprintf("google%v.com", nonce)), dns.TypeAAAA)

	msgR := new(dns.Msg)
	msgR.SetReply(msg)

	cache := NewCache()
	cache.realInsert(msgR)
	t.Logf("message response: \n%v", msgR)
	msgC := cache.Get(msg)
	t.Logf("message get from cache: \n%v", msgC)
	if msgC.Question == nil || len(msgC.Question) == 0 {
		t.Errorf("message get from cache should have Q: \n%v", msgC)
	}
	if msgC.Question[0] != msgR.Question[0]{
		t.Errorf("message get from cache should have the same question: \n%v",msgC)
	}
}

func BenchmarkCache_Insert(b *testing.B) {
	msg := new(dns.Msg)
	ednsSubnet := &dns.EDNS0_SUBNET{
		Address:       parseIPv4("114.114.114.114"),
		SourceNetmask: 24,
	}
	msg.SetEdns0(ednsSubnet.Option(), true)

	msgR := new(dns.Msg)
	msgR.SetReply(msg)

	cache := NewCache()
	for i := 0; i < b.N; i++ {
		msgR.SetQuestion(dns.CanonicalName(fmt.Sprintf("google%v.com", time.Now().UnixNano())), dns.TypeAAAA)
		cache.realInsert(msgR)
	}
}

func BenchmarkCache_Get(b *testing.B) {
	msg := new(dns.Msg)
	ednsSubnet := &dns.EDNS0_SUBNET{
		Address:       parseIPv4("114.114.114.114"),
		SourceNetmask: 24,
	}
	msg.SetEdns0(ednsSubnet.Option(), true)

	msgR := new(dns.Msg)

	cache := NewCache()
	for i := 0; i < b.N; i++ {
		nonce := time.Now().UnixNano()
		msg.SetQuestion(dns.CanonicalName(fmt.Sprintf("google%v.com", nonce)), dns.TypeAAAA)
		msgR.SetReply(msg)
		cache.realInsert(msgR)
		rMsg := cache.Get(msg)
		if rMsg.Question[0].Name != msgR.Question[0].Name{
			b.Errorf("cache result is not valid.")
		}
	}
}
