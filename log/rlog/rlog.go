package rlog

import (
	"encoding/binary"
	"sync/atomic"
	"unsafe"

	"github.com/plexsec/utils/log/shm"
)

// Error 错误
type Error struct {
	Value int
	msg   string
}

func (err Error) Error() string {
	return err.msg
}

var (
	// ErrorRetryError 重试失败
	ErrorRetryError = &Error{
		Value: -1,
		msg:   "Retry Error",
	}
	// ErrorQueryFull 队列满
	ErrorQueryFull = &Error{
		Value: -2,
		msg:   "Query full",
	}
	// ErrorQueryEmpty 队列空
	ErrorQueryEmpty = &Error{
		Value: -3,
		msg:   "query empty",
	}
	// ErrorCasError cas错误
	ErrorCasError = &Error{
		Value: -4,
		msg:   "cas error",
	}
)

const maxQueryLength = 1000

// Message 要写入的信息
type Message struct {
	Module     string
	Msg        string
	RetryTimes int // 重试的次数，0不重试，小于0直接丢弃
}

const maxMessageLength = 2000

// LogQuery log队列，写程序一直往里写，agent一直从里面读
type LogQuery struct {
	writeIndex int32
	readIndex  int32
	message    [maxQueryLength][maxMessageLength]byte
}

var query = &LogQuery{
	writeIndex: 0,
	readIndex:  0,
}

// Init 初始化队列
func Init() error {
	var key uintptr = 2
	var size uintptr = 4 + 4 + maxMessageLength*maxQueryLength
	shmid, err := shm.Shmget(key, size, shm.IPC_CREATE|0600)
	if err != 0 {
		return err
	}

	addr, err := shm.Shmat(shmid)
	if err != 0 {
		return err
	}

	query = (*LogQuery)(unsafe.Pointer(uintptr(addr)))
	query.readIndex = 0
	query.writeIndex = 0

	return nil
}

// Empty 判断队列是否空
func Empty() bool {
	return query.empty()
}

// Full 判断队列是否满
func Full() bool {
	return query.full()
}

// Write 往队列里写信息
func Write(message *Message) error {
	return query.write(message)
}

// Read 从队列里读信息
func Read() (module string, message string, err error) {
	return query.read()
}

func (q *LogQuery) empty() bool {
	writeIndex := atomic.LoadInt32(&q.writeIndex)
	readIndex := atomic.LoadInt32(&q.readIndex)
	return writeIndex == readIndex
}

func (q *LogQuery) full() bool {
	writeIndex := atomic.LoadInt32(&q.writeIndex)
	readIndex := atomic.LoadInt32(&q.readIndex)
	nextIndex := (writeIndex + maxQueryLength + 1) % maxQueryLength
	return nextIndex == readIndex
}

func (q *LogQuery) write(message *Message) error {
	if message.RetryTimes < 0 {
		return ErrorRetryError
	}

	writeIndex := atomic.LoadInt32(&q.writeIndex)
	readIndex := atomic.LoadInt32(&q.readIndex)

	index := (writeIndex + maxQueryLength + 1) % maxQueryLength
	if index == readIndex {
		// 队列满了
		// fmt.Printf("query full, writeIndex: %d, readIndex: %d, index: %d\n", writeIndex, readIndex, index)
		return ErrorQueryFull
	}

	if !atomic.CompareAndSwapInt32(&q.writeIndex, writeIndex, index) {
		// 写失败，重试一次咯
		message.RetryTimes--
		// fmt.Printf("cas errro, writeIndex: %d, index: %d\n", writeIndex, index)
		return q.write(message)
	}

	moduleLen := uint16(len(message.Module))
	msgLen := uint16(len(message.Msg))

	if 2+2+moduleLen+msgLen > maxMessageLength {
		message.Msg = message.Msg[:maxMessageLength-moduleLen-4]
		msgLen = uint16(len(message.Msg))
	}

	// 写入格式为：
	// 2字节module长度+2字节msg长度+module+msg

	binary.LittleEndian.PutUint16(q.message[writeIndex][0:2], moduleLen)
	binary.LittleEndian.PutUint16(q.message[writeIndex][2:4], msgLen)
	copy(q.message[writeIndex][4:4+moduleLen], message.Module)
	copy(q.message[writeIndex][4+moduleLen:], message.Msg)

	return nil

}

func (q *LogQuery) read() (module string, message string, err error) {
	writeIndex := atomic.LoadInt32(&q.writeIndex)
	readIndex := atomic.LoadInt32(&q.readIndex)

	if readIndex == writeIndex {
		err = ErrorQueryEmpty
		// fmt.Printf("query empty, readIndex: %d, writeIndex: %d\n", readIndex, writeIndex)
		return
	}

	index := (readIndex + maxQueryLength + 1) % maxQueryLength
	if !atomic.CompareAndSwapInt32(&q.readIndex, readIndex, index) {
		// 读失败
		err = ErrorCasError
		// fmt.Printf("cas error, readIndex: %d, index: %d\n", readIndex, index)
		return
	}

	moduleLen := binary.LittleEndian.Uint16(q.message[readIndex][0:2])
	msgLen := binary.LittleEndian.Uint16(q.message[readIndex][2:4])
	module = string(q.message[readIndex][4 : 4+moduleLen])
	message = string(q.message[readIndex][4+moduleLen : 4+moduleLen+msgLen])

	return
}
