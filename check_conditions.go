// Package steptest makes transactional load test easy.
package steptest

// checkConditions will check the steps if/conditions and return true if any of the conditions matched.
// Returns boolean.
func (j *job) checkConditions(s *step) bool {
	if len(s.conditions) == 0 {
		return true
	}

	for _, c := range s.conditions {
		switch c.Type {
		case "exists":
			if _, ok := j.vars[c.Var1]; ok {
				return true
			}
		}
	}

	return false
}
