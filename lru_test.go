package lru

import (
	"fmt"
	"testing"
	"time"
)

//2021/07/12 10:38:15 Alloc:268648(bytes) Sys:74531848(bytes) HeapObjects:3499(bytes) HeapInuse:704512(bytes)

func TestCache_StoreOrUpdate(t *testing.T) {
	cache := NewCache(10)
	cache.memyinfo()

	for i := 1; i <= 10; i++ {
		cache.StoreOrUpdate("qmjz", int64(i), "zcjb", []byte("33333333333333"))
		fmt.Println("store")
		time.Sleep(1*time.Second)
	}

	cache.dump()


	fmt.Println(cache.Get("qmjz",5,"zcjb"))
	fmt.Println(cache.Get("qmjz",6,"zcjb"))


	for range time.Tick(5 * time.Second) {
		cache.dump()
	}
}
