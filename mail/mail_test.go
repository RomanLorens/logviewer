package mail

import (
	"context"
	"testing"
)

func TestSend(t *testing.T) {
	e := NewEmail("localhost:1025")

	err := e.Send(context.Background(), []string{"rl@citi.com"}, "test", "<h1>msg</h1>")
	if err != nil {
		t.Fatalf("error %v", err)
	}
}
