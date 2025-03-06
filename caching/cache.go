package caching

import (
	"fmt"
	"sync"
	"time"

	"github.com/keeley1/novelti-backend/models"
)

type cacheItem struct {
	Data      []models.Book
	Timestamp time.Time
}

var (
	cache    sync.Map
	cacheTTL = 10 * time.Minute
)

func GetFromCache(key string) ([]models.Book, bool) {
	if itemToCache, ok := cache.Load(key); ok {
		fmt.Print("Getting book from cache.....")
		item := itemToCache.(cacheItem)
		if time.Since(item.Timestamp) < cacheTTL {
			return item.Data, true
		}
		cache.Delete(key)
	}
	return nil, false
}

func SaveToCache(key string, books []models.Book) {
	fmt.Print("Saving book to cache.....")
	cache.Store(key, cacheItem{
		Data:      books,
		Timestamp: time.Now(),
	})
}
