// Package steptest makes transactional load test easy.
package steptest

import (
	"fmt"
)

// AddJob will parse a job and add it to the *Server.
// It takes steps s, v variables as a map of strings,
// time a when to start the job. Time a can be nil for direct/orderless execution.
// Returns error.
func (srv *Server) AddJob(s string, v map[string]string) error {
	if srv.parsedJobs == nil {
		return fmt.Errorf("Error adding job in *Server.AddJob. *Server.parsedJobs channel is closed")
	}

	j, err := srv.parseJob(&rawJob{s, v})
	if err != nil {
		return err
	}
	go srv.workerAddJob(j)

	return nil
}

// workerAddJob will send the job j to the *Server.parsedJob channel.
// Since this channel is unbuffered we need to run this function in a separate go-routine.
// We need to deep copy all maps and/or slices in each iteration due to the nature of how
// these are passed (by reference) by go.
func (srv *Server) workerAddJob(j *job) {
	srv.wgRun.Add(1)
	srv.parsedJobs <- j
	srv.addedJobsCounterChan <- 1
}

// workerAddedJobCounter will listen for incoming increment changes on the
// *Server.addedJobsCounterChan channel and add those to the *Server.addedJobsCounter.
// This way we can track the amount of added jobs.
func (srv *Server) workerAddedJobCounter() {
	for {
		inc := <-srv.addedJobsCounterChan
		srv.addedJobsCounter += inc
	}
}
