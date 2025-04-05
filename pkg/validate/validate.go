package validate

import (
	"github.com/go-playground/validator/v10"
	"sync"
)

var (
	instance *validator.Validate
	once     sync.Once
)

// Get 获取validator实例
func Get() *validator.Validate {
	once.Do(func() {
		instance = validator.New()
	})
	return instance
}
