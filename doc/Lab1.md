
# LAB1

```sh
# 1. Enter the 'main' folder, then excute the following command, making the wc.so freshly built
go build -buildmode=plugin ../mrapps/wc.go
# 2. Delete the previous output
rm mr-out*
# 3. Start master
go run mrmaster.go pg-*.txt
# 4. Open another shell in the same directory, starting the worker.
go run mrworker.go wc.so
```
## step1
本次实验的任务是实现一个分布式的MapReduce。只需要修改`mr/master.go`,`mr/worker.go`,`mr/rpc.go`三个文件。

根据提示，首先需要修改`workder.go`中的`Worker*()`函数来发送一个RPC给master，让master给点活干；
master回应一个还没开始做的 map 任务, 不至于让worker闲的没事干。

接下来修改worker，让他能够读文件，然后调用写好的 map 函数来干活。

### 看代码结构
上面提到的几个文件的目录结构为

```
main/mrworker.go
main/mrmaster.go

mr/master.go
mr/rpc.go
mr/worker.go

mrapps/wc.go
```

其中，`mrapps`文件夹中实现的是在不同的业务场景下的不同的`map`以及`reduce`函数。
这些都将以插件(plugin)的形式在运行时使用。

`main`文件夹的几个文件主要是调用了`mr`文件夹中我们实现的几个函数。他也提供了基本的工具函数，比如如何加载一个插件。

在我的第二个commit里，能够运行一个示例的程序。可以看一下他的函数调用流程：

MakeMaster -> server, 从而开始监听特定的端口号。在master中，一直循环这个任务，直到master.Done();

Worker这边呢，则发送了一个示例的请求给master。因此我们需要修改worker发送的请求，然后在master里处理对应的请求。



下一步工作具体要看hint的第6点及其之后的东西。

mrsequential 中的实现方法是：map 产生的中间文件全都存在一个文件里，然后通过排序，将其变成有序。
有序之后就开始遍历，如果key相同，则把他加到一个数组里。reduce的任务就是返回数组的长度。(- -，看起来挺憨的。)

hint里说
> The map part of your worker can use the ihash(key) function (in worker.go) to pick the reduce task for a given key.

但我还是不怎么懂他要我做什么。key 来自于哪里，这个哈希值有什么用？
要怎么在多个文件里实现排序呢？

一种理解是，key就是单词，将不同的单词哈希之后，将其放入同一个中间文件，即
放入同一个桶，这样在reduce的时候就可以保证，在这个文件里有的单词一定是这个单词在所有原始文件里出现过的次数。

这样想的话，这个文件里会有很多个像这样的单词。而且会出现的情况是单词分布不均匀，造成分布倾斜的情况。
最理想的情况实际上是每个key都创建一个文件，那么一个reduce的工作只是统计这个文件里的单词出现的频数。git 

但是reduce是有限制的，比如这里默认reduce工作者只有10个，那就意味着只能创建10个文件。

还有一个问题是，怎么调配reduce工作。或者说，怎么提前分配reduce工作？

根据hint的说法，在map工作结束后，他会把中间结果写到本地文件中，命名为mr-X-Y的格式。X是map任务的id，Y
是reduce任务的id，这说明在分配map时，实际上就决定了reduce的任务是谁的。（并不是很确定这种说法）

---
实际上不需要考虑我上面说的`怎么提前分配reduce工作？`这个问题。

分配到reduce的worker只需要接受一个index即可。这个index表示他需要处理哪一个reduce工作。
拿nReduce=10作为一个例子的话，master会给10个worker分配reduce工作。给他们发的index分别是0,1,2,...,9

拿到reduce工作的worker需要去找所有为mr-x-index格式的文件，将其统计即可。其中x是变量。

maybe是因为测试脚本只产生了3个worker？应该让worker干完活之后再去领点任务做做