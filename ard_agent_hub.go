package main

import (
	"fmt"
)

type ARDAgentStreamMap struct {
	Active		bool
	AgentId		string
	StreamId	string
}

type SignalConnectionStateMeta struct {
	Agent 	*ARDAgent
	State	int
}

type ARDStream struct {
	StreamId	string
	Agent		*ARDAgent
}

type ARDAgentState struct {
	//Id of the agent
	Id			string
	
	//List of streams supported by this agent
	Streams		[]ARDStream

	//List of streams for each mapped agent.
	//eg., map["agent-id"]["stream1", "stream2"]
	PeerAgents	map[string][]string
}

type ARDAgentHub struct {
	SignalConnectionState 	chan SignalConnectionStateMeta
	SignalMessage			chan ARDAgentMessage
	Agents					map[string]*ARDAgentState
	
	AgentsConnected			map[*ARDAgent]*ARDAgentStreamMap
}


func ARDHubNew(configPath string) *ARDAgentHub {
	config := ArdAgentHubConfigParse(configPath)
	if config == nil {
		fmt.Println("failed to get config")
		return nil
	}

	hub := &ARDAgentHub{
		SignalConnectionState: 	make(chan SignalConnectionStateMeta, 64),
		SignalMessage: 			make(chan ARDAgentMessage, 64),
		Agents: 				make(map[string]*ARDAgentState),
		AgentsConnected: 		make(map[*ARDAgent]*ARDAgentStreamMap),
	}

	for i := 0; i < len(config.Agents); i++ {
		agentState := &ARDAgentState{
			Id: config.Agents[i].Self.AgentId,
			Streams: make([]ARDStream, 0),
			PeerAgents: make(map[string][]string),
		}

		//Build Self stream array
		for j := 0; j < len(config.Agents[i].Self.StreamIds); j++ {
			stream := ARDStream {
				StreamId: config.Agents[i].Self.StreamIds[j],
				Agent: nil,
			}
			agentState.Streams = append(agentState.Streams, stream)
		}

		//Build peer agent array
		for j := 0; j < len(config.Agents[i].Peers); j++ {
			peers := make([]string, 0)
			for k := 0; k < len(config.Agents[i].Peers[j].StreamIds); k++ {
				peers = append(peers, config.Agents[i].Peers[j].StreamIds[k])
			}

			agentState.PeerAgents[config.Agents[i].Peers[j].AgentId] = peers
		}

		hub.Agents[config.Agents[i].Self.AgentId] = agentState
	}

	return hub
}

func (hub *ARDAgentHub)ClientHubRun() {
	for {
		select {
		case connectionState := <-hub.SignalConnectionState:
			if connectionState.State == ARD_AGENT_CONNECTED {
				//Connected
				hub.AgentsConnected[connectionState.Agent] = &ARDAgentStreamMap{
					AgentId: "",
					StreamId: "",
					Active: false,
				}
			}else if connectionState.State == ARD_AGENT_DISCONNECTED {
				//Disconnected
				agentStreamMap := hub.AgentsConnected[connectionState.Agent]
				if agentStreamMap == nil {
					fmt.Println("no record for client found.")
					break
				}
				
				agent := hub.Agents[agentStreamMap.AgentId]
				if agent == nil {
					fmt.Printf("no agent with agent_id %s is provisioned.\n", agentStreamMap.AgentId)
					delete(hub.AgentsConnected, connectionState.Agent)
					break
				}
				
				//Remove stream from respective agent
				for i := 0; i < len(agent.Streams); i++ {
					if agent.Streams[i].StreamId == agentStreamMap.StreamId {
						agent.Streams[i].Agent = nil
					}
				}

				delete(hub.AgentsConnected, connectionState.Agent)
			}
		case message := <-hub.SignalMessage:
			msgCommon := ArdMessageCommonParse(message.Data)
			if msgCommon == nil {
				fmt.Println("failed to parse message from agent : " + message.ARDAgent.ArdAgentGetAddressString())
				break
			}

			if msgCommon.MessageType == ARD_AGENT_MESSAGE_TYPE_REGISTER {
				//Register agent and stream
				msgRegister := ArdMessageRegisterParse(message.Data)
				if msgRegister == nil {
					fmt.Println("failed to parse message register from agent : " + message.ARDAgent.ArdAgentGetAddressString())
					break
				}
				
				agent := hub.Agents[msgRegister.SrcAgentId]
				if agent == nil {
					fmt.Printf("no agent with agent_id %s is provisioned.\n", msgCommon.SrcAgentId)
					break
				}

				isStreamRegistered := false
				for i := 0; i < len(agent.Streams); i++ {
					if agent.Streams[i].StreamId == msgRegister.StreamId {
						agent.Streams[i].Agent = message.ARDAgent
						isStreamRegistered = true
						break
					}
				}

				if isStreamRegistered {
					fmt.Printf("Agent (%s) with Stream (%s) from %s Registered.\n", msgRegister.SrcAgentId,
						msgRegister.StreamId, message.ARDAgent.ArdAgentGetAddressString())
				}else {
					fmt.Printf("Agent (%s) with Stream (%s) is not provisioned.\n", msgRegister.SrcAgentId,
						msgRegister.StreamId)
				}
				

			} else if msgCommon.MessageType == ARD_AGENT_MESSAGE_TYPE_FORWARD {
				//Route the message to dest_agent_id
				//Register agent and stream
				msgForward := ArdMessageForwardParse(message.Data)
				if msgForward == nil {
					fmt.Println("failed to parse message forward from agent : " + message.ARDAgent.ArdAgentGetAddressString())
					break
				}
				
				agent := hub.Agents[msgForward.SrcAgentId]
				if agent == nil {
					fmt.Printf("no agent with agent_id %s is provisioned.\n", msgCommon.SrcAgentId)
					break
				}
				
				//Check if src is allowed to send message to dst
				isAllowedToForward := false
				peerStreamIds := agent.PeerAgents[msgForward.DstAgentId]
				if peerStreamIds == nil {
					fmt.Printf("%s is not allowed to forward to %s.\n",
						msgForward.SrcAgentId, msgForward.DstAgentId)
					break
				}
				for _, peerStreamId := range peerStreamIds {
					if peerStreamId == msgForward.StreamId {
						isAllowedToForward = true
						break
					}
				}

				if !isAllowedToForward {
					fmt.Printf("%s is not allowed to forward to %s of stream type %s\n",
						msgForward.SrcAgentId, msgForward.DstAgentId, msgForward.StreamId)
					break
				}
				
				peerAgent := hub.Agents[msgForward.DstAgentId]
				if peerAgent == nil {
					fmt.Printf("no peer agent with agent_id %s is provisioned.\n", msgCommon.SrcAgentId)
					break
				}
				
				for _, peerStream := range peerAgent.Streams {
					if peerStream.StreamId == msgForward.StreamId {
						if peerStream.Agent == nil {
							fmt.Printf("agent not yet connected fro agent_id (%s) - stream_id(%s).\n",
								msgForward.DstAgentId, msgForward.StreamId)
						}else {
							peerStream.Agent.SignalMessage <- message.Data
						}

						break
					}
				}
				
			}

		}
	}
}
