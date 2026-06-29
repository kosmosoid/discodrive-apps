package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Pairing struct {
	DeviceCode      string
	UserCode        string
	VerificationURI string
	Interval        int
	ExpiresIn       int
}

func PairInit(ctx context.Context, serverURL, name, kind string) (Pairing, error) {
	body, _ := json.Marshal(map[string]string{"name": name, "kind": kind})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/pair/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := defaultHTTPClient().Do(req)
	if err != nil {
		return Pairing{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return Pairing{}, fmt.Errorf("/pair/init: %s", resp.Status)
	}
	var out struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		Interval        int    `json:"interval"`
		ExpiresIn       int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Pairing{}, err
	}
	return Pairing(out), nil
}

func PairPoll(ctx context.Context, serverURL, deviceCode string, interval time.Duration) (string, error) {
	for {
		body, _ := json.Marshal(map[string]string{"device_code": deviceCode})
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/pair/token", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := defaultHTTPClient().Do(req)
		if err != nil {
			return "", err
		}
		var out struct {
			Status      string `json:"status"`
			DeviceToken string `json:"device_token"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()
		switch out.Status {
		case "approved":
			return out.DeviceToken, nil
		case "pending":
		default:
			return "", fmt.Errorf("pairing not completed: %s", out.Status)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(interval):
		}
	}
}
