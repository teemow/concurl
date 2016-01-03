package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
)

func makeGetEndpoint(svc ConcurlService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		body, err := svc.Get()
		if err != nil {
			return getResponse{body, err.Error()}, nil
		}
		return getResponse{body, ""}, nil
	}
}

func decodeGetRequest(r *http.Request) (interface{}, error) {
	var request getRequest
	return request, nil
}

func decodeGetResponse(r *http.Response) (interface{}, error) {
	var response getResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeResponse(w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}

func encodeRequest(r *http.Request, request interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
}

type getRequest struct {
}

type getResponse struct {
	Body string `json:"body"`
	Err  string `json:"err,omitempty"`
}
