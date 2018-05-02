// Package steptest makes transactional load test easy.
package steptest

// workerResults will listen for []*results to be sent on the *Server.resultJobs channel.
// It will append any data from the channel to *Server.results.
func (srv *Server) workerResults() {
	for {
		results := <-srv.resultJobs

		srv.results = append(srv.results, results...)
		srv.wgRes.Done()
	}
}

// workerResultsCounter will listen for incoming increment changes on the *Server.resultCounterChan channel
// and add those to the *Server.resultsCounter. This way we can track the amount of finished requests.
func (srv *Server) workerResultsCounter() {
	for {
		inc := <-srv.resultCounterChan
		srv.resultsCounter += inc
	}
}
