# steward
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=RaaLabs_steward&metric=alert_status)](https://sonarcloud.io/dashboard?id=RaaLabs_steward)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=RaaLabs_steward&metric=sqale_rating)](https://sonarcloud.io/dashboard?id=RaaLabs_steward)

How do you control or know the state of something that are either allways online or occasionally available online ? like Servers or IOT platforms where the link for reaching them can be a not-always-connected satellite, GSM, or even reliable fiber links that might fail ?

The idea behind Steward is to help out with exactly that.

## What is it ?

Command And Control anything like Servers, Containers, VM's or others by creating and sending messages with methods who will describe what to do. Steward will then take the responsibility for making sure that the message are delivered to the receiver, and that the method specified are executed with the given parameters defined. An example of a message.

```json
[
    {
        "directory":"cli_command__result",
        "fileExtension": ".result",
        "toNode": "ship1",
        "data": ["bash","-c","sleep 5 & tree ./"],
        "method":"REQCliCommand",
        "replyMethod":"REQToFileAppend",
        "ACKTimeout":5,
        "retries":3,
        "replyACKTimeout":5,
        "replyRetries":3,
        "methodTimeout": 10
    }
]
```

If the receiver `toNode` is down when the message was sent, it will be retried until delivered within the criterias set for `timeouts` and `retries`.

## Overview

Send Shell Commands, HTTP Get, or Tail log files to control your servers by passing a message that will have guaranteed delivery if/when the subsribing node is available. Or for example send logs or metrics from an end node back to a central log subscriber. The result of the method executed will be delivered back to you from the node you sent it from.

The idea is to build and use a pure message passing architecture for the commands back and forth from nodes, where delivery is guaranteed, and where all of the processes in the system are running concurrently so if something breaks or some process is slow it will not affect the handling and delivery of the other messages in the system.

By default the system guarantees that the order of the messages are handled by the subscriber in the order they where sent. There have also been implemented a special type `NOSEQ` which will allow messages to be handled within that process in a not sequential manner. This is handy for jobs that will run for a long time, and where other messages are not dependent on it's result.

A node can be a server running any host operating system, a container living in the cloud somewhere, a rapsberry pi, or something else that needs to be controlled that have an operating system installed  . The message passing backend used is <https://nats.io>

## Inspiration

The idea for how to handle processes, messages and errors are based on Joe Armstrongs idea behind Erlang described in his Thesis <https://erlang.org/download/armstrong_thesis_2003.pdf>.

Joe's documents describes how to build a system where everything is based and sending messages back and forth between processes in Erlang, and where everything is done concurrently. I used those ideas as inspiration for building a fully decoupled system to control servers or container based systems by passing  messages between processes asynchronously to execute methods, handle errors if they occur, and handle the retrying if something fails.

Steward is written in programming language Go with NATS as the message broker.

## Why

With existing solutions there is often either a push or a pull kind of setup.

In a push setup the commands to be executed is pushed to the receiver, but if a command fails because for example a broken network link it is up to you as an administrator to detect those failures and retry them at a later time until it is executed successfully.

In a pull setup an agent is installed at the Edge unit, and the configuration or commands to execute locally are pulled from a central repository. With this kind of setup you can be pretty certain that sometime in the future the node will reach it's desired state, but you don't know when. And if you want to know the current state you will need to have some second service which gives you that information.

In it's simplest form the idea about using an event driven system as the core for management of Edge units is that the sender/publisher are fully decoupled from the receiver/subscriber. We can get an acknowledge if a message is received or not, and with this functionality we will at all times know the current state of the receiving end. We can also add information in the ACK message if the command sent to the receiver was successful or not by appending the actual output of the command.

## Publishing and Subscribing processes

All parts of the system like processes, method handlers, messages, error handling are running concurrently.

If one process hangs on a long running message method it will not affect the rest of the system.

### Publisher

- A message in valid format is appended to the in pipe.
- The message is picked up by the system and put on a FIFO ringbuffer.
- The method type of the message is checked, a subject is created based on the content of the message,  and a process to handle that message type for that specific receiving node is started if it does not exist.
- The message is then serialized to binary format, and sent to the subscriber on the receiving node.
- If the message is expected to be ACK'ed by the subcriber then the publisher will wait for an ACK if the message was delivered. If an ACK was not received within the defined timeout the message will be resent. The amount of retries are defined within the message.

### Subscriber

- The receiving end will need to have a subscriber process started on a specific subject and be allowed handle messages from the sending nodes to execute the method defined in the message.
- When a message have been deserialized, it will lookup the correct handler for the method type specified within the message, and execute that handler.
- If the output of the method called is supposed to be returned to the publiser it will do so by using the replyMethod specified, and pick up the next message in the queue.

### Logical structure

![overview](steward.svg)

## Terminology

- Node: Something with an operating system that have network available. This can be a server, a cloud instance, a container, or other.
- Process: A message handler that knows how to handle messages of a given subject concurrently.
- Message:
  - Command: Something to be executed on the message received. An example can be a shell command.
  - Event: Something that have happened. An example can be transfer of syslog data from a host.

## Features

### Messages in order

- By default the system guarantees that the order of the messages are handled by the subscriber in the order they where sent. So if a network link is down when the message is being sent, it will automatically be rescheduled at the specified interval with the given number of retries.

These types of messages have method starting with `REQ<Method name>`

### Messages not in order

- There have been implemented a special method type `REQn<Method name>` which will allow messages to be handled within that process in a not sequential manner. This is handy for jobs that will run for a long time, and where other messages are not dependent on it's result.

### Error messages from nodes

- Error messages will be sent back to the central error handler upon failure on a node.

### Message handling and threads

- The handling of all messages is done by spawning up a process for the handling the message in it's own thread. This allows us to individually down to the message level keep the state for each message both in regards to ACK's, error handling, send retries, and rerun of a method for a message if the first run was not successful.

- Processes for handling messages on a host can be restarted upon failure, or asked to just terminate and send a message back to the operator that something have gone seriously wrong. This is right now just partially implemented to test that the concept works.

- Processes on the publishing node for handling incomming messages for new nodes will automatically be spawned when needed if it does not already exist.

- Publishing processes will potentially be able to send to all nodes. It is the subscribing nodes who will limit from where and what they will receive from.

- Messages not fully processed or not started yet will be automatically handled in chronological order if the service is restarted since the current state of all the messages being processed are stored on the local node in a key value store until they are finished.

- All messages processed by a publisher will be written to a log file as they are processed, with all the information needed to recreate the same message if needed, or it can be used for auditing.

- All handling down to the process and message level are handled concurrently. So if there are problems handling one message sent to a node on a subject it will not affect the messages being sent to other nodes, or other messages sent on other subjects to the same host.

- Message types of both ACK and NACK, so we can decide if we want or don't want an Acknowledge if a message was delivered succesfully.
Example: We probably want an ACK when sending some CLICommand to be executed, but we don't care for an acknowledge (NACK) when we send an "hello I'm here" event.

### Timeouts and retries

- Default timeouts to wait for ACK messages and max attempts to retry sending a message are specified upon startup. This can be overridden on the message level.

- Timeout's can be specified on both the message, and the method. With other words a message can have a timeout, and for example if the method it will trigger is a shell command it can have it's own timeout so processes can have a timeout if they get stuck.

- Setting the retries to `0` is the same as unlimited retries.

### Flags and configuration file

Steward supports both the use of flags/arguments set at startup, and the use of a config file. But how it is used might be a little different than how similar use is normally done.

A default config file will be created at first startup if one does not exist, with standard defaults values set. Any value also provided via a flag will also be written to the config file. If Steward is restarted the current content of the config file will be used as the new defaults. Said with other words, if you restart Steward without any flags specified the values of the last run will be read from the config file and used.

If new values are provided via flags they will take precedence over the ones currently in the config file, and they will also replace the current value in the config file, making it the default for the next restart.

The only exception from the above are the `startSubX` flags which got one extra value that can be used which is the value `RST` for Reset. This will disable the specified subscriber, and also null out the array for which Nodes the subscriber will allow traffic from.

The config file can also be edited directly, making the use of flags not needed.

If just getting back to standard default for all config options needed, then delete the current config file, restart Steward, and a new config file with all the options set to it's default values will be created.

TIP: Most likely the best way to control how the service should behave and what is started is to start Steward the first time so it creates the default config file. Then stop the service, edit the config file and change the defaults needed. Then start the service again.

### Request Methods

#### REQCliCommand / REQnCliCommand

Run CLI command on a node. Linux/Windows/Mac/Docker-container or other.

Will run the command given, and return the stdout output of the command when the command is done.

#### REQnCliCommandCont

Run CLI command on a node. Linux/Windows/Mac/Docker-container or other.

Will run the command given, and return the stdout output of the command continously while the command runs.

#### REQTailFile

Tail log files on some node, and get the result sent back in a reply message.

#### REQHttpGet

Scrape web servers, and get the html sent back in a reply message.

#### REQHello

Get Hello messages from all running nodes.

#### REQErrorLog

Central error logger.

### Request Methods used for reply messages

#### REQToConsole

Print the output of the reply message to the console.

#### REQToFileAppend

Append the output of the reply message to a log file specified with the `directory` and `fileExtension` fields.

#### REQToFile

Write the output of the reply message to a log file specified with the `directory` and `fileExtension` fields.

### Errors reporting

- Report errors happening on some node in to central error handler.

### Prometheus metrics

- Prometheus exporters for Metrics

### Other

- More will come. In active development.

## Howto

### Build and Run

clone the repository, then cd `./steward/cmd` and do `go build -o steward`, and run the application with `./steward --help`

### Options for running

```text
    -brokerAddress string
    the address of the message broker (default "127.0.0.1:4222")
  -centralNodeName string
    The name of the central node to receive messages published by this node (default "central")
  -configFolder string
    folder who contains the config file. Defaults to ./etc/. If other folder is used this flag must be specified at startup. (default "./etc")
  -defaultMessageRetries int
    default amount of retries that will be done before a message is thrown away, and out of the system (default 3)
  -defaultMessageTimeout int
    default message timeout in seconds. This can be overridden on the message level (default 5)
  -nodeName string
    some unique string to identify this Edge unit (default "central")
  -profilingPort string
    The number of the profiling port
  -promHostAndPort string
    host and port for prometheus listener, e.g. localhost:2112 (default ":2112")
  -startPubREQHello int
    Make the current node send hello messages to central at given interval in seconds
  -startSubREQCliCommand value
    Specify comma separated list for nodes to allow messages from. Use "*" for from all. Value RST will turn off subscriber.
  -startSubREQErrorLog value
    Specify comma separated list for nodes to allow messages from. Use "*" for from all. Value RST will turn off subscriber.
  -startSubREQHello value
    Specify comma separated list for nodes to allow messages from. Use "*" for from all. Value RST will turn off subscriber.
  -startSubREQHttpGet value
    Specify comma separated list for nodes to allow messages from. Use "*" for from all. Value RST will turn off subscriber.
  -startSubREQPing value
    Specify comma separated list for nodes to allow messages from. Use "*" for from all. Value RST will turn off subscriber.
  -startSubREQPong value
    Specify comma separated list for nodes to allow messages from. Use "*" for from all. Value RST will turn off subscriber.
  -startSubREQTailFile value
    Specify comma separated list for nodes to allow messages from. Use "*" for from all. Value RST will turn off subscriber.
  -startSubREQToConsole value
    Specify comma separated list for nodes to allow messages from. Use "*" for from all. Value RST will turn off subscriber.
  -startSubREQToFile value
    Specify comma separated list for nodes to allow messages from. Use "*" for from all. Value RST will turn off subscriber.
  -startSubREQToFileAppend value
    Specify comma separated list for nodes to allow messages from. Use "*" for from all. Value RST will turn off subscriber.
  -startSubREQnCliCommand value
    Specify comma separated list for nodes to allow messages from. Use "*" for from all. Value RST will turn off subscriber.
  -subscribersDataFolder string
    The data folder where subscribers are allowed to write their data if needed (default "./var")
```

### How to Run

#### nats-server (the message broker)

The broker for messaging is Nats-server from <https://nats.io>. Download, run it, and use the `-brokerAddress` flag on Steward to point to it.

There is a lot of different variants of how you can setup and confiure Nats. Full mesh, leaf node, TLS, Authentication, and more. You can read more about how to configure the Nats broker called nats-server at <https://nats.io/>.

Some example configuration for the nats-server are located in the `doc` folder in this repository.

#### Steward

On some central server which will act as your command and control server.

`./steward --node="central"`

One the nodes out there

`./steward --node="ship1"` & `./steward --node="ship1"` and so on.

Use the `-help` flag to get all possibilities.

A complete example to start a central node called `central`.

`./steward --nodeName="central" --defaultMessageRetries=3 --defaultMessageTimeout=5 -subscribersDataFolder="./data" --centralNodeName="central" -startSubREQErrorLog="*" -subscribersDataFolder="./var" -brokerAddress="127.0.0.1:4222"`

And start another node that will be managed via central.

`./steward --nodeName="ship1" -startPubREQHello=200 --centralNodeName="central" -promHostAndPort=":12112" -brokerAddress="127.0.0.1:4222"`

#### Start subscriber flags

The start subscribers flags take a string value of which nodes that it will process messages from. Since using a flag to set a value automatically sets that value also in the config file, a value of RST can be given to turn off the subscriber.

### Message fields explanation

```go
// The node to send the message to
toNode
// The actual data in the message
data
// Method, what is this message doing, etc. CLI, syslog, etc.
method
// ReplyMethod, is the method to use for the reply message.
// By default the reply method will be set to log to file, but
// you can override it setting your own here.
replyMethod
// Initial message Reply ACK wait timeout
ACKTimeout
// Normal Resend retries
retries
// The ACK timeout of the new message created via a request event.
replyACKTimeout
// The retries of the new message created via a request event.
replyRetries
// Timeout for long a process should be allowed to operate
methodTimeout
// Directory is a string that can be used to create the
//directory structure when saving the result of some method.
// For example "syslog","metrics", or "metrics/mysensor"
// The type is typically used in the handler of a method.
directory
// FileExtension is used to be able to set a wanted extension
// on a file being saved as the result of data being handled
// by a method handler.
fileExtension
// operation are used to give an opCmd and opArg's.
operation
```

### How to send a Message

Right now the API for sending a message from one node to another node is by pasting a structured JSON object into the socket file file called `steward.sock` living alongside the binary. This file will be read continously, and when updated the content will be picked up, umarshaled, and if OK it will be sent a message to the node specified in the `toNode` field.

The `method` is what defines what the event will do.

The `Operation` field is a little bit special. This field is used with the `REQOpCommand` to specify what operation command to run, and also it's arguments.

#### The current `operation`'s that are available are

To stop a process of a specific type on a node.

```json
...
"method":"REQOpCommand",
        "operation":{
            "opCmd":"stopProc",
            "opArg": {
                "method": "REQHttpGet",
                "kind": "subscriber",
                "receivingNode": "ship2",
                "id":127
            }
        },
...
```

To get a list of all running processes on a node.

```json
...
"method":"REQOpCommand",
        "operation":{
            "opCmd":"ps"
        },
...
```

To start a process of a specified type on a node.

```json
"method":"REQOpCommand",
        "operation":{
            "opCmd":"startProc",
            "opArg": {
                "method": "REQHttpGet",
                "allowedNodes": ["central","node1"]
            }
        },
```

and another example

```json
[
    {
        "directory":"opcommand_logs",
        "fileExtension": ".log",
        "toNode": "ship2",
        "data": [],
        "method":"REQOpCommand",
        "operation":{
            "opCmd":"startProc",
            "opArg": {
                "method": "REQnCliCommandCont",
                "allowedNodes": ["central"]
            }
        },
        "replyMethod":"REQToFileAppend",
        "ACKTimeout":3,
        "retries":3,
        "replyACKTimeout":3,
        "replyRetries":3,
        "MethodTimeout": 7
    }
]
```

NB: Both the keys and the values used are case sensitive.

#### Sending a command from one Node to Another Node

Example JSON for appending a message of type command into the `socket` file

```json
[
    {
        "directory":"/var/steward/cli-command/executed-result",
        "toNode": "ship1",
        "data": ["bash","-c","sleep 3 & tree ./"],
        "method":"REQCliCommand",
        "ACKTimeout":10,
        "retries":3,
        "methodTimeout": 4
    }
]
```

To specify more messages at once do

```json
[
    {
        "directory":"cli-command-executed-result",
        "toNode": "ship1",
        "data": ["bash","-c","sleep 3 & tree ./"],
        "method":"REQCliCommand",
        "ACKTimeout":10,
        "retries":3,
        "methodTimeout": 4
    },
    {
        "directory":"cli-command-executed-result",
        "toNode": "ship2",
        "data": ["bash","-c","sleep 3 & tree ./"],
        "method":"REQCliCommand",
        "ACKTimeout":10,
        "retries":3,
        "methodTimeout": 4
    }
]
```

To send a Op Command message for process listing with custom timeout and amount of retries

```json
[
    {
        "directory":"opcommand_logs",
        "fileExtension": ".log",
        "toNode": "ship2",
        "data": [],
        "method":"REQOpCommand",
        "operation":{
            "opCmd":"ps"
        },
        "ACKTimeout":3,
        "retries":3,
        "replyACKTimeout":3,
        "replyRetries":3,
        "MethodTimeout": 7
    }
]
```

To send and Op Command to stop a subscriber on a node

```json
[
    {
        "directory":"opcommand_logs",
        "fileExtension": ".log",
        "toNode": "ship2",
        "data": [],
        "method":"REQOpCommand",
        "operation":{
            "opCmd":"stopProc",
            "opArg": {
                "method": "REQHttpGet",
                "kind": "subscriber",
                "receivingNode": "ship2"
            }
        },
        "ACKTimeout":3,
        "retries":3,
        "replyACKTimeout":3,
        "replyRetries":3,
        "MethodTimeout": 7
    }
]
```

To send and Op Command to start a subscriber on a node

```json
[
    {
        "directory":"opcommand_logs",
        "fileExtension": ".log",
        "toNode": "ship2",
        "data": [],
        "method":"REQOpCommand",
        "operation":{
            "opCmd":"startProc",
            "opArg": {
                "method": "REQHttpGet",
                "allowedNodes": ["central","node1"]
            }
        },
        "ACKTimeout":3,
        "retries":3,
        "replyACKTimeout":3,
        "replyRetries":3,
        "MethodTimeout": 7
    }
]
```

Tail a log file on a node, and save the result of the tail centrally at the directory specified.

```json
[
    {
        "directory": "./my/log/files/",
        "fileExtension": ".log",
        "toNode": "ship2",
        "data": ["./test.log"],
        "method":"REQTailFile",
        "ACKTimeout":5,
        "retries":3,
        "methodTimeout": 200
    }
]
```

You can save the content to myfile.JSON and append it to the `socket` file.

`nc -U ./steward.sock < example/toShip1-REQCliCommand.json`

## Concepts/Ideas

### Naming

#### Subject

`<nodename>.<method>.<command/event>`

Nodename: Are the hostname of the device. This do not have to be resolvable via DNS, it is just a unique name for the host to receive the message.

Command/Event: Are type of message sent. `CommandACK`/`EventACK`/`CommandNACK`/`EventNACK`. Description of the differences are mentioned earlier.\
Info: The command/event which is called a MessageType are present in both the Subject structure and the Message structure. The reason for this is that it is used both in the naming of a subject, and in the message for knowing what kind of message it is and how to handle it.

Method: Are the functionality the message provide. Example could be `CLICommand` or `Syslogforwarding`

##### Complete subject example

For Hello Message to a node named "central" of type Event and there is No Ack.

`central.REQHello.EventNACK`

For CliCommand message to a node named "ship1" of type Command and it wants an Ack.

`ship1.REQCliCommand.CommandACK`

## TODO

### nACK messages like hello messages should not be stored in db for retry if failed

### Error message should also contain subject
  
```log
2021-06-09 13:04:33.259759 +0000 UTC, info: toNode: ship1, fromNode: ww.central, method:REQOpCommand: max retries reached, check if node is up and running and if it got a subscriber forthe given REQ type
```

### Remove the message from log statements, and just point to the ID of the message instead

### Workflow request type

Implement a workflow request type that will allow steps to be executed, and rollback if something failed during execution.

### Add Op option the remove messages from the queue on nodes

If messages have been sent, and not picked up by a node it might make sense to have some method to clear messages on a node. This could either be done by message ID, and/or time duration.

## Disclaimer

All code in this repository are to be concidered not-production-ready, and the use is at your own responsibility. The code are the attempt to concretize the idea of a purely async management system where the controlling unit is decoupled from the receiving unit, and that that we know the state of all the receiving units at all times.

Also read the license file for further details.
