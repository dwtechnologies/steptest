// Package steptest makes transactional load test easy.
package steptest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	replaceVarSyntax    = "{{%s}}"             // Syntax to search for when replacing variables.
	searchSyntax        = "{{StepTestSyntax}}" // searchSyntax used by VARFROM to look for patterns in BODY/HEADER.
	searchSyntaxReplace = ").+("               // searchSyntaxReplace is what we replace searchSyntax with in our regular expression.
	searchSyntaxRegexp  = "(%s)"               // searchSyntaxRegexp is what we encapsulate the whole search string to make a regular expression.
)

// replaceFromVariables will run replacement functions on data based on the variables stored in job j.
// It will replace the variables found in either URL, Body, Headers or Cookies with those stored in the jobs variables.
func (j *job) replaceFromVariables(s *step) {
	// Replace from variables.
	s.headers = append(j.globalHeaders, s.headers...)
	s.cookies = append([]http.Cookie{}, j.cookies...)

	for n, v := range j.vars {
		j.varReplaceURL(s, &n, &v)
		j.varReplaceBody(s, &n, &v)
		j.varReplaceHeaders(s, &n, &v)
		j.varReplaceCookies(s, &n, &v)
	}
}

// varReplaceURL will replace every occurrence of name n with value v in the URL.
func (*job) varReplaceURL(s *step, n *string, v *string) {
	s.url = strings.Replace(s.url, fmt.Sprintf(replaceVarSyntax, *n), *v, -1)
}

// varReplaceBody will replace every occurrence of name n with value v in the Body.
func (*job) varReplaceBody(s *step, n *string, v *string) {
	s.body = strings.Replace(s.body, fmt.Sprintf(replaceVarSyntax, *n), *v, -1)
}

// varReplaceHeaders will append the jobs j global Headers with the steps s local Headers and
// replace every occurrence of name n with value v in the headers.
// When done it will overwrite the the steps header value with the combined and replaced headers.
func (j *job) varReplaceHeaders(s *step, n *string, v *string) {
	for i := range s.headers {
		s.headers[i].Value = strings.Replace(s.headers[i].Value, fmt.Sprintf(replaceVarSyntax, *n), *v, -1)
	}
}

// varReplaceCookies will copy the jobs cookies and replace every occurrence of name n with value v in cookies.
// When done it will overwrite the the steps cookies value with the replaced cookies.
func (j *job) varReplaceCookies(s *step, n *string, v *string) {
	for i := range s.cookies {
		s.cookies[i].Value = strings.Replace(s.cookies[i].Value, fmt.Sprintf(replaceVarSyntax, *n), *v, -1)
	}
}

// replaceFromVariablesForLoop will run replacement functions on FOR variables on the arrays and variables stored in job j.
// It will first try to match any array with the name specified and replace the for loops values with that array.
// After that it will run variable replacement on the array. So it's possible to store variables in the array.
// It will replace the variables found in either URL, Body, Headers or Cookies with those stored in the jobs variables.
func (j *job) replaceFromVariablesForLoop(s *step) {
	for n, v := range j.arrays {
		j.replaceForLoopArray(s, &n, v)
	}

	for n, v := range j.vars {
		j.replaceForLoopStrings(s, &n, &v)
	}
}

// replaceForLoopStrings will replace every occurrence of name n with value v in the For Loops values.
// If v exists and can be unmarshal from a JSON array we will append them to values.
// It it's a regular string we will replace it.
func (*job) replaceForLoopStrings(s *step, n *string, v *string) {
	for i, storedValue := range s.forloop.values {
		if strings.Contains(storedValue, fmt.Sprintf(replaceVarSyntax, *n)) {
			arr := new([]string)
			err := json.Unmarshal([]byte(*v), arr)

			if err != nil {
				s.forloop.values[i] = *v
				continue
			}

			// Stuff the new slice in the same place that the old variable was.
			*arr = append(s.forloop.values[:i], *arr...)
			s.forloop.values = append(*arr, s.forloop.values[i+1:]...)
		}
	}
}

// replaceForLoopArrays will replace a array name with the associated array.
func (*job) replaceForLoopArray(s *step, n *string, v []string) {
	for _, storedValue := range s.forloop.values {
		if strings.Contains(storedValue, fmt.Sprintf(replaceVarSyntax, *n)) {
			s.forloop.values = v
		}
	}
}
