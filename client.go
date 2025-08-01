package mijnhost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/libdns/libdns"
	"go.uber.org/zap"
)

func (p *Provider) updateRecord(ctx context.Context, zone string, record libdns.Record) (SavedRecordResponse, error) {
	body, err := json.Marshal(libdnsToRecordRequest(record))
	reqURL := fmt.Sprintf("%s/domains/%s/dns", ApiUrl, zone)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, reqURL, bytes.NewReader(body))

	var result SavedRecordResponse
	err = p.doAPIRequest(req, &result)

	return result, err
}

func (p *Provider) replaceRecords(ctx context.Context, zone string, records []libdns.Record) error {
	body, err := json.Marshal(libdnsToRecordList(records))
	reqURL := fmt.Sprintf("%s/domains/%s/dns", ApiUrl, zone)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, bytes.NewReader(body))

	err = p.doAPIRequest(req, nil)

	return err
}

func (p *Provider) doAPIRequest(req *http.Request, result interface{}) error {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("API-Key", p.ApiKey)

	logger := caddy.Log()
	var reqBody []byte
	if req.Body != nil {
		// Read and restore the request body for logging
		reqBody, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
	}
	logger.Info("API request",
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
		zap.Object("headers", caddyhttp.LoggableHTTPHeader{Header: req.Header}),
		zap.ByteString("body", reqBody),
	)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)

	if response.StatusCode >= 400 {
		return fmt.Errorf("got error status: HTTP %d: %+v", response.StatusCode, string(body))
	}

	if response.StatusCode == http.StatusNoContent {
		return err
	}

	err = json.Unmarshal(body, &result)

	return err
}
