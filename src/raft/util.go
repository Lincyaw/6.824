package raft

import "log"

// Debugging
const Debug = 0

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug > 0 {
		log.Printf(format, a...)
	}
	return
}

func Int32Min(a,b int32) int32 {
	if a>b {
		return b
	}
	return a
}
func Int32Max(a int32, b int32) int32 {
	if a<b {
		return b
	}
	return a
}

func Max(a, b interface{}) interface{} {
	if a.(int32) < b.(int32) {
	    return b
	}else{
		return a
	}
}

func IntMax(a,b int) int {
	if a<b {
		return b
	}
	return a
}

func If(condition bool, trueVal, falseVal interface{}) interface{} {
	if condition {
		return trueVal
	}
	return falseVal
}