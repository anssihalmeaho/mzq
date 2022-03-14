package queue

import (
	"github.com/anssihalmeaho/funl/funl"
	"github.com/anssihalmeaho/funl/std"
)

// InitQueue initializes module
func InitQueue(interpreter *funl.Interpreter) (err error) {
	stdModuleName := "mzqque"
	topFrame := funl.NewTopFrameWithInterpreter(interpreter)
	stdFuncs := []std.StdFuncInfo{
		{
			Name:   "new-queue",
			Getter: GetNewQueue,
		},
		{
			Name:   "putq",
			Getter: GetPutQ,
		},
		{
			Name:   "getq",
			Getter: GetGetQ,
		},
	}
	err = std.SetSTDFunctions(topFrame, stdModuleName, stdFuncs, interpreter)
	return
}

// GetGetQ gets value from queue
func GetGetQ(name string) std.StdFuncType {
	return func(frame *funl.Frame, arguments []funl.Value) (retVal funl.Value) {
		if l := len(arguments); l != 1 {
			funl.RunTimeError2(frame, "%s: wrong amount of arguments (%d), need one", name, l)
		}
		if arguments[0].Kind != funl.OpaqueValue {
			funl.RunTimeError2(frame, "%s: requires opaque value", name)
		}

		que := arguments[0].Data.(*OpaqueQueue)
		retVal = (que.q.Get()).(funl.Value)
		return
	}
}

// GetPutQ puts value to queue
func GetPutQ(name string) std.StdFuncType {
	return func(frame *funl.Frame, arguments []funl.Value) (retVal funl.Value) {
		if l := len(arguments); l != 2 {
			funl.RunTimeError2(frame, "%s: wrong amount of arguments (%d), need two", name, l)
		}
		if arguments[0].Kind != funl.OpaqueValue {
			funl.RunTimeError2(frame, "%s: requires opaque value", name)
		}

		que := arguments[0].Data.(*OpaqueQueue)
		que.q.Put(arguments[1])
		retVal = funl.Value{Kind: funl.BoolValue, Data: true}
		return
	}
}

// GetNewQueue creates new queue
func GetNewQueue(name string) std.StdFuncType {
	return func(frame *funl.Frame, arguments []funl.Value) (retVal funl.Value) {
		if l := len(arguments); l != 1 {
			funl.RunTimeError2(frame, "%s: wrong amount of arguments (%d), need one", name, l)
		}
		if arguments[0].Kind != funl.IntValue {
			funl.RunTimeError2(frame, "%s: requires int value", name)
		}

		que := NewQueue(arguments[0].Data.(int))
		retVal = funl.Value{Kind: funl.OpaqueValue, Data: &OpaqueQueue{q: que}}
		return
	}
}

// OpaqueQueue is queue
type OpaqueQueue struct {
	q *Queue
}

// TypeName ...
func (oq *OpaqueQueue) TypeName() string {
	return "queue"
}

// Str ...
func (oq *OpaqueQueue) Str() string {
	return "queue"
}

// Equals ...
func (oq *OpaqueQueue) Equals(with funl.OpaqueAPI) bool {
	return false
}
