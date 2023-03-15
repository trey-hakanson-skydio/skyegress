package util

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"google.golang.org/protobuf/proto"
)

type Method string

const (
	POST   Method = "POST"
	GET    Method = "GET"
	DELETE Method = "DELETE"
)

// simple client for making requests to protobuf services
type ProtoClient struct {
	url     string
	headers map[string]string
}

func NewProtoClient(url string) ProtoClient {
	return ProtoClient{
		url:     url,
		headers: make(map[string]string),
	}
}

func (pc *ProtoClient) AddHeader(key string, value string) *ProtoClient {
	pc.headers[key] = value
	return pc
}

func (pc *ProtoClient) RemoveHeader(key string) *ProtoClient {
	delete(pc.headers, key)
	return pc
}

// make an HTTP request to a protobuf service
func (pc *ProtoClient) Request(
	method Method,
	path string,
	protoReq proto.Message,
	protoRes proto.Message,
) error {
	// serialize request
	out, err := proto.Marshal(protoReq)
	if err != nil {
		return err
	}

	// make request
	endpoint := fmt.Sprintf("%s%s", pc.url, path)
	req, err := http.NewRequest(string(method), endpoint, bytes.NewReader(out))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/x-protobuf")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// deserialize response
	out, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return proto.Unmarshal(out, protoRes)
}
