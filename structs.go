// Package steptest makes transactional load test easy.
package steptest

import (
	"net/http"
	"sync"
	"time"
)

// Server contains the necessary functions and data to run StepTest.
// Should be instantiated with New function.
type Server struct {
	fetchWorkers int
	fetchFunc    func(*http.Request) (*http.Response, error)

	startTime time.Time
	endTime   time.Time

	addedJobsCounter     int
	addedJobsCounterChan chan int

	parsedJobs chan *job

	results           []*Result
	resultJobs        chan []*Result
	resultsCounter    int
	resultCounterChan chan int

	stopping bool
	running  bool
	wgRun    sync.WaitGroup
	wgRes    sync.WaitGroup
}

type rawJob struct {
	steps string
	vars  map[string]string
}

type job struct {
	steps         []step
	vars          map[string]string
	arrays        map[string][]string
	globalHeaders []header
	globalAuth    auth
	cookies       []http.Cookie

	// For variables. The addTo contains which step index to add sub steps to. For now we only use one value in the slice
	// since nested for loops are not supported.
	forcounter        int
	forRemoveNextStep bool
	addTo             []int
}

type step struct {
	method  string
	headers []header

	forloop forloop

	conditions []condition

	// Only used for storing results of replaced cookies. All cookies are global.
	cookies []http.Cookie
	auth    auth
	url     string
	body    string
	varfrom []varfromItem
}

type forloop struct {
	varname string
	values  []string

	steps []step
}

type cookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Path     string    `json:"path"`
	Domain   string    `json:"domain"`
	Expires  time.Time `json:"expires"`
	MaxAge   int       `json:"maxAge"`
	Secure   bool      `json:"secure"`
	HTTPOnly bool      `json:"httpOnly"`
}

type variable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type array struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type auth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type header struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type varfromItem struct {
	From      string `json:"from"`
	Varname   string `json:"name"`
	OrgSyntax string `json:"find"`
	Syntax    string `json:"-"`
}

type condition struct {
	Type string `json:"type"`
	Var1 string `json:"var1"`
	Var2 string `json:"var2"`
}

// Result contains the result of a job.
type Result struct {
	StartTime time.Time     `json:"startTime"`
	Status    int           `json:"status"`
	Duration  time.Duration `json:"duration"`
	Steps     []*ResultStep `json:"steps"`
	Err       *ResultError  `json:"error"`
}

// ResultStep contains the processed step results.
type ResultStep struct {
	StartTime time.Time     `json:"startTime"`
	Status    int           `json:"status"`
	Duration  time.Duration `json:"duration"`
	Method    string        `json:"method"`
	URL       string        `json:"url"`
	Headers   []header      `json:"headers"`
	Cookies   []http.Cookie `json:"cookies"`
	Body      string        `json:"body"`
}

// ResultError contains the error and the step of the error.
type ResultError struct {
	Error  error       `json:"error"`
	URL    string      `json:"url"`
	Status int         `json:"status"`
	Body   string      `json:"body"`
	Step   *ResultStep `json:"step"`
}
