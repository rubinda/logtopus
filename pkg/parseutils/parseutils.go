package parseutils

// Pop removes a key-value pair from a map and returns the value.
func Pop(m map[string]any, key string) any {
	v, ok := m[key]
	if ok {
		delete(m, key)
	}
	return v
}
