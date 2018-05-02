// Package steptest makes transactional load test easy.
package steptest

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

// variablesFrom will set variables from either BODY or HEADERS as defined in step s
// from the response res and add them to the job j.
// Returns error.
func (j *job) variablesFrom(s *step, res *http.Response) error {
	if len(s.varfrom) == 0 {
		return nil
	}

	raw, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("Couldn't read Body of response in *job.variablesFrom. %s", err.Error())
	}
	headers := res.Header

	for _, v := range s.varfrom {
		switch strings.ToUpper(v.From) {
		case "BODY":
			err := j.variableFromBody(&v, &raw)
			if err != nil {
				return err
			}

		case "HEADER":
			j.variableFromHeader(&v, headers)
		}
	}

	return nil
}

// variableFromHeader will create or overwrite a variable in the jobs j vars map based on the
// value stored in the response res headers of the header with name from v.orgSyntax.
func (j *job) variableFromHeader(v *varfromItem, header http.Header) {
	value := header.Get(v.OrgSyntax)
	if value == "" {
		return
	}

	j.vars[v.Varname] = value
}

// variableFromBody will create or overwrite a variable in the jobs j vars map based on the
// search syntax supplied by v.syntax.
// Returns error.
func (j *job) variableFromBody(v *varfromItem, raw *[]byte) error {
	regexp, err := regexp.Compile(v.Syntax)
	if err != nil {
		return fmt.Errorf("Couldn't compile regular expression in *job.variableFromBody. %s", err.Error())
	}

	value := string(regexp.Find(*raw))
	if value == "" {
		return nil
	}

	parts := strings.Split(v.OrgSyntax, searchSyntax)
	for _, part := range parts {
		value = strings.Replace(value, part, "", -1)
	}

	j.vars[v.Varname] = value
	return nil
}
