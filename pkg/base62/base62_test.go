package base62

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/logx"
)

// 实现简化版日志写入器
type testWriter struct {
	buf  bytes.Buffer
	lock sync.Mutex
}

func (w *testWriter) Alert(v interface{}) {
	w.writeString(fmt.Sprint(v))
}

func (w *testWriter) Close() error {
	return nil
}

func (w *testWriter) Debug(v interface{}, _ ...logx.LogField) {
	w.writeString(fmt.Sprint(v))
}

func (w *testWriter) Error(v interface{}, _ ...logx.LogField) {
	w.writeString(fmt.Sprint(v))
}

func (w *testWriter) Info(v interface{}, _ ...logx.LogField) {
	w.writeString(fmt.Sprint(v))
}

func (w *testWriter) Severe(v interface{}) {
	w.writeString(fmt.Sprint(v))
}

func (w *testWriter) Slow(v interface{}, _ ...logx.LogField) {
	w.writeString(fmt.Sprint(v))
}

func (w *testWriter) Stack(v interface{}) {
	w.writeString(fmt.Sprint(v))
}

func (w *testWriter) Stat(v interface{}, _ ...logx.LogField) {
	w.writeString(fmt.Sprint(v))
}

func (w *testWriter) writeString(s string) {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.buf.WriteString(s)
	w.buf.WriteByte('\n')
}

func TestConvert(t *testing.T) {
	// ... 保持原有测试用例不变 ...
}

func TestConvertWithCustomEnv(t *testing.T) {
	// ... 保持原有测试用例不变 ...
}

func TestInitBase62StrInvalidLength(t *testing.T) {
	oldEnv := os.Getenv(base62EnvKey)
	defer os.Setenv(base62EnvKey, oldEnv)
	os.Setenv(base62EnvKey, "invalid_length")

	tw := &testWriter{}
	originalWriter := logx.Reset()
	defer logx.SetWriter(originalWriter)
	logx.SetWriter(tw)

	once = sync.Once{}
	Convert(0)

	assert.Contains(t, tw.buf.String(), "BASE62STR must contain exactly 62 characters")
}

func TestInitBase62StrDuplicateChars(t *testing.T) {
	oldEnv := os.Getenv(base62EnvKey)
	defer os.Setenv(base62EnvKey, oldEnv)
	os.Setenv(base62EnvKey, "aabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	tw := &testWriter{}
	originalWriter := logx.Reset()
	defer logx.SetWriter(originalWriter)
	logx.SetWriter(tw)

	once = sync.Once{}
	Convert(0)

	assert.Contains(t, tw.buf.String(), "BASE62STR contains duplicate characters")
}

func TestHasDuplicateChars(t *testing.T) {
	// ... 保持原有测试用例不变 ...
}
