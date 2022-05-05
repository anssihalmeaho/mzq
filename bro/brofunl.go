package bro

import (
	"github.com/anssihalmeaho/mzq/msg"
	"github.com/anssihalmeaho/mzq/queue"

	"github.com/anssihalmeaho/funl/funl"
	"github.com/anssihalmeaho/funl/std"
)

// InitBro initializes module
func InitBro(interpreter *funl.Interpreter) (err error) {
	stdModuleName := "mzqbro"
	topFrame := funl.NewTopFrameWithInterpreter(interpreter)
	stdFuncs := []std.StdFuncInfo{
		{
			Name:   "new-broker",
			Getter: GetNewBroker,
		},
		{
			Name:   "reg-queue",
			Getter: GetRegQueue,
		},
		{
			Name:   "unreg-queue",
			Getter: GetUnRegQueue,
		},
		{
			Name:   "send-msg",
			Getter: GetSendMsg,
		},
		{
			Name:   "close",
			Getter: GetClose,
		},
	}
	err = std.SetSTDFunctions(topFrame, stdModuleName, stdFuncs, interpreter)
	return
}

// GetSendMsg ...
func GetSendMsg(name string) std.StdFuncType {
	return func(frame *funl.Frame, arguments []funl.Value) (retVal funl.Value) {
		if l := len(arguments); l != 4 {
			funl.RunTimeError2(frame, "%s: wrong amount of arguments (%d)", name, l)
		}
		if arguments[0].Kind != funl.OpaqueValue {
			funl.RunTimeError2(frame, "%s: requires opaque value", name)
		}
		if arguments[1].Kind != funl.StringValue {
			funl.RunTimeError2(frame, "%s: requires string value", name)
		}
		if arguments[2].Kind != funl.StringValue {
			funl.RunTimeError2(frame, "%s: requires string value", name)
		}
		broker := arguments[0].Data.(*OpaqueBroker)
		nodeName := arguments[1].Data.(string)
		qname := arguments[2].Data.(string)

		// encode data
		args := []*funl.Item{
			&funl.Item{
				Type: funl.ValueItem,
				Data: broker.encoder,
			},
			&funl.Item{
				Type: funl.ValueItem,
				Data: arguments[3],
			},
		}

		dataStrVal := funl.HandleCallOP(frame, args)
		dataStr := dataStrVal.Data.(string)
		err := broker.bro.SendMsg(nodeName, qname, []byte(dataStr))

		var isOK bool
		var errorText string
		if err == nil {
			isOK = true
		} else {
			errorText = err.Error()
		}

		values := []funl.Value{
			{
				Kind: funl.BoolValue,
				Data: isOK,
			},
			{
				Kind: funl.StringValue,
				Data: errorText,
			},
		}
		retVal = funl.MakeListOfValues(frame, values)
		return
	}
}

// GetUnRegQueue ...
func GetUnRegQueue(name string) std.StdFuncType {
	return func(frame *funl.Frame, arguments []funl.Value) (retVal funl.Value) {
		if l := len(arguments); l != 2 {
			funl.RunTimeError2(frame, "%s: wrong amount of arguments (%d)", name, l)
		}
		if arguments[0].Kind != funl.OpaqueValue {
			funl.RunTimeError2(frame, "%s: requires opaque value", name)
		}
		if arguments[1].Kind != funl.StringValue {
			funl.RunTimeError2(frame, "%s: requires string value", name)
		}
		broker := arguments[0].Data.(*OpaqueBroker)
		qname := arguments[1].Data.(string)
		err := broker.bro.UnRegisterQueue(qname)

		var isOK bool
		var errorText string
		if err == nil {
			isOK = true
		} else {
			errorText = err.Error()
		}

		values := []funl.Value{
			{
				Kind: funl.BoolValue,
				Data: isOK,
			},
			{
				Kind: funl.StringValue,
				Data: errorText,
			},
		}
		retVal = funl.MakeListOfValues(frame, values)
		return
	}
}

// GetRegQueue ...
func GetRegQueue(name string) std.StdFuncType {
	return func(frame *funl.Frame, arguments []funl.Value) (retVal funl.Value) {
		if l := len(arguments); l != 3 {
			funl.RunTimeError2(frame, "%s: wrong amount of arguments (%d)", name, l)
		}
		if arguments[0].Kind != funl.OpaqueValue {
			funl.RunTimeError2(frame, "%s: requires opaque value", name)
		}
		if arguments[1].Kind != funl.StringValue {
			funl.RunTimeError2(frame, "%s: requires string value", name)
		}
		if arguments[2].Kind != funl.OpaqueValue {
			funl.RunTimeError2(frame, "%s: requires opaque value", name)
		}
		broker := arguments[0].Data.(*OpaqueBroker)
		qname := arguments[1].Data.(string)
		oq := arguments[2].Data.(*queue.OpaqueQueue)
		err := broker.bro.RegisterQueue(qname, oq.GetQinside())

		var isOK bool
		var errorText string
		if err == nil {
			isOK = true
		} else {
			errorText = err.Error()
		}

		values := []funl.Value{
			{
				Kind: funl.BoolValue,
				Data: isOK,
			},
			{
				Kind: funl.StringValue,
				Data: errorText,
			},
		}
		retVal = funl.MakeListOfValues(frame, values)
		return
	}
}

// GetClose ...
func GetClose(name string) std.StdFuncType {
	return func(frame *funl.Frame, arguments []funl.Value) (retVal funl.Value) {
		if l := len(arguments); l != 1 {
			funl.RunTimeError2(frame, "%s: wrong amount of arguments (%d), need one", name, l)
		}
		if arguments[0].Kind != funl.OpaqueValue {
			funl.RunTimeError2(frame, "%s: requires opaque value", name)
		}
		broker := arguments[0].Data.(*OpaqueBroker)
		broker.bro.Close()
		retVal = funl.Value{Kind: funl.BoolValue, Data: true}
		return
	}
}

// GetNewBroker ...
func GetNewBroker(name string) std.StdFuncType {
	return func(frame *funl.Frame, arguments []funl.Value) (retVal funl.Value) {
		getEncoder := func(frame *funl.Frame) funl.Value {
			decItem := &funl.Item{
				Type: funl.ValueItem,
				Data: funl.Value{
					Kind: funl.StringValue,
					Data: "call(proc() import stdser import stdbytes proc(x) _ _ b = call(stdser.encode x): call(stdbytes.string b) end end)",
				},
			}
			return funl.HandleEvalOP(frame, []*funl.Item{decItem})
		}

		getDecoder := func(frame *funl.Frame) func(bdata []byte) interface{} {
			decItem := &funl.Item{
				Type: funl.ValueItem,
				Data: funl.Value{
					Kind: funl.StringValue,
					Data: "call(proc() import stdser proc(__b) _ _ __v = call(stdser.decode __b): __v end end)",
				},
			}
			decoderVal := funl.HandleEvalOP(frame, []*funl.Item{decItem})

			return func(bdata []byte) interface{} {
				arguments := []*funl.Item{
					{
						Type: funl.ValueItem,
						Data: decoderVal,
					},
					{
						Type: funl.ValueItem,
						Data: funl.Value{
							Kind: funl.OpaqueValue,
							Data: std.NewOpaqueByteArray(bdata),
						},
					},
				}
				return funl.HandleCallOP(frame, arguments)
			}
		}

		if l := len(arguments); l != 1 {
			funl.RunTimeError2(frame, "%s: wrong amount of arguments (%d), need one", name, l)
		}
		if arguments[0].Kind != funl.MapValue {
			funl.RunTimeError2(frame, "%s: requires map value", name)
		}
		options := msg.OptionsToGoMap(frame, name, arguments[0])
		broker, err := CreateBrokerV2(options, getDecoder(frame))

		var isOK bool
		var errorText string
		var val funl.Value
		if err == nil {
			isOK = true
			val = funl.Value{
				Kind: funl.OpaqueValue,
				Data: &OpaqueBroker{
					bro:     broker,
					encoder: getEncoder(frame),
				},
			}
		} else {
			errorText = err.Error()
			val = funl.Value{
				Kind: funl.StringValue,
				Data: "",
			}
		}

		values := []funl.Value{
			{
				Kind: funl.BoolValue,
				Data: isOK,
			},
			{
				Kind: funl.StringValue,
				Data: errorText,
			},
			val,
		}
		retVal = funl.MakeListOfValues(frame, values)
		return
	}
}

// OpaqueBroker ...
type OpaqueBroker struct {
	bro     *Broker
	encoder funl.Value
}

// TypeName ...
func (bro *OpaqueBroker) TypeName() string {
	return "broker"
}

// Str ...
func (bro *OpaqueBroker) Str() string {
	return "broker"
}

// Equals ...
func (bro *OpaqueBroker) Equals(with funl.OpaqueAPI) bool {
	return false
}
