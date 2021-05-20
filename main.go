package main

import (
	"encoding/json"

	"fmt"
	"sort"

	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/shirou/gopsutil/process"
)

//process Info to hold list of process details
type SnapprocessInfo struct {
	ProcessList []ProcessData
}

//process info
type ProcessData struct {
	ProcessName   string
	Pid           int
	Ppid          int
	order         int
	MemoryPercent float32
	CpuPercent    float64
	ChildProcess  []ProcessData
}

func main() {
	router := mux.NewRouter()
	log.Println("Creating api table")

	router.HandleFunc("/processdetails", ProcessDetails).Methods("GET")

	// log.Fatal(http.ListenAndServe(":3333", router))
	log.Fatal(http.ListenAndServe(":3333", handlers.CORS(handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD"}), handlers.AllowedOrigins([]string{"*"}))(router)))
}

// respondJSON makes the response with payload as json format
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(response))
}

// respondError makes the error response with payload as json format
func respondError(w http.ResponseWriter, code int, message string) {
	respondJSON(w, code, map[string]string{"error": message})
}

//Add process to  a list recursively
func AddProcessDetails(proc *process.Process, processList *[]ProcessData, level int) {
	pid := proc.Pid
	name, _ := proc.Name()
	ppid, _ := proc.Ppid()
	memory, _ := proc.MemoryPercent()
	cpu, _ := proc.CPUPercent()
	ChildProcess, _ := proc.Children()
	processDetails := ProcessData{
		ProcessName:   name,
		Pid:           int(pid),
		Ppid:          int(ppid),
		order:         level,
		MemoryPercent: memory,
		CpuPercent:    cpu,
	}
	*processList = append(*processList, processDetails)

	if len(ChildProcess) > 0 {
		for i, _ := range ChildProcess {
			AddProcessDetails(ChildProcess[i], processList, level+1)

		}
	}

}

//delete repeated process in the list
func unique(intSlice []ProcessData) []ProcessData {
	keys := make(map[int]bool)
	list := []ProcessData{}
	for _, entry := range intSlice {
		if _, value := keys[entry.Pid]; !value {
			keys[entry.Pid] = true
			list = append(list, entry)
		}
	}
	return list
}

//handler to get process details in list
func ProcessDetails(w http.ResponseWriter, r *http.Request) {

	processes, err := process.Processes()
	if err != nil {
		fmt.Printf("error while getting process info", err)
	}
	var processList []ProcessData
	for i, _ := range processes {

		AddProcessDetails(processes[i], &processList, 0)

	}

	sort.SliceStable(processList, func(i, j int) bool {
		return processList[i].order > processList[j].order
	})

	data := unique(processList)

	var finalList []ProcessData

	ChildProcessIds := make(map[int]bool)
	for i := 0; i <= len(data)-1; i++ {
		for j := i + 1; j <= len(data)-1; j++ {
			if data[i].Ppid == data[j].Pid {
				ChildProcessIds[data[i].Pid] = true
				data[j].ChildProcess = append(data[j].ChildProcess, data[i])
			}
		}
	}

	for i, _ := range data {
		if !ChildProcessIds[data[i].Pid] {
			finalList = append(finalList, data[i])
		}
	}
	sort.SliceStable(finalList, func(i, j int) bool {
		return finalList[i].CpuPercent > finalList[j].CpuPercent
	})
	processDetails := SnapprocessInfo{
		ProcessList: finalList,
	}

	respondJSON(w, http.StatusOK, processDetails)
}
