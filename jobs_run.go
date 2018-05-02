// Package steptest makes transactional load test easy.
package steptest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// workerFetch is called upon when the Server is started with *Server.Start.
// The number of worker spawned is determined by the value supplied in the *Server.New call.
// It fetches by listening to the *Server.parsedJobs channel and calling the *Server.fetchJob
// function for each job sent. The result will be added to a result slice and sent to the
// *Server.resultJobs channel.
func (srv *Server) workerFetch() {
	srv.wgRes.Add(1)
	results := []*Result{}

	for srv.running {
		j := <-srv.parsedJobs
		if j == nil {
			break
		}

		res := srv.fetchJob(j)
		results = append(results, res)
		srv.resultCounterChan <- 1
		srv.wgRun.Done()
	}

	srv.resultJobs <- results
}

// fetchJob will loop through the job j steps and call *job.fetchStep through the *job.runFetchJob on each iteration.
// Any errors will be added to the results error value.
// Returns *result.
func (srv *Server) fetchJob(j *job) *Result {
	r := &Result{StartTime: time.Now()}

	for i := 0; i < len(j.steps); i++ {
		switch {
		// If a for loop is detected, we must run multiple steps inside a single step.
		// First we must replace any variables in the in data for the for loop (raw values).
		// Since these can be based on results from a body, header etc.
		// After we have run the replaceFromVariablesForLoop and the value slice is set we
		// can iterate over the slice and set the for variable to the current value and
		// run all steps in the for loop with that value.
		// v = index of for variable, s = index for step in the for loop.
		case j.steps[i].forloop.varname != "":
			j.replaceFromVariablesForLoop(&j.steps[i])

			for v := range j.steps[i].forloop.values {
				// Set the variable to be used for this iteration of the for loop.
				j.vars[j.steps[i].forloop.varname] = j.steps[i].forloop.values[v]

				for s := range j.steps[i].forloop.steps {
					res, err := j.runFetchJob(srv.fetchFunc, j.steps[i].forloop.steps[s].deepCopyStep())
					r.Steps = append(r.Steps, res)
					r.Status = res.Status

					if err != nil {
						r.Err = err
						break
					}
				}
			}

		// The default fetching method, when we just have normal global steps (ie, not in a for loop).
		default:
			res, err := j.runFetchJob(srv.fetchFunc, &j.steps[i])
			r.Steps = append(r.Steps, res)
			r.Status = res.Status

			if err != nil {
				r.Err = err
				break
			}
		}

		// If any errors where set above, we should not do any more steps.
		// And just break out of the for loop and save the results.
		if r.Err != nil {
			break
		}
	}

	r.Duration = time.Now().Sub(r.StartTime)

	return r
}

// deepCopyStep is used to make a deep copy of a step. Which means that we will copy every array/map it contains
// so that every step can be run independently of another. Otherwise changes to one step on data structures that
// are referenced by memory, such as slices, maps will be updated when we replace vars and such. Which is not
// what we want when running multiple steps inside a step (for loops).
// So we make a deep copy of all the data from step s and return a new step.
// Returns *step.
func (s *step) deepCopyStep() *step {
	newStep := &step{
		method:  s.method,
		forloop: forloop{}, // No nested for loop support, so should be empty.
		auth:    s.auth,
		url:     s.url,
		body:    s.body,
	}

	// Make copy of conditions/if slice.
	for _, i := range s.conditions {
		newStep.conditions = append(newStep.conditions, i)
	}

	// Make copy of varfrom slice.
	for _, v := range s.varfrom {
		newStep.varfrom = append(newStep.varfrom, v)
	}

	// Make copy of headers slice.
	for _, h := range s.headers {
		newStep.headers = append(newStep.headers, h)
	}

	// Make copy of cookies.
	for _, c := range s.cookies {
		newStep.cookies = append(newStep.cookies, c)
	}

	return newStep
}

// runFetchJob will run the actual fetchStep function on the step s with
// func(*http.Request) (*http.Response, error) c. This is split out so that
// the function can be used both for iterations over a for loop
// (multiple steps within a step) or just a basic single step.
// Returns *ResultSteps and *ResultError.
func (j *job) runFetchJob(c func(*http.Request) (*http.Response, error), s *step) (*ResultStep, *ResultError) {
	// Dont run fetch on steps with no URL.
	if s.url == "" {
		return &ResultStep{}, nil
	}

	stepStart := time.Now()
	status, err := j.fetchStep(c, s)

	res := &ResultStep{
		Method:    s.method,
		URL:       s.url,
		Headers:   s.headers,
		Cookies:   s.cookies,
		Body:      s.body,
		StartTime: stepStart,
		Duration:  time.Now().Sub(stepStart),
		Status:    status,
	}

	if err != nil {
		err.Step = res
		return res, err
	}

	return res, nil
}

// fetchStep will make an request against the steps url method.
// If any of the if/conditions are matched we will not fetch anything and directly return a statusCode of 0.
// We will replace any variables from the URL, Body Header and Cookies with the *job.replaceFromVariables.
// Auth, Headers and Cookies are then added to the request addMetaData function.
// Will return the statusCode of the request as well as any error. The error will include the
// step which failed including all the data so it can be easily tracked in logfiles.
// Any response status code 400 or above will result in an error.
// Returns int and *ResultError.
func (j *job) fetchStep(c func(*http.Request) (*http.Response, error), s *step) (int, *ResultError) {
	if !j.checkConditions(s) {
		return 0, nil
	}

	j.replaceFromVariables(s)

	req, err := http.NewRequest(s.method, s.url, bytes.NewBuffer([]byte(s.body)))
	if err != nil {
		return -1, &ResultError{Error: fmt.Errorf("Error creating up the Request in *job.fetchStep. %s", err)}
	}
	j.addOptions(s, req)

	res, err := c(req)
	if err != nil {
		return -1, &ResultError{Error: fmt.Errorf("Error sending the Request in *job.fetchStep. %s", err)}
	}
	defer res.Body.Close()

	if res.StatusCode > 399 {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			body = []byte("")
		}
		return res.StatusCode, &ResultError{Error: fmt.Errorf("%d %s %s", res.StatusCode, s.method, s.url), URL: s.url, Status: res.StatusCode, Body: string(body)}
	}

	j.appendResponseCookiesToJob(res)

	err = j.variablesFrom(s, res)
	if err != nil {
		return -1, &ResultError{Error: err}
	}

	return res.StatusCode, nil
}
