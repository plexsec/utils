package log

import (
	"os"
	"testing"
)

func test() {
	a := 1
	str := "this is test"
	type testStruct struct {
		b int
		c float32
		d string
	}
	st := &testStruct{100, 10.1, "this is struct test"}

	Init("len=%d title=%s struct=%v", a, str, st)
	Info("len=%d title=%s struct=%v", a, str, st)
}

func TestAll(t *testing.T) {
	os.Setenv("LOG_AGENT_PATH", "./agent/agent")
	SetLog("test.log", LOGLEVEL_DEBUG)
	test()
	SetLogFileName("")
	test()
}
