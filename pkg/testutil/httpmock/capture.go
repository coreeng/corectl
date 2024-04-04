package httpmock

import (
	"encoding/json"
	. "github.com/onsi/gomega"
	"io"
	"net/http"
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
			Expect(nextResponseI).To(BeNumerically("<", len(responses)), "unexpected amount of requests to http mock: %s", nextResponseI)
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
			body, err := io.ReadAll(r.Body)
			Expect(err).NotTo(HaveOccurred(), "couldn't read request body from http mock")
			var req REQ
			Expect(json.Unmarshal(body, &req)).To(Succeed(), "couldn't unmarshal http mock body as json")
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
		Expect(err).NotTo(HaveOccurred(), "couldn't marshal response for http mock")
		_, err = w.Write(marshalledResponse)
		Expect(err).NotTo(HaveOccurred(), "couldn't write response for http mock")
	}
	h.handler = f
	return h
}
