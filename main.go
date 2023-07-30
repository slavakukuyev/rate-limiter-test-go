//example of run:   go run main.go -threshold 10 -ttl 1000 -capacity=1000 -workers=20

/**
Alternatives with GoModules:
	HTTP -> fastHTTP
	log -> zap logger
	flag for args -> viper for environment variables
*/

package main

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"sync"
	"time"
)

// --- Schemas
type RequestSchema struct {
	URL        string `json:"url"`
	ResultChan chan bool
}

type ResponseSchema struct {
	Block bool `json:"block"`
}

type MainMap struct {
	Mutex *sync.RWMutex
	Data  map[string]int
}

//---

var threshold int
var ttl int
var requestsQueue chan RequestSchema
var GlobalMapper MainMap

func main() {

	/**
		run the profiler command in different terminal
		pprof
	 go tool pprof -png http://localhost:6060/debug/pprof/allocs > allocs.png
	 go tool pprof -png http://localhost:6060/debug/pprof/mutex > mutex.png
	 go tool pprof -png http://localhost:6060/debug/pprof/heap > heap.png
	*/
	// *** uncomment for profiling ***
	/*go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()*/

	//--- define flags for default arguments
	flag.IntVar(&threshold, "threshold", 10, "Max number of requests per URL")
	flag.IntVar(&ttl, "ttl", 60000, "The time period in which URL visits will be counted.")

	var queueMaxSize int
	flag.IntVar(&queueMaxSize, "capacity", 1000, "Max count of requests in the global queue")
	var workersCount int
	flag.IntVar(&workersCount, "workers", 20, "Ma count of concurrently processed requests")

	var port string
	flag.StringVar(&port, "p", "8090", "Port")

	//Must be called after all flags are defined and before flags are accessed by the program.
	flag.Parse()
	//---

	//define buffered channel
	requestsQueue = make(chan RequestSchema, queueMaxSize)
	//read from requests chan
	RangeOverAdRequestsQueue(requestsQueue, workersCount)

	//init main map
	GlobalMapper = MainMap{
		Mutex: &sync.RWMutex{},
		Data:  make(map[string]int),
	}

	http.HandleFunc("/report", httpHandler)
	// Log server started
	log.Printf("Rate Limiter Server started at port %v", port)
	log.Printf("Args: threshold (%v), ttl in ms (%v), queueMaxSize (%v), workers (%v)", threshold, ttl, queueMaxSize, workersCount)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Printf("Could not start Rate Limiter server:  %v", err.Error())
		os.Exit(0)
	}
}

func httpHandler(w http.ResponseWriter, req *http.Request) {

	c, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	errorMessage := "Bad request"

	var requestObj RequestSchema
	err := json.NewDecoder(req.Body).Decode(&requestObj)
	if err != nil {
		log.Printf("Error on decoding request body %v", err.Error())
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	if !IsValidURL(requestObj.URL) {
		log.Printf("URL is not valid %v", requestObj.URL)
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	//define resultChanel to receive message on end of processing request
	resultChan := make(chan bool, 1)
	requestObj.ResultChan = resultChan
	defer close(resultChan)

	requestsQueue <- requestObj

	errorMessage = "internal error"

	select {
	case <-c.Done():
		err := http.ErrHandlerTimeout
		http.Error(w, err.Error(), http.StatusGatewayTimeout)

	case block := <-resultChan:
		//processing request finished
		respObj := ResponseSchema{Block: block}

		response, err := json.Marshal(&respObj)
		if err != nil {
			log.Printf("Error on Marshalling Response Object %v", err)
			http.Error(w, errorMessage, http.StatusInternalServerError)
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(response)
		if err != nil {
			log.Printf("Error on Writing responce %v", err)
			http.Error(w, errorMessage, http.StatusInternalServerError)
		}
	}
}

// Run in background defined count of goroutines
func RangeOverAdRequestsQueue(requestsQueue <-chan RequestSchema, concurrentWorkers int) {
	for i := 0; i < concurrentWorkers; i++ {
		go workOnRequest(requestsQueue)
	}
}

// each worker provides iteration over Requests Queue
func workOnRequest(requestsQueue <-chan RequestSchema) {
	for j := range requestsQueue {
		processSingleRequest(j)
	}
}

func processSingleRequest(reqObj RequestSchema) {

	hash := convertToSha1(reqObj.URL)
	block := false

	GlobalMapper.Mutex.Lock()
	defer GlobalMapper.Mutex.Unlock()
	count, ok := GlobalMapper.Data[hash]
	if ok {
		if count >= threshold {
			block = true
		} else {
			count++
			GlobalMapper.Data[hash] = count
		}
	} else {
		//define map and timeout
		count = 1
		GlobalMapper.Data[hash] = count
		go thresholdTimeout(hash)
	}

	blockedMessage := "not blocked"
	if block {
		blockedMessage = "blocked"
	}

	log.Printf("URL %v is reported, count=%v, %v", reqObj.URL, count, blockedMessage)

	//condition: in case handler unexpectedly timed out and closed the channel
	if isChannelOpen(reqObj.ResultChan) {
		reqObj.ResultChan <- block
	}
}

/*
*

	simple hash conversion
*/
func convertToSha1(str string) string {
	hash := sha1.Sum([]byte(str))
	return fmt.Sprintf("%x", hash)
}

// delete url hash from global map on timeout
func thresholdTimeout(hash string) {
	select {
	case <-time.After(time.Duration(ttl) * time.Millisecond):
		GlobalMapper.Mutex.Lock()
		delete(GlobalMapper.Data, hash)
		GlobalMapper.Mutex.Unlock()
	}
}

/*
*

	parse valid url
*/
func IsValidURL(str string) bool {
	_, err := url.ParseRequestURI(str)
	if err != nil {
		return false
	}

	uri, errURI := url.Parse(str)
	if errURI != nil || uri.Scheme == "" || uri.Host == "" {
		return false
	}

	return true
}

func isChannelOpen(c chan bool) bool {
	select {
	case <-c:
		return false
	default:
		return true
	}
}
