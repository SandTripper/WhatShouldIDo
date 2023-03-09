package sessionmanager

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"time"
)

const MD5seed = "709394"

type SessionManager struct {
	cookieName    string          //cookie名称
	cookieExpire  int             //cookie有效期时间（单位：秒，0表示会话cookie）
	sessionExpire int64           //session有效期时间（单位：秒）
	gcDuration    int             //垃圾回收机制运行间隔时间（单位：秒）
	provider      sessionProvider //session存储器
	//lock          sync.Mutex      //互斥锁
}

// session的底层存储结构
type sessionProvider interface {
	create(sessionId string, data map[string]interface{}) error //创建session
	get(sessionId, key string) (interface{}, error)             //读取session键值
	getAll(sessionId string) (map[string]interface{}, error)    //读取session所有键值对
	set(sessionId, key string, value interface{}) error         //设置session键值
	destroy(sessionId string) error                             //销毁session
	gc(expire int64) error                                      //垃圾回收：删除过期session
}

func NewSessionManager(cookieName string, cookieExpire int, sessionExpire int64, gcDuration int, provider sessionProvider) *SessionManager {
	return &SessionManager{
		cookieName:    cookieName,
		cookieExpire:  cookieExpire,
		sessionExpire: sessionExpire,
		gcDuration:    gcDuration,
		provider:      provider,
	}
}

// 生成sessionID
func (*SessionManager) createSessionId() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

// 获取请求中的sessionId
func (sm *SessionManager) getSessionId(req *http.Request) (string, error) {
	c, err := req.Cookie(sm.cookieName)
	if err != nil {
		return "", errors.New("Reading cookie failed: " + err.Error())
	}
	if len(c.Value) == 0 { //尚未设置cookie
		return "", errors.New("Cookie does not exists: " + sm.cookieName)
	}
	return c.Value, nil
}

// 创建session
func (sm *SessionManager) Create(writer *http.ResponseWriter, req *http.Request, data map[string]interface{}) error {
	sessionId, _ := sm.getSessionId(req)
	if len(sessionId) > 0 { //请求中已有sessionId
		data, _ := sm.provider.getAll(sessionId)
		if data != nil { //已有session，无需创建
			return nil
		}
	}
	sessionId = sm.createSessionId() //创建新的ID
	if len(sessionId) == 0 {
		return errors.New("length of sessionId is 0")
	}
	err := sm.provider.create(sessionId, data)
	if err != nil {
		return err
	}
	if sm.cookieExpire == 0 { //会话cookie
		http.SetCookie(*writer, &http.Cookie{
			Name:     sm.cookieName,
			Value:    sessionId,
			Path:     "/", //一定要设置为根目录，才能在所有页面生效
			HttpOnly: true,
		})
	} else { //持久cookie
		// expire, _ := time.ParseDuration(strconv.Itoa(sm.cookieExpire) + "s")
		http.SetCookie(*writer, &http.Cookie{
			Name:     sm.cookieName,
			Value:    sessionId,
			Path:     "/", //一定要设置为根目录，才能在所有页面生效
			MaxAge:   sm.cookieExpire,
			HttpOnly: true,
		})
	}
	return nil
}

// 获取session键值err
func (sm *SessionManager) Get(req *http.Request, key string) (interface{}, error) {
	sessionId, err := sm.getSessionId(req)
	if err != nil {
		return "", errors.New("length of sessionId is 0")
	}
	return sm.provider.get(sessionId, key)
}

// 读取session所有键值对
func (sm *SessionManager) GetAll(req *http.Request) (map[string]interface{}, error) {
	sessionId, err := sm.getSessionId(req)
	if err != nil {
		return nil, errors.New("length of sessionId is 0")
	}
	return sm.provider.getAll(sessionId)
}

// 设置session键值
func (sm *SessionManager) Set(req *http.Request, key string, value string) error {
	sessionId, err := sm.getSessionId(req)
	if err != nil {
		return errors.New("length of sessionId is 0")
	}
	return sm.provider.set(sessionId, key, value)
}

// 销毁session
func (sm *SessionManager) Destroy(writer *http.ResponseWriter, req *http.Request) error {
	http.SetCookie(*writer, &http.Cookie{
		Name:    sm.cookieName,
		Expires: time.Now(),
	})
	sessionId, err := sm.getSessionId(req)
	if err != nil {
		return errors.New("length of sessionId is 0")
	}
	return sm.provider.destroy(sessionId)
}

// 垃圾回收：删除过期session
func (sm *SessionManager) Gc() error {
	err := sm.provider.gc(sm.sessionExpire)
	duration, _ := time.ParseDuration(strconv.Itoa(sm.gcDuration) + "s")
	time.AfterFunc(duration, func() { sm.Gc() }) //设置下次运行时间
	return err
}
