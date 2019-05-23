package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"


	"github.com/afex/hystrix-go/hystrix"
)

func main() {

	hystrix.ConfigureCommand("my_command", hystrix.CommandConfig{
		// How long to wait for command to complete, in milliseconds
		Timeout: 1000,

		// MaxConcurrent is how many commands of the same type
		// can run at the same time
		MaxConcurrentRequests: 4,

		// VolumeThreshold is the minimum number of requests
		// needed before a circuit can be tripped due to health
		RequestVolumeThreshold: 5,

		// SleepWindow is how long, in milliseconds,
		// to wait after a circuit opens before testing for recovery
		SleepWindow: 10000,

		// ErrorPercentThreshold causes circuits to open once
		// the rolling measure of errors exceeds this percent of requests
		ErrorPercentThreshold: 10,
	})

	http.HandleFunc("/", logger(HandleSubsystem))

	fmt.Println("==> Main server is started")
	log.Println("listening on :8000")
	http.ListenAndServe(":8000", nil)
}

// HandleSubsystem send request to sub-system and extracts its response
func HandleSubsystem(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	resultCh := make(chan []byte)
	errCh := hystrix.Go("my_command", func() error {

		resp, err := http.Get("http://localhost:8080/response/1/a")
		if err != nil {
			return err
		}

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		resultCh <- b
		return nil
	}, nil)

	select {
	case res := <-resultCh:
		println("HysterixGo")
		log.Println("success to get response from sub-system:", string(res))
		w.WriteHeader(http.StatusOK)
	case err := <-errCh:
		log.Println("failed to get response from sub-system:", err.Error())
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

// log is Handler wrapper function for logging
func logger(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL.Path, r.Method)
		fn(w, r)
	}
}