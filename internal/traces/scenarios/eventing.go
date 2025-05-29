package scenarios

import (
	"context"
	"fmt"
	"time"

	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// EventingScenario simulates a producer-consumer event-driven trace scenario for testing and demonstration purposes.
func EventingScenario(ctx context.Context, tracer trace.Tracer, _ *zap.Logger, serviceName string, _ []string) error {
	// Use different service names for producer and consumer
	producerServiceName := serviceName + "-event-producer"
	consumerServiceName := serviceName + "-event-consumer"

	r := NewRand()

	messageID := fmt.Sprintf("msg-%d", r.Int64())
	conversationID := fmt.Sprintf("conv-%d", r.Int64())

	// Producer
	ctx, producerSpan := tracer.Start(ctx, "event_producer",
		trace.WithAttributes(
			semconv.ServiceNameKey.String(producerServiceName),
			semconv.MessagingSystemKey.String("kafka"),
			semconv.MessagingOperationTypePublish,
			semconv.MessagingDestinationName("user_events"),
			semconv.MessagingMessageIDKey.String(messageID),
			semconv.MessagingMessageConversationIDKey.String(conversationID),
			semconv.MessagingKafkaMessageKeyKey.String(fmt.Sprintf("key-%d", r.Int64())),
			semconv.MessagingMessageBodySizeKey.Int(r.IntN(1000)+100),
		),
	)

	// Simulate producing a message
	time.Sleep(time.Duration(r.IntN(50)) * time.Millisecond)
	producerSpan.End()

	// Simulate some time passing
	time.Sleep(time.Duration(r.IntN(200)) * time.Millisecond)

	// Consumer
	consumerCtx, consumerSpan := tracer.Start(context.Background(), "event_consumer",
		trace.WithAttributes(
			semconv.ServiceNameKey.String(consumerServiceName),
			semconv.MessagingSystemKey.String("kafka"),
			semconv.MessagingOperationTypeReceive,
			semconv.MessagingDestinationName("user_events"),
			semconv.MessagingMessageIDKey.String(messageID),
			semconv.MessagingMessageConversationIDKey.String(conversationID),
			semconv.MessagingEventhubsConsumerGroup("user-events-group"),
			semconv.MessagingKafkaMessageOffsetKey.Int(r.IntN(1000)),
		),
	)

	// Add link to the producer span
	consumerSpan.AddLink(trace.LinkFromContext(ctx))

	// Simulate consuming a message
	time.Sleep(time.Duration(r.IntN(100)) * time.Millisecond)
	consumerSpan.End()

	// Process event
	_, processSpan := tracer.Start(consumerCtx, "process_event",
		trace.WithAttributes(
			semconv.FaaSTriggerPubsub,
			semconv.FaaSInvokedName(fmt.Sprintf("execution-%d", r.Int64())),
			semconv.FaaSDocumentOperationInsert,
		),
	)
	time.Sleep(time.Duration(r.IntN(150)) * time.Millisecond)
	processSpan.End()

	return nil
}
