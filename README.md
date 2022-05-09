# mzq

**mzq** is messaging library for FunL (and Go) programs.
It enables processes to send and receive messages to each other.

There are several layers in **mzq**:

* **msg package (mzqmsg FunL module)**: basic message sending over sockets from one point to another
* **queue package (mzqque FunL module)**: (message-)queue implementation
* **bro package (mzqbro FunL module)**: connecting several messaging nodes and routing to queues

Here's picture about module/package dependencies and API's of **mzq**:

![](https://github.com/anssihalmeaho/mzq/blob/main/mzqpic.png)


## Concepts

**Message** is FunL value delivered to **queue** via broker (when mzqbro used) or
byte array (or string) delivered over remote connection (if Go API or mzqmsg used).

**Queue** is bounded buffer to which FunL values can be written to and read from.
**queue** supports concurrent read and write access.

**Node** is remote messaging target (handled by connection/broker) identified
by unique name (string).


## bro package / mzqbro module

With broker module client can connect to several other nodes (peer brokers).
Broker module also routes messages to target queues and encodes/decodes FunL values
in messages.

### new-broker
Creates new broker. Options map is given as argument.

Options map contains:

Name | Value
---- | -----
'own-name' | name of this broker (string)
'own-addr' | address of this broker (string)
'addrs' | list of peer broker addresses (list of strings)

Format:

```
call(mzqbro.new-broker <options-map>) -> list(ok:bool error:string broker:opaque-value)
```

### reg-queue
Registers queue to broker for receiving messages.

Format:

```
call(mzqbro.reg-queue <opaque:broker> <queue-name:string> <opaque:queue>) -> list(ok:bool error:string)
```

### unreg-queue
Unregisters queue from broker.

Format:

```
call(mzqbro.unreg-queue <opaque:broker> <queue-name:string>) -> list(ok:bool error:string)
```

### send-msg
Sends message (FunL value) to queue (name given) in given node (node name as string).

Format:

```
call(mzqbro.send-msg <opaque:broker> <node-name:string> <queue-name:string> <value>) -> list(ok:bool error:string)
```

### close
Closes broker.

Format:

```
call(mzqbro.close <opaque:broker>) -> true
```

## queue package / mzqque module

Queues are FIFO type of value buffers which have limited length.
Queues can be accessed concurrently from several goroutines/fibers.
In Go values in queue are of **any** (**interface{}**) -type and in
FunL those are any FunL values.
There are services which block caller or return immediately in case:

* caller is reading and queue is empty
* caller is writing and queue is full

Queues can be used locally as independent service or as part of
messaging with **bro/mzqbro**.

### new-queue
Creates new queue with given size.

Format:

```
call(mzqque.new-queue <queue-size: int>) -> <opaque:queue>
```

### putq
Writes value to queue. Blocks caller if queue is full.

Format:

```
call(mzqque.putq <opaque:queue> <value>) -> true
```

### getq
Reads value from queue. Blocks caller if queue is empty.

Format:

```
call(mzqque.getq <opaque:queue>) -> <value>
```
### putq-nw
Writes value to queue. Does not block caller if queue is full.

Format:

```
call(mzqque.putq-nw <opaque:queue> <value>) -> <was-value-added:bool>
```

Return value is:

* **true** if value was added to queue (queue was not full)
* **false** if value was not added to queue (queue was full)

### getq-nw
Reads value from queue. Does not block caller if queue is empty.

Format:

```
call(mzqque.getq-nw <opaque:queue>) -> list(<has-value:bool> <value>)
```

Return value is list of:

1. Boolean value which is **true** if has some value read from queue, **false** if not
2. Value from queue ('' if value was not read from queue)

## msg package / mzqmsg module

Basic messaging service provides services to create and use point-to-point
messaging connections hiding socket communication behind interface.
TCP protocol is used as implementation for messaging.

**Note.** this package/module is lower level when compared to **bro/mzqbro**
package/module and is not needed if broker used.

### create-server
Creates new messaging server for handling several messaging connections.
Options map need to be given as argument.

Options map contains:

Name | Value
---- | -----
'addr' | own address (specifying port, like ':8081')

Format:

```
call(mzqmsg.create-server <options:map>) -> <opaque:msg-server>
```

### open-connection
Opens new point-to-point connection. Target address is given as 2nd argument.

Format:

```
call(mzqmsg.open-connection <opaque:msg-server> <target-address:string>) -> list
```

Return list contains:

1. bool: **true** if succeeded, **false** if failed
2. error text (string)
3. Opaque connection value

### receive
Receives message arriving into server (from any connection).
Blocks caller until message is received.

Format:

```
call(mzqmsg.receive <opaque:msg-server>) -> list
```
Return list contains:

1. bool: **true** if message is received, **false** if not
2. error text (string)
3. Message value (map)

Message is represented as map:

Name | Value
---- | -----
'from-addr' | address from where message was received (string)
'data' | message data as string (can be changed to bytearray)

**Note.** 'from-addr' can be used for opening connection to that address.

### msend
Sends message to given connection. Message data is given
as string (can be changed from bytearray to string) in 2nd argument.

Format:

```
call(mzqmsg.msend <opaque:connection> <data:string>) -> list(<ok:bool> <error-text:string>)
```

### close
Closes connection.

Format:

```
call(mzqmsg.close <opaque:connection>) -> true
```

## Installation
There are several ways to take **mzq** into use.

### Use github.com/anssihalmeaho/funl as basis
If **funla** interpreter is built from **github.com/anssihalmeaho/funl**
then initialization code can be added to **extensions** -package in **funl**.

For example adding following file to **/extensions** directory (**mzq.go**):

```go
package extensions

import (
	"github.com/anssihalmeaho/funl/funl"
	"github.com/anssihalmeaho/mzq/bro"
	"github.com/anssihalmeaho/mzq/msg"
	"github.com/anssihalmeaho/mzq/queue"
)

func init() {
	funl.AddExtensionInitializer(msg.InitMsg)
	funl.AddExtensionInitializer(queue.InitQueue)
	funl.AddExtensionInitializer(bro.InitBro)
}
```
Then just building **funla** interpreter as normally:
[build interpreter](https://github.com/anssihalmeaho/funl)

### Using as part of embedded (in Go) FunL program
If FunL is used as embedded language (in Go program) then
**mzq** can included in Go program.

Here's example:

```go
package main

import (
	_ "embed"
	"fmt"

	"github.com/anssihalmeaho/funl/funl"
	"github.com/anssihalmeaho/funl/std"
	"github.com/anssihalmeaho/mzq/bro"
	"github.com/anssihalmeaho/mzq/msg"
	"github.com/anssihalmeaho/mzq/queue"
)

//go:embed msgexample.fnl
var program string

func init() {
	funl.AddExtensionInitializer(msg.InitMsg)
	funl.AddExtensionInitializer(queue.InitQueue)
	funl.AddExtensionInitializer(bro.InitBro)
}

func main() {
	funl.PrintingRTElocationAndScopeEnabled = true

	retv, err := funl.FunlMainWithArgs(program, []*funl.Item{}, "main", "msgexample.fnl", std.InitSTD)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	fmt.Println(fmt.Sprintf("Result is %v", retv))
}
```

Here's example FunL code in which one fiber sends messages to other fibers queue
via broker locally (**msgexample.fnl**):

```
ns main

import mzqbro
import mzqque

main = proc()
	options = map(
		'own-name' 'receiver'
		'own-addr' ':8082'
		'addrs' list('127.0.0.1:8081')
	)
	_ _ broker = call(mzqbro.new-broker options):
	my-queue = call(mzqque.new-queue 5)

	# spawn queue listener
	_ = spawn(call(proc()
		_ = print('received:' call(mzqque.getq my-queue))
		while(true 'none')
	end))

	# register queue
	_ = print('reg: ' call(mzqbro.reg-queue broker 'some-queue' my-queue))

	# spawn own sender
	_ = spawn(call(proc()
		import stdtime
		_ = call(stdtime.sleep 4)
		_ = print('send local: ' call(mzqbro.send-msg broker 'receiver' 'some-queue' map('Hello' 'World')))
		while(true 'none')
	end))

	# waiting loop
	call(proc()
		import stdtime
		_ = call(stdtime.sleep 2)
		_ = print('...')

		while(true 'none')
	end)
end

endns
```


### Usage as plugin module
See: [Plugin modules in FunL](https://github.com/anssihalmeaho/funl/wiki/plugin-modules)

### Running application in apprunner

If application is to be executed from [apprunner](https://github.com/anssihalmeaho/apprunner)
then all mzq -modules are ready to be imported as those are built-in to **apprunner**.


## Examples

See /examples directory for example codes.

## ToDo

Things to develope in future:

* TLS communication
* Peer (node) discovery with some Gossip protocol
* Peer connection supervision and re-establishment
