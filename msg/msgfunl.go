package msg

import (
	"github.com/anssihalmeaho/funl/funl"
	"github.com/anssihalmeaho/funl/std"
)

// InitMsg initializes module
func InitMsg(interpreter *funl.Interpreter) (err error) {
	stdModuleName := "mzqmsg"
	topFrame := funl.NewTopFrameWithInterpreter(interpreter)
	stdFuncs := []std.StdFuncInfo{
		{
			Name:   "create-server",
			Getter: getCreateServer,
		},
		{
			Name:   "open-connection",
			Getter: getOpenConnection,
		},
		{
			Name:   "receive",
			Getter: getReceive,
		},
		{
			Name:   "msend",
			Getter: getSend,
		},
		{
			Name:   "close",
			Getter: getClose,
		},
	}
	err = std.SetSTDFunctions(topFrame, stdModuleName, stdFuncs, interpreter)
	return
}

// OpaqueServer is server
type OpaqueServer struct {
	server *MessageServer
}

// TypeName ...
func (server *OpaqueServer) TypeName() string {
	return "server"
}

// Str ...
func (server *OpaqueServer) Str() string {
	return "server"
}

// Equals ...
func (server *OpaqueServer) Equals(with funl.OpaqueAPI) bool {
	return false
}

// OpaqueConn ...
type OpaqueConn struct {
	c *Connection
}

// TypeName ...
func (conn *OpaqueConn) TypeName() string {
	return "connection"
}

// Str ...
func (conn *OpaqueConn) Str() string {
	return "connection"
}

// Equals ...
func (conn *OpaqueConn) Equals(with funl.OpaqueAPI) bool {
	return false
}

func getClose(name string) std.StdFuncType {
	return func(frame *funl.Frame, arguments []funl.Value) (retVal funl.Value) {
		if l := len(arguments); l != 1 {
			funl.RunTimeError2(frame, "%s: wrong amount of arguments (%d), need one", name, l)
		}
		if arguments[0].Kind != funl.OpaqueValue {
			funl.RunTimeError2(frame, "%s: requires opaque value", name)
		}

		con := arguments[0].Data.(*OpaqueConn)
		con.c.Close()
		retVal = funl.Value{Kind: funl.BoolValue, Data: true}
		return
	}
}

func getSend(name string) std.StdFuncType {
	return func(frame *funl.Frame, arguments []funl.Value) (retVal funl.Value) {
		if l := len(arguments); l != 2 {
			funl.RunTimeError2(frame, "%s: wrong amount of arguments (%d), need two", name, l)
		}
		if arguments[0].Kind != funl.OpaqueValue {
			funl.RunTimeError2(frame, "%s: requires opaque value", name)
		}
		if arguments[1].Kind != funl.StringValue {
			funl.RunTimeError2(frame, "%s: requires string value", name)
		}

		con := arguments[0].Data.(*OpaqueConn)
		data := arguments[1].Data.(string)
		err := con.c.Send(data)

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

func getReceive(name string) std.StdFuncType {
	return func(frame *funl.Frame, arguments []funl.Value) (retVal funl.Value) {
		if l := len(arguments); l != 1 {
			funl.RunTimeError2(frame, "%s: wrong amount of arguments (%d), need one", name, l)
		}
		if arguments[0].Kind != funl.OpaqueValue {
			funl.RunTimeError2(frame, "%s: requires opaque value", name)
		}

		opaqueserver := arguments[0].Data.(*OpaqueServer)
		server := opaqueserver.server
		message, err := server.Receive()
		var isOK bool
		var errorText string

		if err == nil {
			isOK = true
		} else {
			errorText = err.Error()
		}

		var messageOperands []*funl.Item
		if isOK {
			messageOperands = []*funl.Item{
				&funl.Item{
					Type: funl.ValueItem,
					Data: funl.Value{
						Kind: funl.StringValue,
						Data: "from-addr",
					},
				},
				&funl.Item{
					Type: funl.ValueItem,
					Data: funl.Value{
						Kind: funl.StringValue,
						Data: message.FromAddr,
					},
				},
				&funl.Item{
					Type: funl.ValueItem,
					Data: funl.Value{
						Kind: funl.StringValue,
						Data: "data",
					},
				},
				&funl.Item{
					Type: funl.ValueItem,
					Data: funl.Value{
						Kind: funl.StringValue,
						Data: message.Data,
					},
				},
				/*
					&funl.Item{
						Type: funl.ValueItem,
						Data: funl.Value{
							Kind: funl.OpaqueValue,
							Data: std.NewOpaqueByteArray([]byte(message.Data)),
						},
					},
				*/
			}
		} else {
			messageOperands = []*funl.Item{}
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
			funl.HandleMapOP(frame, messageOperands),
		}
		retVal = funl.MakeListOfValues(frame, values)
		return
	}
}

func getOpenConnection(name string) std.StdFuncType {
	return func(frame *funl.Frame, arguments []funl.Value) (retVal funl.Value) {
		if l := len(arguments); l != 2 {
			funl.RunTimeError2(frame, "%s: wrong amount of arguments (%d), need two", name, l)
		}
		if arguments[0].Kind != funl.OpaqueValue {
			funl.RunTimeError2(frame, "%s: requires opaque value", name)
		}
		if arguments[1].Kind != funl.StringValue {
			funl.RunTimeError2(frame, "%s: requires string value", name)
		}

		opaqueserver := arguments[0].Data.(*OpaqueServer)
		server := opaqueserver.server
		conn, err := server.OpenConnection(arguments[1].Data.(string))
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
			funl.Value{Kind: funl.OpaqueValue, Data: &OpaqueConn{c: conn}},
		}
		retVal = funl.MakeListOfValues(frame, values)
		return
	}
}

func getCreateServer(name string) std.StdFuncType {
	return func(frame *funl.Frame, arguments []funl.Value) (retVal funl.Value) {
		if l := len(arguments); l != 1 {
			funl.RunTimeError2(frame, "%s: wrong amount of arguments (%d), need one", name, l)
		}
		if arguments[0].Kind != funl.MapValue {
			funl.RunTimeError2(frame, "%s: requires map value", name)
		}

		options := optionsToGoMap(frame, name, arguments[0])
		address, addrFound := options["addr"]
		if !addrFound {
			funl.RunTimeError2(frame, "%s: addr not given in options", name)
		}
		server, err := CreateServer(Options{Addr: address.(string)})
		if err != nil {
			funl.RunTimeError2(frame, "%s: error (%v)", name, err)
		}
		retVal = funl.Value{Kind: funl.OpaqueValue, Data: &OpaqueServer{server: server}}
		return
	}
}

func optionsToGoMap(frame *funl.Frame, name string, mapVal funl.Value) map[string]interface{} {
	keyvals := funl.HandleKeyvalsOP(frame, []*funl.Item{&funl.Item{Type: funl.ValueItem, Data: mapVal}})
	kvListIter := funl.NewListIterator(keyvals)
	resultMap := map[string]interface{}{}
	for {
		nextKV := kvListIter.Next()
		if nextKV == nil {
			break
		}
		kvIter := funl.NewListIterator(*nextKV)
		keyv := *(kvIter.Next())
		valv := *(kvIter.Next())
		if keyv.Kind != funl.StringValue {
			funl.RunTimeError2(frame, "%s: header key not a string: %v", name, keyv)
		}
		resultMap[keyv.Data.(string)] = valv.Data
	}
	return resultMap
}
