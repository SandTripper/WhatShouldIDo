package mcache

import (
	"container/list"
	"errors"
	"sync"
)

// const (
// 	FIFO  = iota //先进先出
// 	LFU   = iota //最近最不常用
// 	LRU   = iota //最近最少使用
// 	LRU_K = iota //LRU改进法
// 	TWOQ  = iota //LRU-K 的一个具体版本
// 	ARC   = iota //自适应缓存替换
// )

type LRUCache struct {
	lock       sync.Mutex               //互斥锁
	limit      int                      //数据个数限制
	size       int                      //当前存储的数据数量
	datas      map[string]interface{}   //数据存储
	timeLst    *list.List               //维护最近使用时间
	timeEleMap map[string]*list.Element //维护数据对应的timeLst节点
}

func NewCache(limit int) *LRUCache {
	return &LRUCache{
		limit:      limit,
		size:       0,
		datas:      make(map[string]interface{}),
		timeLst:    list.New(),
		timeEleMap: make(map[string]*list.Element),
	}
}

func (cache *LRUCache) Insert(key string, value interface{}) error {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	if _, ok := cache.datas[key]; ok { //键存在
		cache.makeTimeNew(key)
		return errors.New("key already exists")
	}

	if cache.size >= cache.limit { //存量超出限制，清除最后一个使用的
		ele := cache.timeLst.Front()
		delete(cache.datas, ele.Value.(string))
		delete(cache.timeEleMap, ele.Value.(string))
		cache.timeLst.Remove(ele)

		cache.size -= 1
	}

	cache.size += 1                                  //更新大小
	cache.datas[key] = value                         //设置值
	cache.timeEleMap[key] = cache.insertTimeNew(key) //添加最近使用时间

	return nil
}

func (cache *LRUCache) Replace(key string, value interface{}) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	if _, ok := cache.datas[key]; ok { //键存在
		cache.makeTimeNew(key)
		cache.datas[key] = value
		return
	}

	if cache.size >= cache.limit { //存量超出限制，清除最后一个使用的
		ele := cache.timeLst.Front()
		delete(cache.datas, ele.Value.(string))
		delete(cache.timeEleMap, ele.Value.(string))
		cache.timeLst.Remove(ele)

		cache.size -= 1
	}

	cache.size += 1                                  //更新大小
	cache.datas[key] = value                         //设置值
	cache.timeEleMap[key] = cache.insertTimeNew(key) //添加最近使用时间
}

func (cache *LRUCache) Get(key string) interface{} {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	if value, ok := cache.datas[key]; ok { //键存在
		cache.makeTimeNew(key)
		return value
	}
	return nil
}

func (cache *LRUCache) Remove(key string) error {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	if _, ok := cache.datas[key]; ok { //键存在
		ele := cache.timeEleMap[key]
		cache.timeLst.Remove(ele)

		delete(cache.datas, key)
		delete(cache.timeEleMap, key)

		cache.size -= 1
		return nil
	}
	return errors.New("key not found")
}

// 将使用时间设置为最近
func (cache *LRUCache) makeTimeNew(key string) {
	cache.timeLst.MoveToBack(cache.timeEleMap[key])
}

// 添加一个最新时间
func (cache *LRUCache) insertTimeNew(key string) *list.Element {
	return cache.timeLst.PushBack(key)
}
