package lru

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

type ConfigNode struct {
	Key     string
	GroupId int64
	AppName string
	Content []byte

	PreNode  *ConfigNode
	NextNode *ConfigNode
}

type Cache struct {
	cache                sync.Map
	lastTimeLinkListHead *ConfigNode
	linkListChannel      chan *ConfigNode
	maxSize              int64
}

func NewCache(maxSize int64) *Cache {
	c := &Cache{
		cache:                sync.Map{},
		lastTimeLinkListHead: nil,
		linkListChannel:      make(chan *ConfigNode, maxSize),
		maxSize:              maxSize,
	}
	//维护链表
	go c.updateLinkLish()
	//调用gc
	go c.gc()
	return c
}

func (c *Cache) Get(key string, groupId int64, appName string) (content []byte, err error) {
	value, ok := c.cache.Load(c.cacheHashKey(key, groupId, appName))
	if !ok {
		return nil, ErrNotFound
	}
	node, ok := value.(*ConfigNode)
	if !ok {
		return nil, ErrValueType
	}
	//放进维护链表的channel
	c.linkListChannel <- node
	content = node.Content
	return
}

func (c *Cache) StoreOrUpdate(key string, groupId int64, appName string, content []byte) (err error) {
	var node *ConfigNode

	cachekey := c.cacheHashKey(key, groupId, appName)
	value, ok := c.cache.Load(cachekey)
	if !ok {
		//不存在新存储
		node = &ConfigNode{
			Key:     key,
			GroupId: groupId,
			AppName: appName,
			Content: content,
		}
	} else {
		node = value.(*ConfigNode)
		node.Content = content
	}
	//放进维护链表的channel
	c.linkListChannel <- node
	//更新缓存
	c.cache.Store(cachekey, node)
	return
}

//异步维护链表
func (c *Cache) updateLinkLish() {
	for {
		select {
		case node := <-c.linkListChannel:
			c.removeToLinkListHead(node)
		}
	}
}
func (c *Cache) removeToLinkListHead(node *ConfigNode) {
	//是不是第一个
	if c.lastTimeLinkListHead == nil {
		c.lastTimeLinkListHead = node
		return
	}
	//已经存在,把当前node指向当前头结点
	head := c.lastTimeLinkListHead
	head.PreNode = node

	if node.PreNode != nil {
		node.PreNode.NextNode = node.NextNode
		node.PreNode = nil
	}
	node.NextNode = head
	c.lastTimeLinkListHead = node
}

//每10s触发一次gc
func (c *Cache) gc() {
	tick := time.Tick(10 * time.Second)
	for {
		select {
		case <-tick:
			fmt.Println("start gc...")
			node := c.lastTimeLinkListHead
			var num int64
			for node != nil {
				num++
				//超过了限制
				if num > c.maxSize {
					//fmt.Println("del node: ",node)
					if node.PreNode != nil {
						node.PreNode.NextNode = nil
						node.PreNode = nil
					}
					c.cache.Delete(c.cacheHashKey(node.Key, node.GroupId, node.AppName))
				}
				node = node.NextNode
			}
			fmt.Println("current length:", num)
			c.memyinfo()
		}
	}
}

func (c *Cache) cacheHashKey(key string, groupId int64, appName string) string {
	return fmt.Sprintf("%s_%d_%s", key, groupId, appName)
}

//测试方法
func (c *Cache) memyinfo() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	log.Printf("Alloc:%d(bytes) Sys:%d(bytes) HeapObjects:%d(bytes) HeapInuse:%d(bytes)", ms.Alloc, ms.Sys, ms.HeapObjects, ms.HeapInuse)
}
func (c *Cache) dump() {
	node := c.lastTimeLinkListHead
	for node != nil {
		fmt.Printf("%p -> ", node)
		fmt.Println(node)
		node = node.NextNode
	}

	c.cache.Range(func(key, value interface{}) bool {
		fmt.Println(key, value)
		return true
	})
}
