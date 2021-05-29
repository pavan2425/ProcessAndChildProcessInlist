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

var MAP_KEYS = make(map[int]bool)

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
func AddProcessDetails(proc *process.Process, level int) ProcessData {
	pid := proc.Pid
	name, _ := proc.Name()
	ppid, _ := proc.Ppid()
	memory, _ := proc.MemoryPercent()
	cpu, _ := proc.CPUPercent()
	ChildProcess, _ := proc.Children()
	var childData []ProcessData
	if len(ChildProcess) > 0 {
		for i, _ := range ChildProcess {
			childData = append(childData, AddProcessDetails(ChildProcess[i], level+1))

		}
	}
	processDetails := ProcessData{
		ProcessName:   name,
		Pid:           int(pid),
		Ppid:          int(ppid),
		order:         level,
		MemoryPercent: memory,
		CpuPercent:    cpu,
		ChildProcess:  childData,
	}
	if processDetails.order != 0 {
		MAP_KEYS[int(pid)] = true
	}
	return processDetails
}

//handler to get process details in list
func ProcessDetails(w http.ResponseWriter, r *http.Request) {

	processes, err := process.Processes()
	if err != nil {
		fmt.Printf("error while getting process info", err)
	}
	var processList []ProcessData
	for i, proc := range processes {

		if proc.Pid > 100 && proc.Pid < 1000 {
			if !MAP_KEYS[int(proc.Pid)] {
				processList = append(processList, AddProcessDetails(processes[i], 0))

			}
		}
	}

	sort.SliceStable(processList, func(i, j int) bool {
		return processList[i].CpuPercent > processList[j].CpuPercent
	})
	processDetails := SnapprocessInfo{
		ProcessList: processList,
	}

	respondJSON(w, http.StatusOK, processDetails)
	for k := range MAP_KEYS {
		delete(MAP_KEYS, k)
	}
}
