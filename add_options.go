// Package steptest makes transactional load test easy.
package steptest

import (
	"net/http"
)

// addOptions adds Headers, Cookies and Basic Auth from step s to request req.
func (j *job) addOptions(s *step, req *http.Request) {
	j.setUserAgent(req)
	j.addBasicAuth(s, req)
	j.addHeaders(s, req)
	j.addCookies(s, req)
}

// setUserAgent will set the User Agent of the request o Chrome Chrome/64.0.3282.186 on Mac OS X 10.13.3
func (*job) setUserAgent(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.186 Safari/537.36")
}

// addBasicAuth adds either local or global Basic Auth from step s to *http.Request req.
// If both local and global auth parameters are set the local will take precedence.
func (j *job) addBasicAuth(s *step, req *http.Request) {
	switch {
	case s.auth.Username != "" && s.auth.Password != "":
		req.SetBasicAuth(s.auth.Username, s.auth.Password)

	case j.globalAuth.Username != "" && j.globalAuth.Password != "":
		req.SetBasicAuth(j.globalAuth.Username, j.globalAuth.Password)
	}
}

// addHeaders sets headers from step s to *http.Request req.
// If header is already set it will be overwritten with the new value.
func (*job) addHeaders(s *step, req *http.Request) {
	for _, header := range s.headers {
		req.Header.Set(header.Name, header.Value)
	}
}

// addCookies adds cookies from step s to *http.Request req.
// If cookie is already set it will be overwritten with the new value.
func (*job) addCookies(s *step, req *http.Request) {
	for _, cookie := range s.cookies {
		req.AddCookie(&cookie)
	}
}

// appendResponseCookiesToJob will append/replace the cookies that we received from the response res to job j.
// So that cookies received will automatically be added to the next step of the job.
func (j *job) appendResponseCookiesToJob(res *http.Response) {
	for _, cookie := range res.Cookies() {
		exists := false

		// If we already have the cookie no need to add it again. Just update the value.
		for i := range j.cookies {
			if exists = j.compareCookie(cookie, &j.cookies[i]); exists {
				j.cookies[i].Value = cookie.Value
			}
		}

		if !exists {
			j.cookies = append(j.cookies, *cookie)
		}
	}
}

// compareCookie will check if name, domain, path are the same for *http.Cookie c1 and c2.
// If they are the same the returned value will be true.
// Returns bool.
func (*job) compareCookie(c1 *http.Cookie, c2 *http.Cookie) bool {
	if c1.Name == c2.Name && c1.Domain == c2.Domain && c1.Path == c2.Path {
		return true
	}
	return false
}
