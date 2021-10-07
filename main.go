package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

var bindAddr = flag.String("addr", "0.0.0.0:9955", "ip:port to listen on")
var configPath = flag.String("config", "config.json", "config path")


func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Write([]byte("This is ARD Signalling server.\nRunning websocket at path /ws"))
}


func main() {
	flag.Parse()
	if(!flag.Parsed()) {
		flag.PrintDefaults()
	}
	hub := ARDHubNew(*configPath)
	if hub == nil {
		fmt.Println("failed to create hub.")
		return
	}
	go hub.ClientHubRun()
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ARDAgentServe(hub, w, r)
	})
	err := http.ListenAndServe(*bindAddr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}