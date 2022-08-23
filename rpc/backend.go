package rpc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	bal "github.com/smallnest/weighted"
)

func (s *Server) SetBackends(urls []string) {
	s.balancer = &bal.SW{}
	for _, url := range urls {
		s.balancer.Add(url, 1) // set weight = 1 for all rpc backends
	}
}
func (h *handler) getResponseFromOriginRpcServer(msg *jsonrpcMessage) (*jsonrpcMessage, error) {
	if h.balancer == nil {
		h.log.Info("originRpcUrl is not activate")
		return nil, nil
	}

	bz, err := json.Marshal(msg)
	if err != nil {
		h.log.Error("failed to marshal messege to origin Rpc server, err :", err)
		return nil, err
	}
	h.log.Info(fmt.Sprintf("post (%s) to originRpcUrl", string(bz)))

	url, ok := h.balancer.Next().(string)
	if !ok {
		err := fmt.Errorf("failed to get url as string from balancer")
		h.log.Error(err.Error())
		return nil, err
	}

	resp, err := http.Post(url, "application/json", strings.NewReader(string(bz)))
	if err != nil {
		h.log.Error(fmt.Sprintf("failed to get response from origin Rpc server, err: %s", err.Error()))
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		h.log.Error(fmt.Sprintf("failed to get response from origin Rpc server, errCode: %d", resp.StatusCode))
		return nil, fmt.Errorf("redirect node %s returned http status code:%d", url, resp.StatusCode)
	}
	respBz, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.log.Error(fmt.Sprintf("failed to get response from origin Rpc server, err: %s", err.Error()))
		return nil, err
	}
	respMsg := jsonrpcMessage{}
	err = json.Unmarshal(respBz, &respMsg)
	if err != nil {
		h.log.Error(fmt.Sprintf("failed to unmarshal resp msg from origin Rpc server, err: %s", err.Error()))
		return nil, err
	}
	h.log.Info(fmt.Sprintf("result from originRpcUrl is %s", string(respMsg.Result)))
	return &respMsg, nil
}
