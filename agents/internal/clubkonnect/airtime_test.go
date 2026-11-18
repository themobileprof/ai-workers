package clubkonnect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestAccepted(t *testing.T) {
	if !Accepted("ORDER_RECEIVED") || !Accepted("order_completed") || Accepted("INSUFFICIENT_BALANCE") {
		t.Fatal("accepted")
	}
}

func TestBuy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("UserID") != "CK1" || r.URL.Query().Get("APIKey") != "secret" {
			t.Fatalf("creds")
		}
		if r.URL.Query().Get("MobileNetwork") != "01" || r.URL.Query().Get("MobileNumber") != "08031234567" {
			t.Fatalf("%s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"orderid":"9","statuscode":"100","status":"ORDER_RECEIVED"}`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{UserID: "CK1", APIKey: "secret", BaseURL: srv.URL, HTTP: srv.Client()}
	got, err := c.Buy(context.Background(), "01", "08031234567", "500", "req-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.OrderID != "9" || got.Status != "ORDER_RECEIVED" {
		t.Fatalf("%+v", got)
	}
}

func TestBuyRejectsBalance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"statuscode":"417","status":"INSUFFICIENT_BALANCE"}`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{UserID: "CK1", APIKey: "secret", BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.Buy(context.Background(), "01", "08031234567", "500", "req-1")
	if err == nil || err.Error() == "airtime secret" {
		t.Fatalf("%v", err)
	}
}

func TestRewardAmountDefault(t *testing.T) {
	t.Setenv("REWARD_AMOUNT", "")
	if RewardAmount() != 500 {
		t.Fatal(RewardAmount())
	}
	t.Setenv("REWARD_AMOUNT", "200")
	if RewardAmount() != 200 {
		t.Fatal(RewardAmount())
	}
	t.Setenv("REWARD_AMOUNT", "10")
	if RewardAmount() != 500 {
		t.Fatal("below ClubKonnect minimum stays 500")
	}
}

func TestLiveBuy(t *testing.T) {
	if os.Getenv("AIRTIME_LIVE") != "1" {
		t.Skip("set AIRTIME_LIVE=1 to send a real ₦500 test")
	}
	cli := FromEnv()
	if cli == nil {
		t.Fatal("CLUBKONNECT_USERID / CLUBKONNECT_APIKEY missing")
	}
	phone := os.Getenv("AIRTIME_TEST_PHONE")
	if phone == "" {
		phone = "07075240400"
	}
	network := os.Getenv("AIRTIME_TEST_NETWORK")
	if network == "" {
		network = "01"
	}
	reqID := "tmptest" + time.Now().Format("20060102")
	got, err := cli.Buy(context.Background(), network, phone, "500", reqID)
	if err != nil {
		t.Fatal(err)
	}
	if got.OrderID == "" && got.Status == "" {
		t.Fatalf("%+v", got)
	}
	t.Logf("airtime status=%s order=%s", got.Status, got.OrderID)
}
