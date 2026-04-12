package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"neo_collector_go/internal/domain"
)

type queryResponse struct {
	Status    string `json:"status"`
	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
	Data      struct {
		ResultType string         `json:"resultType"`
		Result     []vectorResult `json:"result"`
	} `json:"data"`
}

type vectorResult struct {
	Metric map[string]string `json:"metric"`
	Value  []json.RawMessage `json:"value"`
}

func (c *Client) Query(ctx context.Context, query string) ([]domain.Datapoint, error) {
	endpoint, err := url.Parse(c.baseURL + "/api/v1/query")
	if err != nil {
		return nil, fmt.Errorf("parse prometheus endpoint: %w", err)
	}

	params := endpoint.Query()
	params.Set("query", query)
	endpoint.RawQuery = params.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build prometheus request: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("execute prometheus request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("prometheus returned status %s", response.Status)
	}

	var payload queryResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode prometheus response: %w", err)
	}

	if payload.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: %s (%s)", payload.Error, payload.ErrorType)
	}

	if payload.Data.ResultType != "vector" {
		return nil, fmt.Errorf("unsupported prometheus result type %q", payload.Data.ResultType)
	}

	datapoints := make([]domain.Datapoint, 0, len(payload.Data.Result))
	for _, item := range payload.Data.Result {
		timestamp, value, err := parseSamplePair(item.Value)
		if err != nil {
			return nil, err
		}

		labels := make(map[string]string, len(item.Metric))
		for key, rawValue := range item.Metric {
			labels[key] = rawValue
		}

		datapoints = append(datapoints, domain.Datapoint{
			Labels:    labels,
			Value:     value,
			Timestamp: timestamp,
		})
	}

	return datapoints, nil
}

func parseSamplePair(raw []json.RawMessage) (time.Time, float64, error) {
	if len(raw) != 2 {
		return time.Time{}, 0, fmt.Errorf("invalid prometheus sample pair")
	}

	timestampValue, err := parseJSONFloat(raw[0])
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("parse prometheus timestamp: %w", err)
	}

	value, err := parseJSONFloat(raw[1])
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("parse prometheus value: %w", err)
	}

	seconds, fraction := math.Modf(timestampValue)
	timestamp := time.Unix(int64(seconds), int64(fraction*float64(time.Second))).UTC()

	return timestamp, value, nil
}

func parseJSONFloat(raw json.RawMessage) (float64, error) {
	var numeric float64
	if err := json.Unmarshal(raw, &numeric); err == nil {
		return numeric, nil
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err != nil {
		return 0, err
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(asString), 64)
	if err != nil {
		return 0, err
	}

	return value, nil
}
