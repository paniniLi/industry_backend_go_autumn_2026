package main

type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]V
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	return &Cache[K, V]{capacity: capacity, items: make(map[K]V, capacity)}
}

func (c *Cache[K, V]) Get(k K) (v V, ok bool) {
	value, found := c.items[k]
	return value, found
}

func (c *Cache[K, V]) Set(k K, v V) bool {
	/* если кэш уже заполнен и по ключу k нет объекта -> возвращаем false - состояние кэша не изменилось */
	if _, found := c.items[k]; c.capacity <= len(c.items) && !found {
		return false
	}
	/* если по ключу k есть закэшированное значение - обновляем его */
	c.items[k] = v
	return true
}
