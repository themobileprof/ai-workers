// Package clubkonnect sends Nigerian airtime via the ClubKonnect (Nellobyte) GET API.
package clubkonnect

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const buyURL = "https://www.nellobytesystems.com/APIAirtimeV1.asp"

// Client talks to ClubKonnect. Secrets stay in env — never log the key or the full request URL.
type Client struct {
	UserID  string
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func FromEnv() *Client {
	uid := strings.TrimSpace(os.Getenv("CLUBKONNECT_USERID"))
	key := strings.TrimSpace(os.Getenv("CLUBKONNECT_APIKEY"))
	if uid == "" || key == "" {
		return nil
	}
	return &Client{UserID: uid, APIKey: key, HTTP: &http.Client{Timeout: 25 * time.Second}}
}

func RewardAmount() int {
	n, err := strconv.Atoi(strings.TrimSpace(os.Getenv("REWARD_AMOUNT")))
	if err != nil || n < 50 {
		return 500
	}
	if n > 200000 {
		return 200000
	}
	return n
}

type Result struct {
	OrderID    string
	StatusCode string
	Status     string
}

func (c *Client) Buy(ctx context.Context, networkCode, phone, amount, requestID string) (Result, error) {
	if c == nil || c.UserID == "" || c.APIKey == "" {
		return Result{}, fmt.Errorf("airtime is not configured")
	}
	q := url.Values{}
	q.Set("UserID", c.UserID)
	q.Set("APIKey", c.APIKey)
	q.Set("MobileNetwork", strings.TrimSpace(networkCode))
	q.Set("Amount", strings.TrimSpace(amount))
	q.Set("MobileNumber", strings.TrimSpace(phone))
	q.Set("RequestID", strings.TrimSpace(requestID))
	base := strings.TrimSpace(c.BaseURL)
	if base == "" {
		base = buyURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"?"+q.Encode(), nil)
	if err != nil {
		return Result{}, err
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return Result{}, err
	}
	if resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("airtime HTTP %d", resp.StatusCode)
	}
	var body struct {
		OrderID    string `json:"orderid"`
		StatusCode string `json:"statuscode"`
		Status     string `json:"status"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return Result{}, fmt.Errorf("airtime: unexpected reply")
	}
	out := Result{OrderID: body.OrderID, StatusCode: body.StatusCode, Status: strings.ToUpper(strings.TrimSpace(body.Status))}
	if !Accepted(out.Status) {
		return out, fmt.Errorf("airtime %s", firstNonEmpty(out.Status, out.StatusCode))
	}
	return out, nil
}

func Accepted(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "ORDER_RECEIVED", "ORDER_COMPLETED", "ORDER_PROCESSING":
		return true
	default:
		return false
	}
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
