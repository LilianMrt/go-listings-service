package testsupport

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/kafka"
)

// StartKafka launches a throwaway single-node Kafka container and returns its
// broker addresses. The container is terminated on cleanup.
func StartKafka(t testing.TB) []string {
	t.Helper()
	ctx := context.Background()

	container, err := kafka.Run(ctx, "confluentinc/confluent-local:7.6.1")
	if err != nil {
		t.Fatalf("start kafka container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("terminate kafka container: %v", err)
		}
	})

	brokers, err := container.Brokers(ctx)
	if err != nil {
		t.Fatalf("get kafka brokers: %v", err)
	}
	return brokers
}
