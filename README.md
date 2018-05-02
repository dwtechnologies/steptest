# StepTest

Package and program made to make transactional load test easy.

## Motivation

Most other load testing packages/frameworks was made for testing REST APIs in a non transactional way.
This package was born out of the necessity to test our Magento based eCommerce platforms checkout over
multiple website ids and payment methods.

For this we needed a package that can, in a flexible way, test multiple steps in a transactional manner.
It must also be able to loop over values and also set variables based on the result body / headers from
previous transaction.

It also has the possibility to replay real scenarios. Jobs can be added with a "start after" value.
Making it possible to replay old load exactly as it happened.

## Usage

The first part of making StepTest is work is defining a steps -file.
This includes the different steps that the Server will run for each job that is added with the specified steps -file.

The steps -files syntax support a range of different functions such as `VAR`, `VARFROM`, `ARRAY`, `FOR`, `AUTH`, `HEADER`, `COOKIE` and of course HTTP
functions such as `GET`, `POST`, `PUT`, `PATCH`, `DELETE`. Functions can be declared either in upper or lower case.

Each step is divided by a dash `-`, any leading/trailing spaces and tabs will be removed.

## Example

### steps.txt

    - var { "name": "url", "value": "example.com" }
      array { "name": "productList", "values": [ "prodId1", "prodId2", "prodId3" ] }

    - get https://{{url}}/getSession
      varfrom { "from": "body", "name": "session", "syntax": "<input name=\"session\" type=\"hidden\" value=\"{{StepTestSyntax}}\" />"}

    - for product in {{productList}}
      post https://{{url}}/addProduct {"session":"{{session}}","product":"{{product}}"}
      forend

    - get https://{{url}}/getCart

### main.go

```go
package main

import (
    "fmt"
    "log"

    "github.com/dwtechnologies/steptest"
)

func main() {
    // Create a new StepTest server with 10 virtual users and 15s http timeout.
    srv, err := steptest.New(10, 15)
    if err != nil {
        log.Fatal(err)
    }

    // Add a job with no vars or startAfter value.
    srv.AddJob("steps.txt", 1, nil, nil)

    // Start the server and then wait until StepTest has finished all requests.
    srv.Start()
    srv.WaitDone()

    // Print some results.
    if errors := srv.GetNumberOfErrors(); errors > 0 {
        fmt.Printf(">>> Number of errors: %d\n\n", errors)

        for _, err := range srv.GetErrorMessages() {
            fmt.Printf("%s\n", err.Error)
        }
        fmt.Printf("\n")
    }

    fmt.Printf(">>> Total number of fetches: %d\n", srv.GetNumberOfRequests())
    fmt.Printf(">>> Average time: %d ms\n", srv.GetAverageFetchTime())
    fmt.Printf(">>> Total runtime: %d s\n", srv.GetTotalRunTime())
}
```

## Reference - Stepfile

Every line that starts with a dash followed by a space will be defined as a step separator.
Every function in a step is divided by every line that starts with two spaces.

### GET

`get http://example.com`

> Creates a new GET request against http://example.com

### POST

`post http://example.com {"name":"value"}`

> Creates a new POST request against http://example.com with a JSON body.

### PUT

`put http://example.com name%3Dvalue`

> Creates a new PUT request against http://example.com with a URL Encoded body.

### PATCH

`patch http://example.com/id/1234 {"partial":"info"}`

> Creates a new PATCH request against http://example.com with a JSON body.

### DELETE

`delete http://example.com/id/1234`

> Creates a new DELETE request against http://example.com with a empty body.

### VAR

`var { "name": "var1", "value": "val1" }`

> Creates a new variable called var1 with a value of val1.

### ARRAY

`array { "name": "arr1", "values": [ "val1", "val2", "val3" ] }`

> Creates a new array called arr1 with values val1, val2 and val3.

### VARFROM

`varfrom { "from": "body", "name": "var1", "syntax": "<input name=\"session\" type=\"hidden\" value=\"{{StepTestSyntax}}\" />" }`

> Creates a variable called var1. The value of var1 will be based on the requests BODY where it will look for the syntax `<input name=\"session\" type=\"hidden\" value=\"{{StepTestSyntax}}\" />`. And anything thats contained in the `{{StepTestSyntax}}` will be the value of the variable.

### COOKIE

`cookie { }`

> Creates a new cookie with the values...

### HEADER

`header { "name": "header1", "value": "val1" }`

> Creates a new header with name header1 and value val1. (local to the step)

### \@HEADER

`@header { "name": "header1", "value": "val1" }`

> Creates a new global header with name header1 and value val1. (global for whole job)

### AUTH

`auth { "username": "user1", "password": "pass1" }`

> Adds Auth to the request with username and password user1 and pass1. (local to the step)

### \@AUTH

`@auth { "username": "user1", "password": "pass1" }`

> Adds Global Auth to the request with username and password user1 and pass1. (global for whole job)

### FOR

`for i in {{arr1}}`
`for i in [ "val1", "val2", "val3" ]`
`for i in {{var1}}` // var1 needs to contain a stringified JSON array that can be unmarshaled.

> Creates a for loop that will loop through all the values in the array and set the variable i to the value
> from the array. More than one step can be included in the forloop. Should be ended with a forend.
> The step that the for is defined in will be included in the for loop.

### FOREND

`forend`

> Ends a for loop. Can be part of the same step as for. Then only that step will be looped over.

## Reference - Exported functions

### New

```go
steptest.New(v int, t int) (*Server, error)
```

> New takes a number of virtual users v and request timeout t and creates a StepTest Server.
> Returns *Server and error.

### AddJob

```go
*Server.AddJob(s string, r int, v map[string]string, a *time.Time) error
```

> AddJob will parse a job and add it to the *Server.
> It takes the path to a stepsfile s, r number of runs, v variables as a map of strings
> and time a when to start the job, for direct execution just nil.
> Returns error.

### Start

```go
*Server.Start()
```

> Start will start the execution of parsed jobs. If there are any unparsed jobs left
> in the queue it will wait for them to finish before starting execution.

### StopParsing

```go
*Server.StopParsing() error
```

> StopParsing will send a signal to stop all parsing being done on the Server.
> StopParsing can only be called when the Server is in a IsParsing -state.
> Returns error.

### StopRunning

```go
*Server.StopRunning() error
```

> StopRunning will send a signal to stop fetching requests on the Server.
> StopRunning can only be called when the Server is in a IsRunning -state.
> Returns error.

### WaitDone

```go
*Server.WaitDone()
```

> WaitDone will wait until the Server has finished fetching all the requests in the *Server.jobs map.
> WaitDone will block the program until it has finished.

### GetNumberOfVirtualUsers

```go
*Server.GetNumberOfVirtualUsers() int
```

> GetNumberOfVirtualUsers returns the number of virtual users.
> Returns int.

### GetNumberOfJobs

```go
*Server.GetNumberOfJobs() int
```

> GetNumberOfJobs returns the number of jobs stored on the Server.
> Returns int

### GetNumberOfRequests

```go
*Server.GetNumberOfRequests() int
```

> GetNumberOfRequests returns the number of successfull requests.
> Returns int.

### GetNumberOfErrors

```go
*Server.GetNumberOfErrors() int
```

> GetNumberOfErrors will return the amount of requests that errored.
> Returns int.

### GetErrorMessages

```go
*Server.GetErrorMessages() []error
```

> GetErrorMessages will return all the error messages since the Server was started.
> Returns Error.

### GetAverageFetchTime

```go
*Server.GetAverageFetchTime() time.Duration
```

> GetAverageFetchTime will return the average fetch time for all the requests. Requests that resultet in errors will be ignored in the average.
> Returns time.Duration.

### IsParsing

```go
*Server.IsParsing() bool
```

> IsParsing returns true if the Server is still parsing jobs. False if it has finished or manually been stopped.
> Returns bool.

### IsRunning

```go
*Server.IsRunning() bool
```

> IsRunning returns true if the Server is still running jobs. False if it has finished or manually been stopped.
> Returns bool.

### GetTotalRunTime

```go
*Server.GetTotalRunTime() time.Duration
```

> GetTotalRunTime will return the total runtime since Server start.
> Returns time.Duration.

## Installation

`go get -u github.com/dwtechnologies/steptest`

## Contributors

To improve on the project, please submit a pull request.

## License

The code is copyright under the MIT license.