package bitcoind

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"log"
)

// A rpcClient represents a JSON RPC client (over HTTP(s)).
type rpcClient struct {
	serverAddr string
	user       string
	passwd     string
	httpClient *http.Client
}

// rpcRequest represent a RCP request
type rpcRequest struct {
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Id      int64       `json:"id"`
	JsonRpc string      `json:"jsonrpc"`
}

// rpcError represents a RCP error
/*type rpcError struct {
	Code    int16  `json:"code"`
	Message string `json:"message"`
}*/

type rpcResponse struct {
	Id     int64           `json:"id"`
	Result json.RawMessage `json:"result"`
	Err    interface{}     `json:"error"`
}

func newClient(host string, port int, user, passwd string, useSSL bool) (c *rpcClient, err error) {
	if len(host) == 0 {
		err = errors.New("Bad call missing argument host")
		return
	}
	var serverAddr string
	var httpClient *http.Client
	if useSSL {
		serverAddr = "https://"
		t := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient = &http.Client{Transport: t}
	} else {
		serverAddr = "http://"
		httpClient = &http.Client{}
	}
	c = &rpcClient{serverAddr: fmt.Sprintf("%s%s:%d", serverAddr, host, port), user: user, passwd: passwd, httpClient: httpClient}
	return
}

// doTimeoutRequest process a HTTP request with timeout
func (c *rpcClient) doTimeoutRequest(timeout time.Duration, req *http.Request) (*http.Response, error) {
	c.httpClient.Timeout = timeout
	return c.httpClient.Do(req)
}

// call prepare & exec the request
func (c *rpcClient) call(method string, params interface{}) (rr rpcResponse, err error) {
	connectTimeout := RPCCLIENT_TIMEOUT * time.Second
	rpcR := rpcRequest{method, params, time.Now().UnixNano(), "1.0"}
	payloadBuffer := &bytes.Buffer{}
	jsonEncoder := json.NewEncoder(payloadBuffer)
	err = jsonEncoder.Encode(rpcR)
	if err != nil {
		return
	}
	req, err := http.NewRequest("POST", c.serverAddr, payloadBuffer)
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", "application/json;charset=utf-8")
	req.Header.Add("Accept", "application/json")

	// Auth ?
	if len(c.user) > 0 || len(c.passwd) > 0 {
		req.SetBasicAuth(c.user, c.passwd)
	}

	resp, err := c.doTimeoutRequest(connectTimeout, req)

	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
    }()

	if err != nil {
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(data))
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = errors.New("HTTP error: " + resp.Status)
		return
	}
	err = json.Unmarshal(data, &rr)
	return
}
