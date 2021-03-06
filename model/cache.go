package model

import (
	"database/sql"
	"encoding/json"

	"github.com/bradfitz/gomemcache/memcache"

	"github.com/aiyi/go-user/db"
	"github.com/aiyi/go-user/mc"
)

// 从缓存里获取 user 信息, 如果没有找到返回 ErrNotFound.
func getFromCache(userId int64, user *User) (err error) {
	item, err := mc.Client().Get(mc.UserCacheKey(userId))
	if err != nil {
		if err == memcache.ErrCacheMiss {
			err = ErrNotFound
		}
		return
	}
	return json.Unmarshal(item.Value, user)
}

func putToCache(user *User) (err error) {
	userBytes, err := json.Marshal(user)
	if err != nil {
		return
	}
	mcItem := memcache.Item{
		Key:   mc.UserCacheKey(user.Id),
		Value: userBytes,
	}
	return mc.Client().Set(&mcItem)
}

func removeFromCache(userId int64) (err error) {
	if err = mc.Client().Delete(mc.UserCacheKey(userId)); err != nil {
		if err == memcache.ErrCacheMiss {
			err = nil
		}
		return
	}
	return
}

func syncToCache(userId int64) (err error) {
	stmt, err := db.GetStmt("select * from user where id=?")
	if err != nil {
		return
	}

	var user User
	if err = stmt.Get(&user, userId); err != nil {
		if err == sql.ErrNoRows {
			err = ErrNotFound
		}
		return
	}

	return putToCache(&user)
}
