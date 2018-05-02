// Package steptest makes transactional load test easy.
//FIXME: Whole jobs_parse.go should be replaced by something fancier and more javascript-syntax like.
//And support for nested for loops, scoped variables and such. Next version :) :) :)
package steptest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

const (
	stepSeparatorRegexp = "(?m:^- )"     // Each step is divided by a leading dash and a following space.
	lineSeparatorRegexp = "(?m:^  )"     // Each function in a step is divided by two leading spaces.
	removeVarCurls      = "(?m:^{{|}}$)" // Regexp to remove curls from brackets.
	separator           = " "            // Each command, value etc. is separated by a space.
	emptyRow            = ""             // Empty rows will be blank, since we trim whitespaces.
	trim                = " \t"          // Trim whitespaces and tabs.
	newline             = "\n"           // Character to match newlines.
	forInSeparator      = "in"           // Separator between variable name and array.
)

var (
	// Allowed condition types for the if/condition statement.
	allowedConditions = []string{"exists", "equals", "greater", "less", "true", "false"}
)

// stepTypes contains all the supported functions of the stepsfile.
// If any row begins with anything else than described below it will result in an error.
// Variables and cookies are always global. So keep this in mind that there are no scopes.
// Auth and headers can be either local to the step or global. Declare global auth headers
// by adding @ in front of cookie/header.
// Rows only containing one newline will be ignored.
var stepTypes = map[string]func(*job, *step, *string) error{
	"get":     createGet,
	"post":    createPost,
	"patch":   createPatch,
	"put":     createPut,
	"delete":  createDelete,
	"var":     createVar,
	"array":   createArray,
	"varfrom": createVarFrom,
	"cookie":  createCookie,
	"header":  createHeader,
	"auth":    createAuth,
	"@header": createGlobalHeader,
	"@auth":   createGlobalAuth,
	"for":     startForLoop,
	"forend":  endForLoop,
	"if":      createIf,
}

// parseJob takes raw job r and creates a job out of it.
// It then parses r.steps and turns it into a parsed job.
// Returns *job and error.
func (srv *Server) parseJob(r *rawJob) (*job, error) {
	j := &job{arrays: make(map[string][]string), vars: r.vars}

	if j.vars == nil {
		j.vars = make(map[string]string)
	}

	err := j.createSteps(r)
	if err != nil {
		return nil, err
	}

	return j, nil
}

// createSteps reads the stepsfile s and splits it into steps based on the stepSeparator.
// It will iterate over each step and call *job.createStep for each step.
// Returns error.
func (j *job) createSteps(r *rawJob) error {
	regexp, err := regexp.Compile(stepSeparatorRegexp)
	if err != nil {
		return fmt.Errorf("Couldn't compile regular expression in *job.createSteps. %s", err.Error())
	}

	steps := regexp.Split(r.steps, -1)
	for _, s := range steps {
		// Skip empty steps only containing spaces / tabs or are empty.
		if strings.Trim(s, trim) == "" {
			continue
		}

		err := j.createStep(&s)
		if err != nil {
			return err
		}
	}

	// If forcounter is greater or smaller than 0, we had a FOR
	// loop without a FOREND or FOREND with without a FOR.
	switch {
	case j.forcounter > 0:
		return fmt.Errorf("Received for statement without a forend in steps in *job.createSteps")
	case j.forcounter < 0:
		return fmt.Errorf("Received forend statement without a for in steps in *job.createSteps")
	}

	return nil
}

// createStep takes step s and splits it into lines based on the lineSeparator.
// It will iterate over each line and call *job.createStepLine for each line.
// Returns error.
func (j *job) createStep(s *string) error {
	stp := new(step)
	regexp, err := regexp.Compile(lineSeparatorRegexp)
	if err != nil {
		return fmt.Errorf("Couldn't compile regular expression in *job.createStep. %s", err.Error())
	}

	// Remove all newlines and split by lineSeparatorRegexp.
	r := regexp.Split(*s, -1)
	for _, row := range r {
		row = strings.Replace(row, newline, "", -1)
		err := j.createStepLine(stp, &row)
		if err != nil {
			return err
		}
	}

	// Determine if we should add the step to the job or to a for loop.
	// If the addTo slice is not empty, the tep should be added to a for loop step.
	// Otherwise we will hit the default case, which is just to add it as a regular
	// step directly on the jobs steps slice.
	switch {
	case len(j.addTo) > 0:
		j.addStepToForLoop(stp)

	default:
		j.addStepToJob(stp)
	}

	// If we should leave the for loop for next step, remove the value form j.addTo.
	// Since we don't support nested for loops at this time we just remove the slice
	// otherwise we would pop the last value out.
	if j.forRemoveNextStep {
		j.forRemoveNextStep = false
		j.addTo = []int{}
	}

	return nil
}

// addStepToJob will add a step to the global steps slice of the job. So all regular steps
// will be added by this function.
func (j *job) addStepToJob(stp *step) {
	j.steps = append(j.steps, *stp)
}

// addStepToForLoop will add the step to the for loop steps slice. All steps belonging to a for loop
// will be added here. Note nested for loops are not supported at this time!
func (j *job) addStepToForLoop(stp *step) {
	if len(j.steps) == j.addTo[0] {
		j.steps = append(j.steps, step{forloop: stp.forloop})
	}

	stp.forloop = forloop{}
	j.steps[j.addTo[0]].forloop.steps = append(j.steps[j.addTo[0]].forloop.steps, *stp)
}

// createStepLine will call the function based on what keyword is defined in the step.
// See stepTypes for the different types/keywords. Any empty rows will be ignored.
// We will trim all leading and empty spaces so that empty rows with a singel space will not cause an error.
// We will assign the function to f based on the stepTypes map.
// Returns error.
func (j *job) createStepLine(step *step, r *string) error {
	s := strings.SplitN(strings.Trim(*r, trim), separator, 2)
	t := strings.ToLower(s[0])

	if t == emptyRow {
		return nil
	}

	f, ok := stepTypes[t]
	if !ok {
		return fmt.Errorf("Couldn't find function type %s in stepTypes map in *job.createStepLine", t)
	}

	args := ""
	if len(s) > 1 {
		args = s[1]
	}

	err := f(j, step, &args)
	return err
}

// createGet will create a HTTP GET step based on step s and args a by
// calling *step.createHTTPStep. job j is not needed and will be ignored.
// Returns error.
func createGet(j *job, s *step, a *string) error {
	return s.createHTTPStep("GET", a)
}

// createGet will create a HTTP POST step based on step s and args a by
// calling *step.createHTTPStep. job j is not needed and will be ignored.
// Returns error.
func createPost(j *job, s *step, a *string) error {
	return s.createHTTPStep("POST", a)
}

// createGet will create a HTTP PATCH step based on step s and args a by
// calling *step.createHTTPStep. job j is not needed and will be ignored.
// Returns error.
func createPatch(j *job, s *step, a *string) error {
	return s.createHTTPStep("PATCH", a)
}

// createGet will create a HTTP PUT step based on step s and args a by
// calling *step.createHTTPStep. job j is not needed and will be ignored.
// Returns error.
func createPut(j *job, s *step, a *string) error {
	return s.createHTTPStep("PUT", a)
}

// createGet will create a HTTP DELETE step based on step s and args a by
// calling *step.createHTTPStep. job j is not needed and will be ignored.
// Returns error.
func createDelete(j *job, s *step, a *string) error {
	return s.createHTTPStep("DELETE", a)
}

// createHTTPStep will take m method and a args and set the correct method, url
// and body to the step. If method is GET or no body argument is supplied no body will be set.
// Returns error.
func (s *step) createHTTPStep(m string, a *string) error {
	v := strings.SplitN(*a, separator, 2)

	switch {
	case v[0] == "":
		return fmt.Errorf("%s was declared but URL was not supplied in *step.createHTTPStep. Raw %s", m, *a)

	case len(v) > 1 && m != "GET":
		s.body = v[1]
	}

	s.method = m
	s.url = v[0]
	return nil
}

// createVar will add a variable from args a to the jobs j vars map. Step s will be ignored.
// These variables will be global and accessible to the whole job after they have been declared.
// Returns error.
func createVar(j *job, s *step, a *string) error {
	v := new(variable)
	err := json.Unmarshal([]byte(*a), v)
	if err != nil {
		return fmt.Errorf("var was declared but we couldn't unmarshal it in createVar. Raw %s", *a)
	}

	switch {
	case v.Name == "":
		return fmt.Errorf("var was declared but NAME was not supplied in createVar. Raw %s", *a)

	case v.Value == "":
		return fmt.Errorf("var was declared but VALUE was not supplied in createVar. Raw %s", *a)
	}

	j.vars[v.Name] = v.Value
	return nil
}

// createVarFrom will add a variable to the jobs j vars map depending on the result from the steps HTTP request.
// The value can be fetched by specifying either BODY or HEADER and then specifying a pattern to look for in args a.
// Substitute the value to get from the search syntax with searchSyntax. Step s will be ignored.
// Returns error.
func createVarFrom(j *job, s *step, a *string) error {
	v := new(varfromItem)
	err := json.Unmarshal([]byte(*a), v)
	if err != nil {
		return fmt.Errorf("varfrom was declared but we couldn't unmarshal it in createVarFrom. Raw %s", *a)
	}

	switch {
	case v.From == "":
		return fmt.Errorf("varfrom was declared but FROM was not supplied in createVarFrom. Raw %s", *a)

	case v.Varname == "":
		return fmt.Errorf("varfrom was declared but NAME was not supplied in createVarFrom. Raw %s", *a)

	case v.OrgSyntax == "":
		return fmt.Errorf("varfrom was declared but FIND was not supplied in createVarFrom. Raw %s", *a)
	}

	v.Syntax = *j.createSearchPattern(&v.OrgSyntax)
	s.varfrom = append(s.varfrom, *v)
	return nil
}

// createSearchPattern will take an input pattern p and convert it to a regular expression we can use to parse the BODY/HEADERS of a result.
// It does this by replace the searchSyntax word with searchSyntaxReplace and then inserting it inside searchSyntaxRegexp.
func (*job) createSearchPattern(p *string) *string {
	n := fmt.Sprintf(searchSyntaxRegexp, strings.Replace(*p, searchSyntax, searchSyntaxReplace, -1))
	return &n
}

// createCookie will append a cookie to the jobs j cookies slice based on the data in args a. Step s will be ignored.
// Returns error.
func createCookie(j *job, s *step, a *string) error {
	c := new(cookie)
	err := json.Unmarshal([]byte(*a), c)
	if err != nil {
		return fmt.Errorf("cookie was declared but we couldn't unmarshal it in createCookie. Raw %s", *a)
	}

	switch {
	case c.Name == "":
		return fmt.Errorf("cookie was declared but NAME was not supplied in createCookie. Raw %s", *a)

	case c.Value == "":
		return fmt.Errorf("cookie was declared but VALUE was not supplied in createCookie. Raw %s", *a)
	}

	j.cookies = append(j.cookies, http.Cookie{
		Name:     c.Name,
		Value:    c.Value,
		Path:     c.Path,
		Domain:   c.Domain,
		Expires:  c.Expires,
		MaxAge:   c.MaxAge,
		Secure:   c.Secure,
		HttpOnly: c.HTTPOnly,
	})
	return nil
}

// createHeader will append a header to the steps headers slice based on the data in args a.
// Headers added with createHeader will be local to the specified step s only. Jobs j will be ignored.
// Returns error.
func createHeader(j *job, s *step, a *string) error {
	h := new(header)
	err := json.Unmarshal([]byte(*a), h)
	if err != nil {
		return fmt.Errorf("header was declared but we couldn't unmarshal it in createHeader. Raw %s", *a)
	}

	switch {
	case h.Name == "":
		return fmt.Errorf("header was declared but NAME was not supplied in createHeader. Raw %s", *a)

	case h.Value == "":
		return fmt.Errorf("header was declared but VALUE was not supplied in createHeader. Raw %s", *a)
	}

	s.headers = append(s.headers, *h)
	return nil
}

// createAuth will add Basic Auth to the step s based on data in args a.
// Basic Auth added with createAuth will be local to the specified step s only. Jobs j will be ignored.
// Returns error.
func createAuth(j *job, s *step, a *string) error {
	ba := new(auth)
	err := json.Unmarshal([]byte(*a), ba)
	if err != nil {
		return fmt.Errorf("auth was declared but we couldn't unmarshal it in createAuth. Raw %s", *a)
	}

	switch {
	case ba.Username == "":
		return fmt.Errorf("auth was declared but USERNAME was not supplied in createAuth. Raw %s", *a)

	case ba.Password == "":
		return fmt.Errorf("auth was declared but PASSWORD was not supplied in createAuth. Raw %s", *a)
	}

	s.auth.Username, s.auth.Password = ba.Username, ba.Password
	return nil
}

// createGlobalHeader will append a header to the jobs j headers slice based on the data in args a.
// Headers added with createGlobalHeader will be global to all steps in job j. Step s will be ignored.
// Returns error.
func createGlobalHeader(j *job, s *step, a *string) error {
	h := new(header)
	err := json.Unmarshal([]byte(*a), h)
	if err != nil {
		return fmt.Errorf("@header was declared but we couldn't unmarshal it in createGlobalHeader. Raw %s", *a)
	}

	switch {
	case h.Name == "":
		return fmt.Errorf("@header was declared but NAME was not supplied in createGlobalHeader. Raw %s", *a)

	case h.Value == "":
		return fmt.Errorf("@header was declared but VALUE was not supplied in createGlobalHeader. Raw %s", *a)
	}

	j.globalHeaders = append(j.globalHeaders, *h)
	return nil
}

// createGlobalAuth will add Basic Auth to the job j based on data in args a.
// Basic Auth added with createGlobalAuth will be global to all steps in job j. Step s will be ignored.
// Returns error.
func createGlobalAuth(j *job, s *step, a *string) error {
	ba := new(auth)
	err := json.Unmarshal([]byte(*a), ba)
	if err != nil {
		return fmt.Errorf("@auth was declared but we couldn't unmarshal it in createGlobalAuth. Raw %s", *a)
	}

	switch {
	case ba.Username == "":
		return fmt.Errorf("@auth was declared but USERNAME was not supplied in createGlobalAuth. Raw %s", *a)

	case ba.Password == "":
		return fmt.Errorf("@auth was declared but PASSWORD was not supplied in createGlobalAuth. Raw %s", *a)
	}

	j.globalAuth.Username, j.globalAuth.Password = ba.Username, ba.Password
	return nil
}

// startForLoop will create a for that will loop all the steps contained within based on the supplied separator.
// It will run until len of the forloop struct becomes zero. And loop between the step for was declared and forend.
// Returns error.
func startForLoop(j *job, s *step, a *string) error {
	f := strings.SplitN(*a, separator, 3)
	switch {
	case len(j.addTo) > 0:
		return fmt.Errorf("Sorry, but nested FOR loops are not yet supported in startForLoop. Raw %s", *a)

	case len(f) < 3:
		return fmt.Errorf("for was declared but with an invalid syntax. FOR needs to be in 'for VARNAME in ARRAY' format in createFor. Raw %s", *a)

	case strings.ToLower(f[1]) != forInSeparator:
		return fmt.Errorf("for was declared but with an invalid syntax. FOR needs to be in 'for VARNAME in ARRAY' format in createFor. Raw %s", *a)
	}

	arr := new([]string)

	// If we couldn't unmarshal the data but the stored data looks to be an variable/array.
	// Search for the name of the variable in the jobs array list. If it exists we can safely
	// set the values to that array. If it doesn't exists we return error.
	regexp, err := regexp.Compile(removeVarCurls)
	if err != nil {
		return fmt.Errorf("Couldn't compile regular expression in *job.startForLoop. %s", err.Error())
	}

	_, ok := j.arrays[regexp.ReplaceAllString(f[2], "")]

	switch {
	case ok:
		*arr = []string{f[2]}

	default:
		err := json.Unmarshal([]byte(f[2]), arr)
		if err != nil {
			return fmt.Errorf("for was declared but we couldn't unmarshal values in it in createFor. Raw %s", *a)
		}
	}

	s.forloop = forloop{varname: f[0], values: *arr}

	j.addTo = []int{len(j.steps)} // Add the current steps index to the addTo slice. Hardcorded for now...
	j.forcounter++
	return nil
}

// endForLoop will end a previously created for loop. If no previous for loop was declared it will return error.
// Returns error.
func endForLoop(j *job, s *step, a *string) error {
	if j.forcounter < 1 {
		return fmt.Errorf("forend was encountered but no for was declared previously")
	}

	j.forRemoveNextStep = true
	j.forcounter--
	return nil
}

// createArray creates an array that can be used by other functions such as rand.
// Returns error.
func createArray(j *job, s *step, a *string) error {
	ar := new(array)
	err := json.Unmarshal([]byte(*a), ar)
	if err != nil {
		return fmt.Errorf("array was declared but we couldn't unmarshal it in createArray. Raw %s", *a)
	}

	switch {
	case ar.Name == "":
		return fmt.Errorf("array was declared but NAME was not supplied in createArray. Raw %s", *a)

	case len(ar.Values) == 0:
		return fmt.Errorf("array was declared but VALUES was not supplied in createArray. Raw %s", *a)
	}

	j.arrays[ar.Name] = ar.Values
	return nil
}

// createIf will create a conditional variable based on the supplied condition.
// It will also check that the supplied condition type is valid.
// Returns error.
func createIf(j *job, s *step, a *string) error {
	i := new(condition)
	err := json.Unmarshal([]byte(*a), i)
	if err != nil {
		return fmt.Errorf("if was declared but we couldn't unmarshal it in createIf. Raw %s", *a)
	}

	switch {
	case i.Type == "":
		return fmt.Errorf("if was declared but TYPE was not supplied in createIf. Raw %s", *a)

	case i.Var1 == "":
		return fmt.Errorf("if was declared but VAR1 was not supplied in createIf. Raw %s", *a)

	case i.Var2 == "" && i.Type != "exists":
		return fmt.Errorf("if was declared but VAR2 was not supplied in createIf. Raw %s", *a)
	}

	for _, c := range allowedConditions {
		if i.Type == c {
			s.conditions = append(s.conditions, *i)
			return nil
		}
	}

	return fmt.Errorf("if was declared but the supplied TYPE is not supported. Supported types are %s in createId. Raw %s", allowedConditions, *a)
}
