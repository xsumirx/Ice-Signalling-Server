# Registration & Signalling Server

## Goals

- Register Client and Server with Unique ID
- Mapping [Client ↔ Server]
- Forward Messages

### Registration - Config

1. Server and Client will be put in SQLite Database [Config Files]
2. Mapping will also be put into Database or Config File

```jsx
{
	"agents": [
		{
			"agent_id":"4-char alpha-numeric"
			"peer_agents":[
				{
					"agent_id": "4-char alpha-numeric"
					"streams": ["control", "video", "primary"]
				},
				{
					"agent_id": "4-char alpha-numeric"
					"streams": ["control", "video", "primary"]
				},
				...
			]
		},
		...
	]
}
```

### Registration - Message

This message is always sent by agent just after making the connection to let signalling server know about it presence.

```jsx
{
	"src_agent_id": "C0000000",
	"message_type": "REGISTER",
	"message_id": "1234",
	"stream_id": "primary0"
}
```

### Forward - Message

This message is sent by agent to forward data to other agents, and based on the configuration present at signalling server either message will be forwarded or discarded with error in return.

```jsx
{
	"src_agent_id": "C0000000",
	"dst_agent_id": "S0000000",
	"message_type": "FORWARD",
	"message_id": "12345",
	"stream_id": "primary0",
	"data" : "Hello how are you"
}
```
