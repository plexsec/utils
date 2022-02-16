package stat

import (
	"math/rand"
	//"sync"
	"fmt"
	"time"
	//"sync"
	//"sync/atomic"
	"testing"
)

func TestCPU(t *testing.T) {
	if err := AttrInit("10.2.122.58:10185", "", "qc-tick"); err != nil {
		t.Fatal(err)
	}

	SetMemoryStat("write")
	SetCpuStat("write")
	SetHeapStat("write")

	time.Sleep(1000 * time.Second)
}

/*
func TestAttr(t *testing.T) {
	if err := AttrInit("10.2.122.58:20090", "t", "test"); err != nil {
		t.Fatal(err)
	}
	SetAttr("test.accept", 1)
	SetAttr("test.del", 1)
	SetAttr("test.tick", 100)

	bg := SetAttrDura("test.time", 0)
	time.Sleep(1 * time.Second)
	SetAttrDura("test.time", bg)

	time.Sleep(2 * time.Second)
	t.Log("test end")

	var wg sync.WaitGroup
	bt := time.Now().UnixNano()
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < 1000000; j++ {
				SetAttr("test.accept", j)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	et := time.Now().UnixNano()
	fmt.Printf("all:%d num:%d  %d ns/op", et-bt, 10*1000000, (et-bt)/(10*1000000))
}

func BenchmarkAttr(b *testing.B) {
	if err := AttrInit("10.2.122.58:20090", "t", "test"); err != nil {
		return
	}
	for i := 0; i < b.N; i++ {
		SetAttr("test.accept", i)
	}
}

func BenchmarkAttrDura(b *testing.B) {
	if err := AttrInit("10.2.122.58:20090", "t", "test"); err != nil {
		return
	}
	for i := 0; i < b.N; i++ {
		bg := SetAttrDura("test.accept", 0)
		SetAttrDura("test.accept", bg-int64(i))
	}
}
*/
func BenchmarkAttrParallel(b *testing.B) {
	if err := AttrInit("10.2.122.58:20190", "t", "test"); err != nil {
		return
	}
	for i := 0; i < 1000; i++ {
		SetAttr("test.accept"+fmt.Sprintf("%d", i), 1)
	}
	b.RunParallel(func(pb *testing.PB) {
		rand.Seed(time.Now().UnixNano())
		k := rand.Int31n(1000)
		for pb.Next() {
			SetAttr("test.accept"+fmt.Sprintf("%d", k%1000), 1)
			k++
		}
	})
}

/*
func BenchmarkLock(b *testing.B) {
	var mn = map[int]int{}
	for i := 1; i < 1000; i++ {
		mn[i] = 1
	}
	var s int
	var mu sync.Mutex
	for i := 0; i < b.N; i++ {
		mu.Lock()
		s = mn[i]
		i++
		s = s / 10
		mu.Unlock()
	}
}
func BenchmarkCAS(b *testing.B) {
	var s uint32
	for i := 0; i < b.N; i++ {
		a := s
		i++
		atomic.CompareAndSwapUint32(&s, a, uint32(i))
	}
}
func BenchmarkAtomic(b *testing.B) {
	var s uint32
	for i := 0; i < b.N; i++ {
		i++
		atomic.AddUint32(&s, uint32(i))
	}
}
func BenchmarkLockParallel(b *testing.B) {
	var mn = map[int]int{}
	for i := 1; i < 1000; i++ {
		mn[i] = 1
	}

	var s int
	var mu sync.Mutex
	b.RunParallel(func(pb *testing.PB) {
		i := int(0)
		for pb.Next() {
			mu.Lock()
			s = mn[i]
			i++
			mu.Unlock()
		}
	})
}
func BenchmarkCASParallel(b *testing.B) {
	var s uint32
	b.RunParallel(func(pb *testing.PB) {
		i := uint32(0)
		for pb.Next() {
			a := s
			i++
			atomic.CompareAndSwapUint32(&s, a, i)
		}
	})
}
func BenchmarkAtomicParallel(b *testing.B) {
	var s uint32
	b.RunParallel(func(pb *testing.PB) {
		i := uint32(0)
		for pb.Next() {
			atomic.AddUint32(&s, i)
			i++
		}
	})
}
*/
