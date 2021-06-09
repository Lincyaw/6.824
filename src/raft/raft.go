package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"../labrpc"
	"github.com/sirupsen/logrus"
)

// import "bytes"
// import "../labgob"

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

const (
	FOLLOWER   int32 = 1
	CANDICATER int32 = 2
	LEADER     int32 = 3
)

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex // Lock to protect shared access to this peer's state
	voteLock  sync.Mutex
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	Term    int32 // 当前的任期
	VoteFor int   //candidateId that received vote in current term (or null if none)
	Logs    []Log

	CommitIndex int
	LastApplied int

	NextIndex  []int
	MatchIndex []int

	// 我自己添加的状态
	CurrentState     int32 // 当前的状态
	receiveHeartBeat bool  // 在定时器超时之前是否收到了心跳
}

func (rf *Raft) String() string {
	b, err := json.Marshal(*rf)
	if err != nil {
		return fmt.Sprintf("%+v", *rf)
	}
	var out bytes.Buffer
	err = json.Indent(&out, b, "", "    ")
	if err != nil {
		return fmt.Sprintf("%+v", *rf)
	}
	return out.String()
}

func (r *RequestVoteArgs) String() string {
	b, err := json.Marshal(*r)
	if err != nil {
		return fmt.Sprintf("%+v", *r)
	}
	var out bytes.Buffer
	err = json.Indent(&out, b, "", "    ")
	if err != nil {
		return fmt.Sprintf("%+v", *r)
	}
	return out.String()
}

func (r *RequestVoteReply) String() string {
	b, err := json.Marshal(*r)
	if err != nil {
		return fmt.Sprintf("%+v", *r)
	}
	var out bytes.Buffer
	err = json.Indent(&out, b, "", "    ")
	if err != nil {
		return fmt.Sprintf("%+v", *r)
	}
	return out.String()
}

type Log struct {
	Term    int
	Command string
}

var logger *logrus.Logger

func init() {
	logger = logrus.New()

	file, err := os.OpenFile("logs/"+time.Now().Format(time.UnixDate)+".log", os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		logger.Out = file
	} else {
		logger.Info("Failed to log to file, using default stderr")
	}
	logger.SetLevel(logrus.TraceLevel)
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	// Your code here (2A).
	return int(rf.Term), rf.CurrentState == LEADER
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).

	// 候选人的任期号
	Term int
	// 请求选票的候选人 id
	CandidateId int
	// 候选人最后日志条目的索引值
	LastLogIndex int
	// 候选人最后日志条目的任期号
	LastLogTerm int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	// 当前任期号，以便候选人更新自己的任期号
	Term int
	// 是否同意这次选票
	VoteGranted bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {

	rf.mu.Lock()
	if args.Term >= int(rf.Term) && args.LastLogIndex >= len(rf.Logs) {
		reply.VoteGranted = true
		reply.Term = args.Term
		logger.Trace(rf.me, " 收到了 ", args.CandidateId, " 的选举请求, 并且同意了")
	} else {
		reply.VoteGranted = false
		reply.Term = int(rf.Term)
		logger.Trace(rf.me, " 收到了 ", args.CandidateId, " 的选举请求, 但不同意")
	}
	rf.mu.Unlock()
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble gecztting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
// func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
// 	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
// 	return ok
// }

type AppendEntriesArgs struct {
	// leader's term
	Term         int
	LeaderId     int
	PrevLogIndex int
	Entries      []int
	LeaderCommit int
}
type AppendEntriesReply struct {
	Term    int
	Success bool
}

// 心跳通知、日志追加 RPC
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	logger.Trace(rf.me, " 收到 id ", args.LeaderId, "的心跳请求，其任期为 ", args.Term, " 自己的任期为 ", rf.Term)
	rf.mu.Lock()
	if args.Term < int(rf.Term) {
		// 自己的任期号比发来的心跳的大
		reply.Success = false
		reply.Term = int(rf.Term)
	} else {
		reply.Success = true
		rf.Term = int32(args.Term)
		rf.receiveHeartBeat = true
		rf.CurrentState = FOLLOWER
	}
	rf.mu.Unlock()
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
// 让 raft 把 command 添加到 log 中
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
// peers: 一组网络标识符
// me: 本端在 peers 中的下标
func Make(peers []*labrpc.ClientEnd, me int, persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.Term = 0
	rf.CurrentState = FOLLOWER
	rf.receiveHeartBeat = false
	rf.Logs = append(rf.Logs, Log{
		Term:    0,
		Command: "Init",
	})

	// 启动心跳计时器，超时则发起选举，自己变成候选人状态
	ctx := context.Background()
	// follower
	go rf.CheckHeartBeatsClock(ctx, rand.Intn(200)+200)
	// leader
	go rf.SendHearBeatsClock(ctx)
	// candicator
	go rf.startVote()

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	return rf
}

func (rf *Raft) startVote() {
	for {
		rf.mu.Lock()
		if rf.CurrentState != CANDICATER {
			rf.mu.Unlock()
			continue
		}
		rf.mu.Unlock()

		logger.Debug("id ", rf.me, " 开始选举 ", rf.CurrentState)
		// 自增任期号
		rf.mu.Lock()
		rf.Term++
		rf.mu.Unlock()
		// 自己给自己投票
		cnt := 1

		args := RequestVoteArgs{}
		args.CandidateId = rf.me
		args.Term = int(rf.Term)
		args.LastLogIndex = len(rf.Logs)
		args.LastLogTerm = rf.Logs[len(rf.Logs)-1].Term
		// todo: 添加剩余的参数
		replies := make([]RequestVoteReply, len(rf.peers))

		logger.Trace("raft 对象信息", rf)
		logger.Trace("发送的选举消息:", args)

		// 在这里发起一次选举，向所有的 server 发送选举请求
		for idx, server := range rf.peers {
			if idx != rf.me {
				logger.Trace(rf.me, " 遍历到 ", idx, " 了！！！")
				replies[idx] = RequestVoteReply{}
				logger.Trace(rf.me, " 向 ", idx, " 发送了选举请求")
				ok := server.Call("Raft.RequestVote", &args, &(replies[idx]))
				if !ok {
					logger.Warn(rf.me, "给 ", idx, " 发的选举没有得到回复")
					continue
				}
				logger.Trace("收到的选举回复, Term:", replies[idx].Term, ", Agree? ", replies[idx].VoteGranted)
			}
		}

		for idx, v := range replies {
			if idx != rf.me {
				if v.VoteGranted {
					cnt++
				}
				if v.Term > int(rf.Term) {
					rf.mu.Lock()
					rf.Term = int32(v.Term)
					// 这里可以直接变成 follower，因为就算所有的服务器都是 follower，他们也会因为心跳超时而重新发起一轮选举
					rf.CurrentState = FOLLOWER
					rf.mu.Unlock()
					logger.Info("id ", rf.me, "选举失败, 收到的任期号为 ", v.Term, ", 自己的任期号为", rf.Term)
					return
				}
			}
		}
		logger.Debug("id ", rf.me, "获得选票", cnt, "张，总共有", len(rf.peers), "人")
		rf.mu.Lock()
		if cnt > len(rf.peers)/2 {
			rf.CurrentState = LEADER
			rf.mu.Unlock()
		} else {
			// 没有当选成功，并且自己也没有变成 follower，则继续选举，并且设置一定的超时时间，防止多数服务器同时又开始一轮选举
			rf.CurrentState = CANDICATER
			rf.mu.Unlock()
			time.Sleep((time.Duration(200) * time.Millisecond))
		}
	}
}

// 因为不允许使用 time.ticker，指导书建议用一个 for 循环 + sleep 来实现计时器
// 检查心跳计时器
// 超时 timeout ms
func (rf *Raft) CheckHeartBeatsClock(ctx context.Context, timeout int) {
	for {
		select {
		default:
			// 初始化标志位
			rf.mu.Lock()
			rf.receiveHeartBeat = false
			rf.mu.Unlock()

			time.Sleep(time.Duration(timeout) * time.Millisecond)

			// 没收到心跳
			rf.mu.Lock()
			if !rf.receiveHeartBeat {
				if rf.CurrentState != LEADER {
					rf.CurrentState = CANDICATER
				}
			} else {
				rf.CurrentState = FOLLOWER
			}
			rf.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

// 发送心跳计时器  master -> follower
// 每 150ms 向所有的服务器发送心跳，如果返回的心跳 Term 比自己大，则把自己的状态转变为 Follower，然后退出向所有服务器发送心跳的循环
func (rf *Raft) SendHearBeatsClock(ctx context.Context) {
	for {
		if rf.CurrentState != LEADER {
			continue
		}
		logger.Debug("id ", rf.me, " 发送心跳")
		select {
		default:
			args := AppendEntriesArgs{}
			args.Term = int(rf.Term)
			args.LeaderId = rf.me
			replies := make([]AppendEntriesReply, len(rf.peers))

			noReplyNumber := 0
			for idx, server := range rf.peers {
				if idx != rf.me {
					replies[idx] = AppendEntriesReply{}
					ok := server.Call("Raft.AppendEntries", &args, &(replies[idx]))
					if !ok {
						noReplyNumber++
						logger.Warn(rf.me, "给 ", idx, " 发的心跳没有得到回复")
						continue
					}
					if replies[idx].Term > int(rf.Term) {
						rf.mu.Lock()
						rf.Term = int32(replies[idx].Term)
						rf.CurrentState = FOLLOWER
						rf.mu.Unlock()
						goto CON
					}
				}
			}
			// if noReplyNumber > len(rf.peers)/2 {
			// 	logger.Error(rf.me, "发送的心跳有", noReplyNumber, "人都没有回复")
			// 	rf.mu.Lock()
			// 	rf.CurrentState = FOLLOWER
			// 	rf.mu.Unlock()
			// 	goto CON
			// }
			time.Sleep(time.Duration(150) * time.Millisecond)
		case <-ctx.Done():
			return
		}
	CON:
	}
}
