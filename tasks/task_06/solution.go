package main

import "container/list"

type entry[K comparable, V any] struct {
	key   K
	value V
}
type LRUCache[K comparable, V any] struct {
	capacity int
	ll       list.List
	items    map[K]*list.Element
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{capacity: capacity, items: make(map[K]*list.Element, capacity)}
}

func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	valueRef, found := c.items[key]
	if found {
		/* если элемент был найден -> делаем его самым "свежим" */
		c.ll.MoveToBack(valueRef)
	} else {
		return value, false
	}
	return valueRef.Value.(*entry[K, V]).value, found
}

func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		/* если capacity <= 0, то ничего не делаем с кэшом */
		return
	}
	if currentValue, found := c.items[key]; found {
		/* если в кэше уже есть объект по переданному ключу, то обновляем его значение */
		currentValue.Value.(*entry[K, V]).value = value
	} else {
		/* если в кэше нет объекта с переданным ключом, то... */
		if c.capacity == len(c.items) {
			/* ...вытесняем самое старое значение из кэша (первый элемент связного списка) */
			var first = c.ll.Front()
			if first == nil {
				/* первый элемент гарантированно найдется, поэтому ветка недостижима */
				panic("LRU invariant violated: expected a non-empty list")
			}
			delete(c.items, first.Value.(*entry[K, V]).key)
			c.ll.Remove(first)
		}
		/* ...добавляем новое значение в кэш */
		var newCacheItem = &entry[K, V]{key, value}
		c.ll.PushBack(newCacheItem)
		c.items[key] = c.ll.Back()
	}
}

type LRU[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
}
