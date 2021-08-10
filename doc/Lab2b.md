# Lab2b
需求：实现日志追加的部分。

需要在以下四个地方添加代码：
1. type Raft struct 
2. type RequestVoteArgs struct
3. func (rf *Raft) Start(command interface{}) (int, int, bool) 
4. Make(peers []*labrpc.ClientEnd, me int, persister *Persister, applyCh chan ApplyMsg)

![raft](images/raft.png


## start
首先思考 start 函数的作用到底是什么？我要如何在心跳的基础上传送日志？

按照论文里的描述，当 AppendEntries 中不带有日志的时候，就是心跳；如果带有日志，就是日志追加。那带有日志的时候，这个 RPC 还具有心跳的功能吗？

经过思考之后， 我认为带有日志的 RPC 也可以被认为是心跳。

而通过 Start 函数的注释，不难理解其作用是在 master 节点上增加一条日志。

1. 若该节点不是 master, 则返回 false。（此时当然不应该添加日志）
2. 若该节点是 master, 则将输入的 command 写入日志。并返回这条日志应具有的 term 和 index。

> 问题：是否需要在 Start 函数里向从服务器发送 appendEntries PRC 呢？

答案应该是 No。在 master 的发送心跳的函数里，已经使用了一个计时器，定时发送心跳。而现在应该补充日志追加这一块的代码。
即检查自己的日志是否与节点的日志匹配，
如果不匹配则发送不匹配部分的日志，如果匹配，则发送心跳。
(这里的匹不匹配指的是 master 端维护的 matchIndex[] 数组。

如果从节点接受了这条日志，则将对应的 matchIndex++。如果超过半数的从节点（包括自己）都接受了这条日志，则这条日志被标记为 commit，即 commitIndex++。

lastApplied 与 commitIndex 之间可能会存在不一致，因为对于 master 来说，一条日志被 apply 后，还需要等待半数以上的节点同意，才能被标记为 commit。

> 问题：对于从节点，commitIndex 在何时更新呢？在收到来自 master 的新日志之后，马上就认为这条日志被 commit 吗？