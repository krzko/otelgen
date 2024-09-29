package scenarios

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func EventingScenario(ctx context.Context, tracer trace.Tracer, logger *zap.Logger, serviceName string) error {
	// Use different service names for producer and consumer
	producerServiceName := fmt.Sprintf("%s-event-producer", serviceName)
	consumerServiceName := fmt.Sprintf("%s-event-consumer", serviceName)

	messageID := fmt.Sprintf("msg-%d", rand.Int63())
	conversationID := fmt.Sprintf("conv-%d", rand.Int63())

	// Producer
	ctx, producerSpan := tracer.Start(ctx, "event_producer",
		trace.WithAttributes(
			semconv.ServiceNameKey.String(producerServiceName),
			semconv.MessagingSystemKey.String("kafka"),
			semconv.MessagingOperationTypePublish,
			semconv.MessagingDestinationName("user_events"),
			semconv.MessagingMessageIDKey.String(messageID),
			semconv.MessagingMessageConversationIDKey.String(conversationID),
			semconv.MessagingKafkaMessageKeyKey.String(fmt.Sprintf("key-%d", rand.Int63())),
			semconv.MessagingMessageBodySizeKey.Int(rand.Intn(1000)+100),
		),
	)

	// Simulate producing a message
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
	producerSpan.End()

	// Simulate some time passing
	time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)

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
			semconv.MessagingKafkaMessageOffsetKey.Int(rand.Intn(1000)),
		),
	)

	// Add link to the producer span
	consumerSpan.AddLink(trace.LinkFromContext(ctx))

	// Simulate consuming a message
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	consumerSpan.End()

	// Process event
	_, processSpan := tracer.Start(consumerCtx, "process_event",
		trace.WithAttributes(
			semconv.FaaSTriggerPubsub,
			semconv.FaaSInvokedName(fmt.Sprintf("execution-%d", rand.Int63())),
			semconv.FaaSDocumentOperationInsert,
		),
	)
	time.Sleep(time.Duration(rand.Intn(150)) * time.Millisecond)
	processSpan.End()

	return nil
}
