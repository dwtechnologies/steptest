package steptest

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	srv, err := New(50, 15000, nil)
	if err != nil {
		t.Error(err)
	}

	// Check the number of fetchWorkers
	if srv.fetchWorkers != 50 {
		t.Errorf("Wrong number of workers. Expected %d but got %d", 50, srv.fetchWorkers)
	}

	if srv.addedJobsCounterChan == nil {
		t.Error("*Server.addedJobsCounterChan was nil. Expected chan int but got nil")
	}

	if srv.parsedJobs == nil {
		t.Error("*Server.parsedJobs was nil. Expected chan *job but got nil")
	}

	if srv.resultJobs == nil {
		t.Error("*Server.resultJobs was nil. Expected chan []*Result but got nil")
	}

	if srv.resultCounterChan == nil {
		t.Error("*Server.resultCounterChan was nil. Expected chan in but got nil")
	}

	if srv.fetchFunc == nil {
		t.Error("*Server.fetchFunc was nil. Expected func(*http.Request) (*http.Response, error) but got nil")
	}
}

func TestStart(t *testing.T) {
	srv, err := New(1, 30000, nil)
	if err != nil {
		t.Error(err)
	}

	srv.Start()
	if srv.running != true {
		t.Error("Server isn't running. Expected *Server.running to be true")
	}
}

func TestStopRunning(t *testing.T) {
	srv, err := New(1, 30000, nil)
	if err != nil {
		t.Error(err)
	}

	srv.Start()
	err = srv.StopRunning()
	if err != nil {
		t.Error(err)
	}

	if srv.running != false {
		t.Errorf("Server is still in running state. Expected *Server.running to be false")
	}

	if srv.stopping != true {
		t.Errorf("Server is not in stopping state. Expected *Server.stopping to be true")
	}
}

func TestWaitDone(t *testing.T) {
	srv, err := New(1, 30000, nil)
	if err != nil {
		t.Error(err)
	}

	srv.Start()
	srv.WaitDone()

	if srv.running != false {
		t.Errorf("Server is still in running state. Expected *Server.running to be false")
	}

	if srv.stopping != true {
		t.Errorf("Server is not in stopping state. Expected *Server.stopping to be true")
	}
}

func TestGetNumberOfVirtualUsers(t *testing.T) {
	srv, err := New(153, 30000, nil)
	if err != nil {
		t.Error(err)
	}

	if srv.GetNumberOfVirtualUsers() != 153 {
		t.Errorf("Wrong number of Virtual Users. Expected %d but got %d", 153, srv.GetNumberOfVirtualUsers())
	}
}

func TestGetNumberOfJobs(t *testing.T) {
	srv := &Server{addedJobsCounter: 12}

	if srv.GetNumberOfJobs() != 12 {
		t.Errorf("Wrong number of Added Jobs. Expected %d but got %d", 12, srv.GetNumberOfJobs())
	}
}

func TestGetResults(t *testing.T) {
	start := time.Now()
	end := start.Sub(start.Add(time.Duration(10) * time.Second))
	srv := &Server{
		results: []*Result{
			&Result{
				StartTime: start,
				Duration:  end,
				Err:       nil,
				Status:    200,
				Steps:     nil,
			},
		},
	}

	res := srv.GetResults()

	if len(res) != 1 {
		t.Errorf("Wrong number of Results. Expected %d but got %d", 1, len(res))
	}

	if res[0].StartTime != start {
		t.Errorf("Wrong StartTime. Expected %s but got %s", start.String(), res[0].StartTime.String())
	}

	if res[0].Duration != end {
		t.Errorf("Wrong StartTime. Expected %s but got %s", end.String(), res[0].Duration.String())
	}

	if res[0].Err != nil {
		t.Errorf("Err was not nil")
	}

	if res[0].Status != 200 {
		t.Errorf("Wrong Status. Expected %d but got %d", 200, res[0].Status)
	}

	if res[0].Steps != nil {
		t.Errorf("Steps was not nil")
	}
}
