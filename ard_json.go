package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

const (
	ARD_AGENT_MESSAGE_TYPE_REGISTER = "REGISTER"
	ARD_AGENT_MESSAGE_TYPE_FORWARD = "FORWARD"
)

type ARDConfigStream struct {
	AgentId		string				`json:"agent_id"`
	StreamIds	[]string 			`json:"stream_ids"`
}

type ARDConfigAgent struct {
	Self		ARDConfigStream		`json:"self"`
	Peers		[]ARDConfigStream	`json:"peers"`
}

type ARDConfigAgentHub struct {
	Agents		[]ARDConfigAgent		`json:"agents"`
}


type ARDMessageCommon struct {
	MessageType		string 		`json:"message_type"`
	SrcAgentId		string		`json:"src_agent_id"`
	MessageId		string		`json:"message_id"`
	StreamId		string		`json:"stream_id"`
}

type ARDMessageRegister struct {
	ARDMessageCommon
}

type ARDMessageForward struct {
	ARDMessageCommon
	DstAgentId		string		`json:"dst_agent_id"`
	Data			string		`json:"data"`
}


func ArdMessageCommonParse(data []byte) *ARDMessageCommon {
	msgCommon := &ARDMessageCommon{}
	err := json.Unmarshal(data, msgCommon)
	if err != nil {
		fmt.Print("failed to unmarshal msg common : ")
		fmt.Println(err.Error())
		return nil
	}

	return msgCommon
}

func ArdMessageRegisterParse(data []byte) *ARDMessageRegister {
	msgRegister := &ARDMessageRegister{}
	err := json.Unmarshal(data, msgRegister)
	if err != nil {
		fmt.Print("failed to unmarshal msg REGISTER : ")
		fmt.Println(err.Error())
		return nil
	}

	return msgRegister
}

func ArdMessageForwardParse(data []byte) *ARDMessageForward {
	msgForward := &ARDMessageForward{}
	err := json.Unmarshal(data, msgForward)
	if err != nil {
		fmt.Print("failed to unmarshal msg FORWARD : ")
		fmt.Println(err.Error())
		return nil
	}

	return msgForward
}

func ArdAgentHubConfigParse(path string) *ARDConfigAgentHub {
	config := &ARDConfigAgentHub{}

	configFile, err := os.Open(path)
	if err != nil {
		fmt.Print("failed to open the file " + path + " : ")
		fmt.Println(err.Error())
		return nil
	}

	configBytes, err := ioutil.ReadAll(configFile)
	if err != nil {
		fmt.Print("failed to read the file " + path + " : ")
		fmt.Println(err.Error())
		return nil
	}

	err = json.Unmarshal(configBytes, config)
	if err != nil {
		fmt.Print("failed to unmarshal the file " + path + " : ")
		fmt.Println(err.Error())
		return nil
	}

	return config
}