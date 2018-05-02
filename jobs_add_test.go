// Package steptest makes transactional load test easy.
package steptest

import (
	"testing"
)

func TestAddJob(t *testing.T) {
	// Create the server.
	srv, err := New(10, 30000, nil)
	if err != nil {
		t.Error(err)
	}

	steps := "- GET https://{{url}}\n"
	steps += `  var { "name": "url", "value": "google.com" }`
	steps += "\n\n"
	steps += "- POST https://www.sunet.se/"

	err = srv.AddJob(steps, nil)
	if err != nil {
		t.Error(err)
	}

	// Receive jobs.
	jobs := []*job{}
	for i := 0; i < 5; i++ {
		job := <-srv.parsedJobs
		jobs = append(jobs, job)
		<-srv.addedJobsCounterChan
	}

	// Check that the added job has been added to run 5 times.
	if len(jobs) != 5 {
		t.Errorf("Wrong amount of runs for the specified job seems to have added. Expected %d but got %d", 5, len(jobs))
	}

	// Check that the data for each job is correct.
	for _, job := range jobs {
		if job.vars["url"] != "google.com" {
			t.Errorf("Expected var url to be %s but got %s", "google.com", job.vars["url"])
		}

		// Check first step (google)
		if job.steps[0].method != "GET" {
			t.Errorf("Expected first METHOD of steps to be %s but got %s", "GET", job.steps[0].method)
		}
		if job.steps[0].url != "https://{{url}}" {
			t.Errorf("Expected first URL of steps to be %s but got %s", "https://{{url}}", job.steps[0].url)
		}

		// Check second step (sunet.se)
		if job.steps[1].method != "POST" {
			t.Errorf("Expected second METHOD of steps to be %s but got %s", "POST", job.steps[0].method)
		}
		if job.steps[1].url != "https://www.sunet.se/" {
			t.Errorf("Expected second URL of steps to be %s but got %s", "https://www.sunet.se/", job.steps[0].url)
		}
	}
}
