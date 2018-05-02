// Package steptest makes transactional load test easy.
package steptest

import (
	"fmt"
	"net/http"
	"time"
)

// New takes a number of virtual user v and request timeout t and creates a StepTest Server.
// If c is not nil that function will be used for all requests. This is usefull when you
// need to sign your requests or do anything else fancy with the them :) :) :).
// If c is set timeout will be ignored for obvious reasons, so please handle this in your
// own function if that is the case.
// Returns *Server and error.
func New(v int, t int, c func(*http.Request) (*http.Response, error)) (*Server, error) {
	// Default to 100 workers.
	if v < 1 {
		v = 100
	}

	// Default to 30000ms in fetch timeout.
	if t < 1 {
		t = 30000
	}

	// Default to a simple function if no one was specified.
	if c == nil {
		client := &http.Client{Timeout: time.Duration(t) * time.Millisecond}
		c = func(req *http.Request) (*http.Response, error) { return client.Do(req) }
	}

	srv := &Server{
		fetchWorkers:         v,
		fetchFunc:            c,
		addedJobsCounterChan: make(chan int),
		parsedJobs:           make(chan *job),
		resultJobs:           make(chan []*Result),
		resultCounterChan:    make(chan int),
	}

	return srv, nil
}

// Start will start the execution of parsed jobs. If there are any un-parsed jobs left
// in the queue it will wait for them to finish before starting execution.
func (srv *Server) Start() {
	go srv.workerResults()
	go srv.workerResultsCounter()
	go srv.workerAddedJobCounter()

	srv.running = true
	srv.startTime = time.Now()

	for i := 0; i < srv.fetchWorkers; i++ {
		go srv.workerFetch()
	}
}

// StopRunning will send a signal to stop fetching requests on the Server.
// StopRunning can only be called when the Server is in a IsRunning -state.
// Returns error.
//FIXME: We need to re-implement this so we can stop a running program without getting send on closed channel.
func (srv *Server) StopRunning() error {
	switch {
	case !srv.IsRunning():
		return fmt.Errorf("Failed to StopRunning. The Server isn't in a Running state")
	}

	srv.closeRunning()
	return nil
}

// WaitDone will wait until the Server has finished fetching all the requests in the *Server.jobs map.
// WaitDone will block the program until it has finished.
func (srv *Server) WaitDone() {
	srv.wgRun.Wait()
	srv.closeRunning()
}

// closeRunning will close everything associated with the Server in running state.
func (srv *Server) closeRunning() {
	srv.stopping = true
	srv.running = false
	close(srv.parsedJobs)
	srv.wgRes.Wait()
	srv.endTime = time.Now()
}

// GetNumberOfVirtualUsers returns the number of virtual users.
// Returns int.
func (srv *Server) GetNumberOfVirtualUsers() int {
	return srv.fetchWorkers
}

// GetNumberOfJobs returns the number of jobs stored on the Server.
// Returns int
func (srv *Server) GetNumberOfJobs() int {
	return srv.addedJobsCounter
}

// GetResults will return the all the results.
// Returns []*Result.
func (srv *Server) GetResults() []*Result {
	return srv.results
}

// GetSteps will return all the steps from a *Result.
// Returns []*ResultStep.
func (res *Result) GetSteps() []*ResultStep {
	return res.Steps
}

// GetNumberOfRequests returns the number of successfull requests.
// Returns int.
func (srv *Server) GetNumberOfRequests() int {
	return srv.resultsCounter
}

// GetError will return any error from a *Result.
// Returns []*ResultError.
func (res *Result) GetError() *ResultError {
	return res.Err
}

// GetNumberOfErrors will return the amount of requests that had an error.
// Returns int.
func (srv *Server) GetNumberOfErrors() int {
	errors := 0
	for _, res := range srv.results {
		if res.Err != nil {
			errors++
		}
	}
	return errors
}

// GetErrorMessages will return all the errors since the Server was started.
// Returns []*ResultError.
func (srv *Server) GetErrorMessages() []*ResultError {
	errors := []*ResultError{}
	for _, res := range srv.results {
		if res.Err != nil {
			errors = append(errors, res.Err)
		}
	}
	return errors
}

// GetAverageFetchTime will return the average fetch time for all the requests. Requests that resulted in errors will be ignored in the average.
// Returns time.Duration.
func (srv *Server) GetAverageFetchTime() time.Duration {
	duration := int64(0)
	for _, res := range srv.results {
		if res.Err != nil {
			continue
		}

		duration += int64(res.Duration)
	}

	numRequests := int64(len(srv.results))
	if duration == 0 || numRequests == 0 {
		return 0
	}

	return time.Duration(duration/numRequests) / time.Millisecond
}

// IsRunning returns true if the Server is still running jobs. False if it has finished or manually been stopped.
// Returns bool.
func (srv *Server) IsRunning() bool {
	return srv.running
}

// GetTotalRunTime will return the total runtime since Server start.
// Returns time.Duration.
func (srv *Server) GetTotalRunTime() time.Duration {
	duration := srv.endTime.Sub(srv.startTime)
	if duration == 0 {
		return 0
	}

	return duration / time.Second
}
