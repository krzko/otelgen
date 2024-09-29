package scenarios

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func MicroservicesScenario(ctx context.Context, tracer trace.Tracer, logger *zap.Logger, serviceName string) error {
	services := []string{
		"api_gateway", "auth_service", "user_service", "product_service", "inventory_service",
		"order_service", "payment_service", "shipping_service", "notification_service",
		"recommendation_service", "search_service", "analytics_service", "logging_service",
		"cache_service", "config_service", "monitoring_service",
	}

	ctx, rootSpan := tracer.Start(ctx, "complex_request",
		trace.WithAttributes(
			semconv.HTTPRequestMethodPost,
			semconv.HTTPRouteKey.String("/api/v1/order"),
			semconv.URLScheme("https"),
			semconv.URLFull("https://api.example.com/api/v1/order"),
			semconv.URLPath("/api/v1/order"),
			semconv.ClientAddress("203.0.113.195"),
			semconv.ClientPort(56789),
			semconv.UserAgentOriginal("ExampleApp/1.0"),
			semconv.HTTPRequestBodySize(2048),
			semconv.ServiceNameKey.String(fmt.Sprintf("%s_api_gateway", serviceName)),
		),
	)
	defer rootSpan.End()

	for i := 0; i < 100; i++ {
		microserviceName := services[rand.Intn(len(services))]
		specificServiceName := fmt.Sprintf("%s_%s", serviceName, microserviceName)

		_, span := tracer.Start(ctx, fmt.Sprintf("%s_operation", microserviceName),
			trace.WithAttributes(
				semconv.ServiceNameKey.String(specificServiceName),
				semconv.ServiceVersionKey.String(fmt.Sprintf("1.%d.0", rand.Intn(10))),
				semconv.ServiceInstanceIDKey.String(fmt.Sprintf("%s-instance-%d", microserviceName, rand.Intn(5))),
				semconv.ProcessRuntimeNameKey.String("OpenJDK Runtime Environment"),
				semconv.ProcessRuntimeVersionKey.String("11.0.9+11-Ubuntu-0ubuntu1.20.04"),
			),
		)

		// Add some events
		span.AddEvent("operation_started")

		// Simulate some work
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

		// Add some random attributes based on the service
		switch microserviceName {
		case "api_gateway":
			span.SetAttributes(
				semconv.HTTPRouteKey.String("/api/v1/order"),
				semconv.HTTPResponseStatusCode(200),
			)
		case "auth_service":
			span.SetAttributes(
				semconv.EnduserIDKey.String(fmt.Sprintf("user-%d", rand.Intn(1000))),
				semconv.EnduserRoleKey.String("customer"),
			)
		case "database_service":
			span.SetAttributes(
				semconv.DBSystemKey.String("postgresql"),
				semconv.DBNamespace("orders"),
				semconv.DBQueryText("INSERT INTO orders (user_id, product_id, quantity) VALUES ($1, $2, $3)"),
				semconv.DBOperationNameKey.String("INSERT"),
			)
		case "cache_service":
			span.SetAttributes(
				semconv.DBSystemKey.String("redis"),
				semconv.DBOperationNameKey.String("SET"),
			)
		case "payment_service":
			span.SetAttributes(
				semconv.RPCSystemKey.String("grpc"),
				semconv.RPCServiceKey.String("PaymentService"),
				semconv.RPCMethodKey.String("ProcessPayment"),
			)
		}

		if rand.Float32() < 0.1 { // 10% chance of an error
			span.SetStatus(codes.Error, "Operation failed")
			span.RecordError(fmt.Errorf("random error in %s", microserviceName))
		} else {
			span.SetStatus(codes.Ok, "Operation successful")
		}

		span.AddEvent("operation_ended")
		span.End()
	}

	rootSpan.SetStatus(codes.Ok, "")
	return nil
}
