package expires

import (
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/logs"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"math"
	"runtime"
	"testing"
	"time"
)

func TestList_Add(t *testing.T) {
	list := NewList()
	list.Add(1, time.Now().Unix())
	t.Log("===BEFORE===")
	logs.PrintAsJSON(list.expireMap, t)
	logs.PrintAsJSON(list.itemsMap, t)

	list.Add(1, time.Now().Unix()+1)
	list.Add(2, time.Now().Unix()+1)
	list.Add(3, time.Now().Unix()+2)
	t.Log("===AFTER===")
	logs.PrintAsJSON(list.expireMap, t)
	logs.PrintAsJSON(list.itemsMap, t)
}

func TestList_Add_Overwrite(t *testing.T) {
	var timestamp = time.Now().Unix()

	list := NewList()
	list.Add(1, timestamp+1)
	list.Add(1, timestamp+1)
	list.Add(1, timestamp+2)
	logs.PrintAsJSON(list.expireMap, t)
	logs.PrintAsJSON(list.itemsMap, t)

	var a = assert.NewAssertion(t)
	a.IsTrue(len(list.itemsMap) == 1)
	a.IsTrue(len(list.expireMap) == 1)
	a.IsTrue(list.itemsMap[1] == timestamp+2)
}

func TestList_Remove(t *testing.T) {
	list := NewList()
	list.Add(1, time.Now().Unix()+1)
	list.Remove(1)
	logs.PrintAsJSON(list.expireMap, t)
	logs.PrintAsJSON(list.itemsMap, t)
}

func TestList_GC(t *testing.T) {
	list := NewList()
	list.Add(1, time.Now().Unix()+1)
	list.Add(2, time.Now().Unix()+1)
	list.Add(3, time.Now().Unix()+2)
	list.GC(time.Now().Unix()+2, func(itemId int64) {
		t.Log("gc:", itemId)
	})
	logs.PrintAsJSON(list.expireMap, t)
	logs.PrintAsJSON(list.itemsMap, t)
}

func TestList_Start_GC(t *testing.T) {
	list := NewList()
	list.Add(1, time.Now().Unix()+1)
	list.Add(2, time.Now().Unix()+1)
	list.Add(3, time.Now().Unix()+2)
	list.Add(4, time.Now().Unix()+5)
	list.Add(5, time.Now().Unix()+5)
	list.Add(6, time.Now().Unix()+6)
	list.Add(7, time.Now().Unix()+6)
	list.Add(8, time.Now().Unix()+6)

	list.OnGC(func(itemId int64) {
		t.Log("gc:", itemId, timeutil.Format("H:i:s"))
		time.Sleep(2 * time.Second)
	})

	go func() {
		SharedManager.Add(list)
	}()

	time.Sleep(20 * time.Second)
}

func TestList_ManyItems(t *testing.T) {
	list := NewList()
	for i := 0; i < 1_000; i++ {
		list.Add(int64(i), time.Now().Unix())
	}
	for i := 0; i < 1_000; i++ {
		list.Add(int64(i), time.Now().Unix()+1)
	}

	now := time.Now()
	count := 0
	list.GC(time.Now().Unix()+1, func(itemId int64) {
		count++
	})
	t.Log("gc", count, "items")
	t.Log(time.Now().Sub(now))
}

func TestList_Map_Performance(t *testing.T) {
	t.Log("max uint32", math.MaxUint32)

	var timestamp = time.Now().Unix()

	{
		m := map[int64]int64{}
		for i := 0; i < 1_000_000; i++ {
			m[int64(i)] = timestamp
		}

		now := time.Now()
		for i := 0; i < 100_000; i++ {
			delete(m, int64(i))
		}
		t.Log(time.Now().Sub(now))
	}

	{
		m := map[uint64]int64{}
		for i := 0; i < 1_000_000; i++ {
			m[uint64(i)] = timestamp
		}

		now := time.Now()
		for i := 0; i < 100_000; i++ {
			delete(m, uint64(i))
		}
		t.Log(time.Now().Sub(now))
	}

	{
		m := map[uint32]int64{}
		for i := 0; i < 1_000_000; i++ {
			m[uint32(i)] = timestamp
		}

		now := time.Now()
		for i := 0; i < 100_000; i++ {
			delete(m, uint32(i))
		}
		t.Log(time.Now().Sub(now))
	}
}

func Benchmark_Map_Uint64(b *testing.B) {
	runtime.GOMAXPROCS(1)
	var timestamp = uint64(time.Now().Unix())

	var i uint64
	var count uint64 = 1_000_000

	m := map[uint64]uint64{}
	for i = 0; i < count; i++ {
		m[i] = timestamp
	}

	for n := 0; n < b.N; n++ {
		for i = 0; i < count; i++ {
			_ = m[i]
		}
	}
}
