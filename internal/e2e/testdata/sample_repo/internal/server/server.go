package server

import "example.com/sample/internal/store"

func Health(v any) {
	_ = store.Ready
	_ = v
}
