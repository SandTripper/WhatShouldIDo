package sessionmanager

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

type sessionTimePair struct {
	createTimeUnix int64
	sessionId      string
}

type MemoryProvider struct {
	sessionsLock        sync.RWMutex
	createTimeUnixsLock sync.RWMutex
	sessions            map[string]map[string]interface{}
	createTimeUnixs     *list.List
}

func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{
		sessions:        make(map[string]map[string]interface{}),
		createTimeUnixs: list.New(),
	}
}

func (mp *MemoryProvider) create(sessionId string, data map[string]interface{}) error { //创建session
	mp.sessionsLock.Lock()
	defer mp.sessionsLock.Unlock()
	if _, ok := mp.sessions[sessionId]; ok { //已存在sessionId
		return errors.New("sessionId is already exist")
	}
	mp.sessions[sessionId] = data //设置值
	mp.createTimeUnixsLock.Lock()
	defer mp.createTimeUnixsLock.Unlock()
	mp.createTimeUnixs.PushBack(sessionTimePair{
		createTimeUnix: time.Now().Unix(),
		sessionId:      sessionId,
	}) //设置创建时间
	return nil
}

func (mp *MemoryProvider) get(sessionId, key string) (interface{}, error) { //读取session键值
	mp.sessionsLock.RLock()
	defer mp.sessionsLock.RUnlock()
	if _, ok := mp.sessions[sessionId]; !ok { //不存在sessionId
		return "", errors.New("sessionId not found")
	}
	if v, ok := mp.sessions[sessionId][key]; ok {
		return v, nil
	} else { //不存在键
		return "", errors.New("key not found")
	}
}
func (mp *MemoryProvider) getAll(sessionId string) (map[string]interface{}, error) { //读取session所有键值对
	mp.sessionsLock.RLock()
	defer mp.sessionsLock.RUnlock()
	if data, ok := mp.sessions[sessionId]; !ok { //不存在sessionId
		return nil, errors.New("sessionId not found")
	} else {
		return data, nil
	}
}
func (mp *MemoryProvider) set(sessionId, key string, value interface{}) error { //设置session键值
	mp.sessionsLock.Lock()
	defer mp.sessionsLock.Unlock()
	if _, ok := mp.sessions[sessionId]; !ok { //不存在sessionId
		return errors.New("sessionId not found")
	}
	mp.sessions[sessionId][key] = value //设置值
	return nil
}
func (mp *MemoryProvider) destroy(sessionId string) error { //销毁session
	mp.sessionsLock.Lock()
	defer mp.sessionsLock.Unlock()
	if _, ok := mp.sessions[sessionId]; !ok { //不存在sessionId
		return errors.New("sessionId not found")
	} else {
		delete(mp.sessions, sessionId)
		return nil
	}
}
func (mp *MemoryProvider) gc(expire int64) error { //垃圾回收：删除过期session
	currentTime := time.Now().Unix() //获取当前时间

	mp.createTimeUnixsLock.Lock() //对createTimeUnixs加写锁
	defer mp.createTimeUnixsLock.Unlock()
	for mp.createTimeUnixs.Len() > 0 {
		if mp.createTimeUnixs.Front().Value.(sessionTimePair).createTimeUnix < currentTime-expire { //链表头部的session过期

			//删除对应的session数据
			mp.sessionsLock.Lock()
			defer mp.sessionsLock.Unlock()
			delete(mp.sessions, mp.createTimeUnixs.Front().Value.(sessionTimePair).sessionId)

			//在链表中删除
			mp.createTimeUnixs.Remove(mp.createTimeUnixs.Front())
		}
	}
	return nil
}
