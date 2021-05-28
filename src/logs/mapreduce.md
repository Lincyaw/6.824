# One
```bigquery
# nn @ ubuntu in ~/Documents/school/6.824/src/main on git:master x [17:06:03] 
$ sh test-mr.sh

*** Starting wc test.
2021/05/28 17:06:13 rpc.Register: method "Done" has 1 input parameters; needs exactly three
--- wc test: PASS
*** Starting indexer test.
2021/05/28 17:06:23 rpc.Register: method "Done" has 1 input parameters; needs exactly three
panic: runtime error: index out of range [8] with length 8

goroutine 36 [running]:
_/home/nn/Documents/school/6.824/src/mr.waitJob(0xc0000b6360, 0x7ac2b3, 0x3, 0xc00022e2a0, 0x26)
	/home/nn/Documents/school/6.824/src/mr/master.go:72 +0x317
created by _/home/nn/Documents/school/6.824/src/mr.(*Master).SendTask
	/home/nn/Documents/school/6.824/src/mr/master.go:122 +0x288
2021/05/28 17:06:34 dialing:dial unix /var/tmp/824-mr-1000: connect: connection refused
2021/05/28 17:06:34 dialing:dial unix /var/tmp/824-mr-1000: connect: connection refused
--- indexer test: PASS
*** Starting map parallelism test.
2021/05/28 17:06:34 rpc.Register: method "Done" has 1 input parameters; needs exactly three
--- map parallelism test: PASS
*** Starting reduce parallelism test.
2021/05/28 17:06:49 rpc.Register: method "Done" has 1 input parameters; needs exactly three
panic: runtime error: index out of range [8] with length 8

goroutine 72 [running]:
_/home/nn/Documents/school/6.824/src/mr.waitJob(0xc000074120, 0x7ac2b3, 0x3, 0xc000220240, 0x26)
	/home/nn/Documents/school/6.824/src/mr/master.go:72 +0x317
created by _/home/nn/Documents/school/6.824/src/mr.(*Master).SendTask
	/home/nn/Documents/school/6.824/src/mr/master.go:122 +0x288
2021/05/28 17:07:04 dialing:dial unix /var/tmp/824-mr-1000: connect: connection refused
--- reduce parallelism test: PASS
2021/05/28 17:07:04 dialing:dial unix /var/tmp/824-mr-1000: connect: connection refused
*** Starting crash test.
2021/05/28 17:07:04 rpc.Register: method "Done" has 1 input parameters; needs exactly three
2021/05/28 17:07:35 dialing:dial unix /var/tmp/824-mr-1000: connect: connection refused
mr-crash-all mr-correct-crash.txt differ: byte 138, line 1
--- crash output is not the same as mr-correct-crash.txt
--- crash test: FAIL
*** FAILED SOME TESTS

```