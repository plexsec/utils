package log

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/plexsec/utils/log/rlog"
)

const (
	OFF      Level = 0
	VERBOSE  Level = 1
	DEBUG    Level = 2
	INFO     Level = 3
	WARN     Level = 4
	ERROR    Level = 5
	CRITICAL Level = 6
	FATAL    Level = 7
)

// 日志级别
type Level int

func newLevel(v interface{}) Level {
	switch l := v.(type) {
	case int:
		return Level(l)
	case string:
		s := strings.ToUpper(l)
		switch s {
		case "OFF":
			return OFF
		case "FATAL":
			return FATAL
		case "CRITICAL":
			return CRITICAL
		case "ERROR":
			return ERROR
		case "WARN":
			return WARN
		case "INFO":
			return INFO
		case "DEBUG":
			return DEBUG
		case "VERBOSE":
			return VERBOSE
		}
	}

	return INFO
}

func (l Level) name() string {
	switch l {
	case OFF:
		return "OFF"
	case FATAL:
		return "FATAL"
	case CRITICAL:
		return "CRITICAL"
	case ERROR:
		return "ERROR"
	case WARN:
		return "WARN"
	case INFO:
		return "INFO"
	case DEBUG:
		return "DEBUG"
	case VERBOSE:
		return "VERBOSE"
	default:
		return "NONE"
	}
}

func (l Level) log(cmp Level) bool {
	if l == OFF {
		return false
	}

	if cmp >= l {
		return true
	} else {
		return false
	}
}

type fileLogger struct {
	on         bool
	name       string //日志文件名
	level      Level
	maxSize    int64
	maxFileNum int
	count      int
}

type remoteLogger struct {
	on          bool
	level       Level
	retry       int
	initialized bool
	initLock    *sync.Mutex
	ready       bool
}

type stdLogger struct {
	on    bool
	level Level
}

type logger struct {
	module string

	std    stdLogger
	file   fileLogger
	remote remoteLogger
}

var dl = logger{
	module: filepath.Base(os.Args[0]),

	std: stdLogger{
		level: INFO,
	},
	file: fileLogger{
		name:       "",
		level:      INFO,
		maxSize:    128 * 1024 * 1024,
		maxFileNum: 10,
	},
	remote: remoteLogger{
		level:    INFO,
		retry:    1,
		initLock: &sync.Mutex{},
	},
}

// fork一个子进程来启动agent
// 同时要监听子进程是否退出，一退出的话，就重新fork
func forkExec(addr string) {
	pa := &syscall.ProcAttr{
		Env: append(os.Environ(), "LOG_AGENT_KAFKA="+addr),
	}
	agentPath := os.Getenv("LOG_AGENT_PATH")
	if agentPath == "" {
		agentPath = path.Join("/plexsec", "bin", "log-agent")
	}

	// 检查agentPath是否存在且可执行
	fileInfo, err := os.Stat(agentPath)
	if err != nil {
		fmt.Printf("agentPath: %s, Stat, err: %v\n", agentPath, err)
		// 这里不退出进程，因为可能是旧的业务还没支持远程日志而没有设agent
		return
	}
	if fileInfo.IsDir() {
		fmt.Printf("agentPath: %s IsDir, disable remote log\n", agentPath)
		return
	}
	perm := fileInfo.Mode() & os.ModePerm
	if perm&0500 == 0 {
		fmt.Printf("agent: %s is no executable, mode:%d, disable remote log\n", agentPath, perm)
		return
	}

	for {
		pid, err := syscall.ForkExec(agentPath, nil, pa)
		if err != nil {
			fmt.Printf("Log agent forkExec error, agent: %s, err: %v\n", agentPath, err)
			time.Sleep(10 * time.Second)
		}

		fmt.Printf("Log agent forkExec success, agent: %s, pid: %d\n", agentPath, pid)

		var wstatus syscall.WaitStatus
		rusage := &syscall.Rusage{}
		_, err = syscall.Wait4(pid, &wstatus, syscall.WUNTRACED, rusage)
		if err != nil {
			fmt.Printf("Wait4 error, err: %s, disable remote log\n", err)
			return
		}

		if wstatus.CoreDump() || wstatus.Exited() || wstatus.Signaled() {
			continue
		} else {
			fmt.Println("Remote log stopped.")
		}
	}
}

// InitRemoteLog 初始化远端log
func initRemoteLog(addr string) error {
	dl.remote.initLock.Lock()
	defer dl.remote.initLock.Unlock()
	if dl.remote.initialized {
		return nil
	}
	dl.remote.initialized = true

	go forkExec(addr)

	err := rlog.Init()
	if err != nil {
		fmt.Printf("Remote logger initial failed: %v", err)
		return err
	}
	dl.remote.ready = true

	return nil
}

type StdLogConfig struct {
	Level interface{}
}

type FileLogConfig struct {
	Path       string
	MaxSize    int
	MaxFileNum int
	Level      interface{}
}

type RemoteLogConfig struct {
	Addr  string
	Level interface{}
}

type Config struct {
	Module string

	Std    *StdLogConfig
	File   *FileLogConfig
	Remote *RemoteLogConfig
}

func Init(cfg *Config) {
	if cfg == nil {
		fmt.Println("Log config is empty, disable any logger.")
		return
	}

	SetModule(cfg.Module)

	if cfg.Std != nil {
		SetStdLog(cfg.Std.Level)
		fmt.Printf("Enable stdout log level %v.\n", cfg.Std.Level)
	}

	if cfg.File != nil {
		SetFileLog(cfg.File.Path, cfg.File.Level)
		SetMaxLogFileNum(cfg.File.MaxFileNum)
		SetMaxLogFileSize(cfg.File.MaxSize)
		fmt.Printf("Enable file log level %v at path %s.\n", cfg.File.Level, cfg.File.Path)
	}

	if cfg.Remote != nil {
		SetRemoteLog(cfg.Remote)
		fmt.Printf("Enable remove log level %v\n", cfg.Remote.Level)
	}
}

// SetStdLog 文件日志的名字
func SetStdLog(level interface{}) {
	dl.std.on = true
	dl.std.level = newLevel(level)
}

func DisableStdLog(on bool) {
	dl.std.on = true
}

// 设置模块名，默认为程序名
func SetModule(module string) {
	if module == "" {
		dl.module = filepath.Base(os.Args[0])
	} else {
		dl.module = module
	}
}

// SetLogFileName 文件日志的名字
func SetFileLog(name string, level interface{}) {
	if dl.file.name == "" {
		return
	}

	dl.file.on = true
	dl.file.name = name
	dl.file.level = newLevel(level)
}

func DisableRemoteLog() {
	dl.remote.on = false
}

// SetLogMaxSize 设置log文件的大小
func SetMaxLogFileSize(logSize int) {
	if logSize == 0 {
		logSize = 128 * 1024 * 1024
	}

	if logSize < 1024*1024 {
		logSize = 1024 * 1024
	}

	dl.file.maxSize = int64(logSize)
}

// SetLogMaxFileNum 设置log文件数
func SetMaxLogFileNum(maxFileNum int) {
	if maxFileNum == 0 {
		maxFileNum = 10
	}

	if maxFileNum > 100 {
		maxFileNum = 100
	}

	if maxFileNum < 1 {
		maxFileNum = 1
	}

	dl.file.maxFileNum = maxFileNum
}

// 开启远程日志
func SetRemoteLog(cfg *RemoteLogConfig) {
	dl.remote.on = true
	dl.remote.level = newLevel(cfg.Level)
	initRemoteLog(cfg.Addr)
}

func DisableFileLog() {
	dl.file.on = false
}

// SetRemoteRetryCount 写远程日志失败的情况下，再重试的次数，默认是1
func SetRemoteRetryCount(retries int) {
	dl.remote.retry = retries
}

// Fatal 大于等于FATAL时都打印
func Fatal(format string, v ...interface{}) {
	output(FATAL, format, v...)
}

// Critial 大于等于CRITICAL时都打印
func Critical(format string, v ...interface{}) {
	output(CRITICAL, format, v...)
}

// Error 大于等于ERROR时都打印
func Error(format string, v ...interface{}) {
	output(ERROR, format, v...)
}

// Warn 大于等于WARN时都打印
func Warn(format string, v ...interface{}) {
	output(WARN, format, v...)
}

// Info 大于等于INFO时都打印
func Info(format string, v ...interface{}) {
	output(INFO, format, v...)
}

// Debug 大于等于DEBUG时都打印
func Debug(format string, v ...interface{}) {
	output(DEBUG, format, v...)
}

// Verbose 大于等于VERBOSE时都打印
func Verbose(format string, v ...interface{}) {
	output(VERBOSE, format, v...)
}

func output(level Level, format string, v ...interface{}) {
	writeStd := dl.std.on && dl.std.level.log(level)
	writeFile := dl.file.on && dl.file.level.log(level)
	writeRemote := dl.remote.on && dl.remote.level.log(level)

	if !writeStd && !writeFile && !writeRemote {
		return
	}

	var funcName, file string
	var pc uintptr
	var ok bool
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	} else {
		funcName = runtime.FuncForPC(pc).Name()
		for j := len(funcName) - 1; j > 0; j-- {
			if funcName[j] == '.' {
				funcName = funcName[j+1:]
				break
			}
		}
	}

	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			file = file[i+1:]
			break
		}
	}

	timeStr := time.Now().String()[0:26]
	str := fmt.Sprintf(format, v...)
	str = fmt.Sprintf(
		"%s|%s|%s|%s:%d %s|%s\n",
		timeStr,
		level.name(),
		dl.module,
		file,
		line,
		funcName,
		str)

	if writeStd {
		outputToStd(str)
	}

	if writeFile {
		outputToFile(str)
	}

	if writeRemote {
		outputToRemote(str)
	}

	if level == FATAL {
		panic(str)
	}
}

//const colTitle = "__________00_01_02_03_04_05_06_07__08_09_0A_0B_0C_0D_0E_0F\n"

func outputToStd(str string) {
	fmt.Fprintf(os.Stdout, str)
}

func outputToFile(str string) {
	if dl.file.name == "" {
		return
	}

	f, err := os.OpenFile(dl.file.name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return
	}
	fmt.Fprintf(f, str)
	f.Close()

	dl.file.count++
	if dl.file.count > 1000 {
		dl.file.count = 0
		shiftFiles()
	}
}

func outputToRemote(str string) {
	if !dl.remote.ready {
		return
	}

	rlog.Write(&rlog.Message{
		Module:     dl.module,
		Msg:        str,
		RetryTimes: dl.remote.retry,
	})
}

func shiftFiles() error {
	fileInfo, err := os.Stat(dl.file.name)
	if err != nil {
		return err
	}

	if fileInfo.Size() < dl.file.maxSize {
		return nil
	}
	//shift file
	for i := dl.file.maxFileNum - 2; i >= 0; i-- {
		var nameOld string
		if i == 0 {
			nameOld = dl.file.name
		} else {
			nameOld = fmt.Sprintf("%s.%d", dl.file.name, i)
		}
		fileInfo, err := os.Stat(nameOld)
		if err != nil {
			continue
		}
		if fileInfo.IsDir() {
			continue
		}
		nameNew := fmt.Sprintf("%s.%d", dl.file.name, i+1)
		os.Rename(nameOld, nameNew)
	}
	return nil
}
