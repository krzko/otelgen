package logs

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

// Run executes the test scenario
func Run(c *Config) error {
	if c.NumLogs <= 0 {
		return fmt.Errorf("number of logs must be greater than zero")
	}

	for i := 0; i < c.NumLogs; i++ {
		record := log.Record{}

		record.SetBody(log.StringValue(fmt.Sprintf("Test log %d", i+1)))

		now := time.Now()
		record.SetObservedTimestamp(now)
		record.SetTimestamp(now)

		record.SetSeverity(log.SeverityInfo)

		global.Logger(c.ServiceName).Emit(context.Background(), record)
	}

	return nil
}
