package mr

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

type Master struct {
	// 要处理哪些文件
	files []string
	// 对应的这些文件工作的状态
	nWorker []TaskStatus
	// 要有几个reduce工作
	nReduce int
	// 对应的这几个reduce工作是否完成
	reduceWork []TaskStatus
	// 所有工作都完成了吗
	done bool

	taskMap map[string]int
}

func (m *Master) ReceiveStatus(args *WorkStatus, reply *Reply) error {
	if args.WorkType == "map" {
		if args.Done {
			m.nWorker[m.taskMap[args.WorkerId]] = Finish
			log.Println("Map", m.taskMap[args.WorkerId], "finished")
		} else {
			// 归零
			m.nWorker[m.taskMap[args.WorkerId]] = NotStarted
			log.Println("Map", m.taskMap[args.WorkerId], "failed")
		}
	} else {
		if args.Done {
			m.reduceWork[m.taskMap[args.WorkerId]] = Finish
			log.Println("Reduce", m.taskMap[args.WorkerId], "finished")
		} else {
			// 归零
			m.reduceWork[m.taskMap[args.WorkerId]] = NotStarted
			log.Println("Reduce", m.taskMap[args.WorkerId], "failed")
		}
	}
	// Send map work first
	for i := range m.nWorker {
		if m.nWorker[i] != Finish {
			return nil
		}
	}
	for i := range m.reduceWork {
		if m.reduceWork[i] != Finish {
			return nil
		}
	}
	m.done = true
	return nil
}

// Your code here -- RPC handlers for the worker to call.

//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//

// Worker request a task to master
func (m *Master) SendTask(args *Args, reply *Reply) error {
	log.Println("taskId: ", args.WorkerId)
	// 每次调用这个函数，只分配一个任务
	for i := range m.nWorker {
		if m.nWorker[i] == NotStarted {
			reply.Filename = m.files[i]
			file, err := os.Open(reply.Filename)
			if err != nil {
				log.Fatalf("cannot open %v", reply.Filename)
			}
			defer file.Close()
			content, err := ioutil.ReadAll(file)
			if err != nil {
				log.Fatalf("cannot read %v", reply.Filename)
			}
			// 将id和对应的任务映射起来
			m.taskMap[args.WorkerId] = i

			reply.Content = string(content) // send all the doc.
			reply.WorkType = "map"
			reply.Id = i
			reply.NReduce = m.nReduce
			reply.Valid = true

			m.nWorker[i] = InTheMid
			log.Println("Map", i, "send out")

			// todo：发射一个定时器，来检查 10 秒后任务是否完成，否则就任务这个 worker 挂掉了
			return nil
		}
	}
	// 如果到这一步，说明map任务都发送出去了
	for k := 0; k < m.nReduce; k++ {
		if m.reduceWork[k] == NotStarted {
			m.taskMap[args.WorkerId] = k
			reply.Valid = true
			reply.WorkType = "reduce"
			reply.Id = k
			reply.NMap = len(m.files)
			reply.NReduce = m.nReduce
			m.reduceWork[k] = InTheMid
			log.Println("Reduce", k, "send out")
			return nil
		}
	}
	log.Println("All Done!!!!!!!!")
	reply.Valid = false
	return nil
}

//
// start a thread that listens for RPCs from worker.go
//
func (m *Master) server() {
	rpc.Register(m)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := masterSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrmaster.go calls Done() periodically to find out
// if the entire job has finished.
//
func (m *Master) Done() bool {
	//ret := false

	// Your code here.

	return m.done
}

//
// create a Master.
// main/mrmaster.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeMaster(files []string, nReduce int) *Master {
	m := Master{}
	log.Println(files)
	// Your code here.
	m.files = files
	m.nReduce = nReduce
	m.nWorker = make([]TaskStatus, len(files))
	m.reduceWork = make([]TaskStatus, nReduce)
	m.taskMap = make(map[string]int)
	for i := range m.nWorker {
		m.nWorker[i] = NotStarted
	}
	for i := range m.reduceWork {
		m.reduceWork[i] = NotStarted
	}
	m.server()
	return &m
}
