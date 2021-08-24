package lru

import (
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type NodeKey struct {
	Key     string
	GroupId int64
	AppName string
}

type ConfigNode struct {
	Key 	NodeKey
	Content []byte

	PreNode  *ConfigNode
	NextNode *ConfigNode

	visitTimeUnix int64
}

type Cache struct {
	cache                sync.Map
	linkListHead *ConfigNode
	linkListChannel      chan *ConfigNode
	maxSize              int64
}

func NewCache(maxSize int64) *Cache {
	c := &Cache{
		cache:                sync.Map{},
		linkListHead: nil,
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
			Key: NodeKey{
				Key:     key,
				GroupId: groupId,
				AppName: appName,
			},
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

//获取所有的data
func (c *Cache) AllData() (data []ConfigNode) {
	c.cache.Range(func(key, value interface{}) bool {
		data = append(data, *(value.(*ConfigNode)))
		return true
	})
	return
}


//异步维护链表
func (c *Cache) updateLinkLish() {
	for {
		select {
		case node := <-c.linkListChannel:
			//更新访问时间
			node.visitTimeUnix = time.Now().Unix()
			//移到队列头部
			c.removeToLinkListHead(node)
		}
	}
}
func (c *Cache) removeToLinkListHead(node *ConfigNode) {
	//是不是第一个
	if c.linkListHead == nil {
		c.linkListHead = node
		return
	}
	//当前head==node不移动
	if c.linkListHead == node {
		return
	}

	//已经存在,把当前node指向当前头结点
	head := c.linkListHead
	head.PreNode = node

	if node.PreNode != nil {
		node.PreNode.NextNode = node.NextNode
		node.PreNode = nil
	}
	node.NextNode = head
	c.linkListHead = node
}

//每10s触发一次gc
func (c *Cache) gc() {
	tick := time.Tick(30 * time.Minute)
	for {
		select {
		case <-tick:
			node := c.linkListHead
			var num int64
			lastUnix := time.Now().AddDate(0,0,-3).Unix()
			//lastUnix := time.Now().Unix() - 60

			//遍历链表
			for node != nil {
				num++
				//1. 超过了限制后面的删掉
				//2. 最后使用时间是三天前也删掉
				if num > c.maxSize || node.visitTimeUnix < lastUnix {
					//删除node操作
					if node.PreNode != nil {
						node.PreNode.NextNode = nil
						node.PreNode = nil
					}else{
						//node.PreNode==nil 说明当前是第一个节点
						c.linkListHead = node.NextNode
					}
					c.cache.Delete(c.cacheHashKey(node.Key.Key, node.Key.GroupId, node.Key.AppName))
				}
				node = node.NextNode
			}
			log.Println("start gc, current length:", num)
			c.memyinfo()
		}
	}
}

//处理cache key
func (c *Cache) cacheHashKey(key string, groupId int64, appName string) string {
	return fmt.Sprintf("%s_%d_%s", key, groupId, appName)
}
func (c *Cache) cacheSplitKey(key string) NodeKey {
	ks := strings.Split(key,"_")
	gid,_ := strconv.ParseInt(ks[1],10,64)
	return NodeKey{
		Key:     ks[0],
		GroupId: gid,
		AppName: ks[2],
	}
}



//测试方法
func (c *Cache) memyinfo() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	log.Printf("Alloc:%d(bytes) Sys:%d(bytes) HeapObjects:%d(bytes) HeapInuse:%d(bytes)", ms.Alloc, ms.Sys, ms.HeapObjects, ms.HeapInuse)
}
func (c *Cache) dump() {
	node := c.linkListHead
	for node != nil {
		fmt.Printf("%p -> ", node)
		fmt.Println(node)
		node = node.NextNode
	}
	c.cache.Range(func(key, value interface{}) bool {
		fmt.Println(key,value)
		return true
	})
}
