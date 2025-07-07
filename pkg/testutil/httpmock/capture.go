package httpmock

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/onsi/gomega"
)

type HttpCaptureHandler[REQ any] struct {
	Requests []REQ
	handler  http.HandlerFunc
}

func (h *HttpCaptureHandler[REQ]) Func() http.HandlerFunc {
	return h.handler
}

func NewCaptureHandler[REQ any](
	responses ...any,
) *HttpCaptureHandler[REQ] {
	nextResponseI := 0
	return NewCaptureHandlerWithResponseFn(
		func(_ *REQ) any {
			gomega.Expect(nextResponseI).To(gomega.BeNumerically("<", len(responses)), "unexpected amount of requests to http mock: %s", nextResponseI)
			defer func() {
				nextResponseI += 1
			}()
			return responses[nextResponseI]
		},
	)
}

func NewCaptureHandlerWithResponseFn[REQ any](
	provideResponseFn func(*REQ) any,
) *HttpCaptureHandler[REQ] {
	return NewCaptureHandlerWithMappingFns(
		func(r *http.Request) REQ {
			var req REQ
			if r.Body != nil && r.ContentLength > 0 {
				body, err := io.ReadAll(r.Body)
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), "couldn't read request body from http mock")
				gomega.Expect(json.Unmarshal(body, &req)).To(gomega.Succeed(), "couldn't unmarshal http mock request body as json")
			} else {
				req = *new(REQ)
			}
			return req
		},
		provideResponseFn,
	)
}

func NewCaptureHandlerWithMappingFns[REQ any](
	provideRequestFn func(*http.Request) REQ,
	provideResponseFn func(*REQ) any,
) *HttpCaptureHandler[REQ] {
	h := &HttpCaptureHandler[REQ]{}
	f := func(w http.ResponseWriter, r *http.Request) {
		req := provideRequestFn(r)
		h.Requests = append(h.Requests, req)

		response := provideResponseFn(&req)
		marshalledResponse, err := json.Marshal(response)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "couldn't marshal response for http mock")
		_, err = w.Write(marshalledResponse)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "couldn't write response for http mock")
	}
	h.handler = f
	return h
}
