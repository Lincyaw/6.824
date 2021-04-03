package mr

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
)
import "net"
import "net/rpc"
import "net/http"

type Master struct {
	// Your definitions here.
	files   []string
	nWorker []bool
	nReduce int
}

// Your code here -- RPC handlers for the worker to call.

//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (m *Master) Example(args *Args, reply *Reply) error {
	fmt.Println(m.files)
	for i:=range m.nWorker{
		if m.nWorker[i] == false{
			reply.Filename = m.files[i]

			file, err := os.Open(reply.Filename)
			if err != nil {
				log.Fatalf("cannot open %v", reply.Filename)
			}
			content, err := ioutil.ReadAll(file)
			if err != nil {
				log.Fatalf("cannot read %v", reply.Filename)
			}
			reply.Content = string(content)
			reply.Job = "map"
			reply.Number = i
			reply.NReduce = m.nReduce
			m.nWorker[i] = true
			err = file.Close()
			if err != nil {
				log.Fatalf("close file error %v", reply.Filename)
			}
			break
		}
	}

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
	ret := false

	// Your code here.

	return ret
}

//
// create a Master.
// main/mrmaster.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeMaster(files []string, nReduce int) *Master {
	m := Master{}

	// Your code here.
	m.files = files
	m.nReduce = nReduce
	m.nWorker = make([]bool, nReduce)
	for i := range m.nWorker {
		m.nWorker[i] = false
	}
	m.server()
	return &m
}
