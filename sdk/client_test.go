package sdk_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"policylens/internal/server"
	"policylens/sdk"
)

func TestSDKRoundTrip(t *testing.T) {
	service, err := server.New(server.Config{EnginePath: "/missing/kyverno"})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(service.Handler())
	defer srv.Close()
	client := sdk.New(srv.URL)
	items, err := client.Policies(context.Background())
	if err != nil || len(items) != 6 {
		t.Fatalf("policies: %d %v", len(items), err)
	}
	answer, err := client.Ask(context.Background(), "Which registries are allowed?", "")
	if err != nil || len(answer.Citations) == 0 || answer.Mode != "excerpts" {
		t.Fatalf("ask: %+v %v", answer, err)
	}
	_, err = client.Check(context.Background(), "invalid")
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 400 || apiErr.RequestID == "" {
		t.Fatalf("structured API error missing: %v", err)
	}
}
