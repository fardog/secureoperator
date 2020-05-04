package dohProxy

import (
	"fmt"
	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/miekg/dns"
	"sync"
	"time"
)

const (
	// Header.Bits :
	//_QR = 1 << 15 // query/response (response=1)
	//_AA = 1 << 10 // authoritative
	//_TC = 1 << 9  // truncated
	//_RD = 1 << 8  // recursion desired
	//_RA = 1 << 7  // recursion available
	//_Z  = 1 << 6  // Z
	//_AD = 1 << 5  // authenticated data
	//_CD = 1 << 4  // checking disabled

	// Question :
	// Name   string
	// Qtype  uint16
	// Qclass uint16
	queryFormatString string = "[OPCODE:%v][TC:%v][RD:%v][Z:%v][CD:%v][QName:%v]" +
		"[QType:%v][QClass:%v][EDNS0Subnet:%v]"
)

// Use map to store cache, red-black tree to index cache.
// red-black tree also used to implement the cache expire mechanism.
type Cache struct {
	nextExpireTime int64
	cacheStore     map[string]*cacheItem
	cacheReg       *RedBlackTreeExtended
	lock           sync.RWMutex
}

type cacheItem struct {
	TimeArrival int64
	MsgBytes    []byte
}

type cacheEntry struct {
	// key for cacheStore
	Key        string
	TimeExpire int64
}

func NewCache() *Cache {
	cache := &Cache{
		// init a max int64, to let any other expire time rewrite it.
		nextExpireTime: int64(^uint64(0) >> 1),
		cacheStore:     make(map[string]*cacheItem),
		cacheReg: &RedBlackTreeExtended{rbt.NewWith(
			func(a, b interface{}) int {
				if a == b {
					return 0
				}
				diff := a.(int64) - b.(int64)
				switch diff > 0 {
				case true:
					return 1
				case false:
					return -1
				default:
					return 0
				}
			},
		)},
	}
	go cache.expire()
	return cache
}

func (c *Cache) expire() {
	// infinite loop
	for c.cacheReg != nil {
		Log.Debugf("will drop cache on: %v, current cache size: %v", c.nextExpireTime, c.cacheReg.Size())
		if time.Now().Unix() > c.nextExpireTime {
			c.doExpire()
			Log.Debugf("cache expire scheduled, current cache size: %v", c.cacheReg.Size())
		}
		time.Sleep(2 * time.Second)
	}
}

func (c *Cache) doExpire(/*notify chan bool*/) {
	now := time.Now().Unix()
	for hang, found := c.cacheReg.GetMin()
		found && now > hang.(cacheEntry).TimeExpire; {

		hangEntry := hang.(cacheEntry)
		c.expireAction(hangEntry)

		hang, found = c.cacheReg.GetMin()
		if found {
			c.nextExpireTime = hang.(cacheEntry).TimeExpire
		}
		Log.Debugf("set next expire time %v, current cache size: %v",c.nextExpireTime, c.cacheReg.Size())
	}
	Log.Infof("current cache size: %v", c.cacheReg.Size())
}

func (c *Cache) expireAction(entry cacheEntry) {
	c.lock.Lock()
	defer c.lock.Unlock()
	Log.Debugf("cache dropping : %v", entry)
	c.cacheReg.Remove(entry.TimeExpire)
	delete(c.cacheStore, entry.Key)
	Log.Debugf("cache dropped once, cache size: %v", c.cacheReg.Size())
}

func (c *Cache) Insert(msgCh <-chan *dns.Msg) {
	msg := <-msgCh
	c.realInsert(msg)
}

func (c *Cache) realInsert(msg *dns.Msg) {
	qStr := getQueryStringForCache(msg)
	Log.Debugf("start insert cache: \n%v \n <= \n %v", qStr, msg)
	now := time.Now().Unix()
	minTTL := GetMinTTLFromDnsMsg(msg)
	bytesMsg, err := msg.Pack()
	if err != nil {
		Log.Errorf("can't pack dns-message: %v", err)
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	c.cacheStore[qStr] = &cacheItem{TimeArrival: now, MsgBytes: bytesMsg}
	// use minimal ttl in dns-message to expire early.
	expireTime := now + int64(minTTL)
	c.cacheReg.Put(expireTime,
		cacheEntry{
			Key: qStr, TimeExpire: expireTime,
		})
	if c.cacheReg.Size() < 100 {
		Log.Debugf("cache registry: %v", c.cacheReg)
	}
	if expireTime < c.nextExpireTime {
		c.nextExpireTime = expireTime
		Log.Debugf("next cache entry expire on: %v <= %vs",
			c.nextExpireTime, time.Now().Unix()-c.nextExpireTime)
	}
}

func (c *Cache) Get(msgQ *dns.Msg) (rMsg *dns.Msg) {
	qStr := getQueryStringForCache(msgQ)

	c.lock.RLock()
	defer c.lock.RUnlock()

	cacheRet := c.cacheStore[qStr]
	if cacheRet == nil || cacheRet.MsgBytes == nil {
		return nil
	}
	cacheArrivalTime := cacheRet.TimeArrival
	msgRet := new(dns.Msg)
	err := msgRet.Unpack(cacheRet.MsgBytes)
	if err != nil {
		Log.Errorf("can't unpack dns-message: %v", err)
		return nil
	}
	Log.Debugf("cache query result: \n%v \n => cacheArrivalTime: %v\n %v", qStr, cacheArrivalTime, msgRet)
	// recalculate ttl.
	for _, rs :=
	range [][]dns.RR{msgRet.Answer, msgRet.Ns} {
		for _, r := range rs {
			rh := r.Header()
			ttlNew := cacheArrivalTime + int64(rh.Ttl) - time.Now().Unix()
			if ttlNew <= 0 {
				return nil
			}
			rh.Ttl = uint32(ttlNew)
		}
	}
	return msgRet
}

func getQueryStringForCache(msg *dns.Msg) (q string) {
	if msg.Question == nil || len(msg.Question) == 0 {
		return ""
	}
	edns0Subnet := ""
	subnet := ObtainEDN0Subnet(msg)
	if subnet.Address != nil {
		edns0Subnet = subnet.String()
	}

	queryStr := fmt.Sprintf(queryFormatString,
		msg.Opcode, msg.Truncated, msg.RecursionDesired, msg.Zero, msg.CheckingDisabled,
		dns.CanonicalName(msg.Question[0].Name), msg.Question[0].Qtype, msg.Question[0].Qclass,
		edns0Subnet)
	Log.Debugf("cache query string: %v", queryStr)
	return queryStr
}
