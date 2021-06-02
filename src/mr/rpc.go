package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.
type Args struct {
	WorkerId string
}

type ReplyWorker struct {
	// 是否有任务
	Valid bool
	// 文件名（实际上并没有用到这个参数
	Filename string
	// 文件的内容
	Content string
	// 任务，判断是map还是reduce
	WorkType string
	// 一共有几个map任务
	MapWork int
	// 当前是哪一个任务
	Id int
}

type WorkStatus struct {
	WorkerId string
	WorkType string
	Done     bool
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the master.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func masterSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
