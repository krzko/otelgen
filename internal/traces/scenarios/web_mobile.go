package scenarios

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// WebMobileScenario simulates a web/mobile client-server trace scenario for testing and demonstration purposes.
func WebMobileScenario(ctx context.Context, tracer trace.Tracer, _ *zap.Logger, serviceName string, _ []string) error {
	clientTypes := []string{"web_browser", "ios_app", "android_app"}
	r := NewRand()
	clientType := clientTypes[r.IntN(len(clientTypes))]

	clientServiceName := serviceName + "-web-mobile"
	webServerServiceName := serviceName + "-web-server"
	appServerServiceName := serviceName + "-app-server"
	dbServerServiceName := serviceName + "-web-server"

	var userAgent, deviceModel, osName, osVersion string
	switch clientType {
	case "web_browser":
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
		deviceModel = "PC"
		osName = "Windows"
		osVersion = "10"
	case "ios_app":
		userAgent = "MyApp/1.0 (iPhone; iOS 14.7.1; Scale/3.00)"
		deviceModel = "iPhone12,1"
		osName = "iOS"
		osVersion = "14.7.1"
	case "android_app":
		userAgent = "MyApp/1.0 (Linux; Android 11; Pixel 4 Build/RQ3A.210805.001.A1)"
		deviceModel = "Pixel 4"
		osName = "Android"
		osVersion = "11"
	}

	// Start the root span
	ctx, rootSpan := tracer.Start(ctx, "client_request",
		trace.WithAttributes(
			semconv.ServiceNameKey.String(clientServiceName),
			semconv.UserAgentOriginal(userAgent),
			semconv.UserAgentName(fmt.Sprintf("MyApp (%s)", clientType)),
			semconv.UserAgentVersion("1.0"),
			semconv.DeviceModelIdentifier(deviceModel),
			semconv.OSName(osName),
			semconv.OSVersion(osVersion),
			semconv.HTTPRequestMethodGet,
			semconv.HTTPRouteKey.String("/api/data"),
			semconv.URLScheme("https"),
			semconv.URLFull("https://api.example.com/api/data?user=123"),
			semconv.URLPath("/api/data"),
			semconv.URLQuery("user=123"),
			semconv.ClientAddress("192.0.2.4"),
			semconv.ClientPort(51234),
		),
	)
	defer rootSpan.End()

	// Web Server
	ctx, webSpan := tracer.Start(ctx, "web_server",
		trace.WithAttributes(
			semconv.ServiceNameKey.String(webServerServiceName),
			semconv.ServerAddress("api.example.com"),
			semconv.ServerPort(443),
			semconv.HTTPResponseStatusCode(200),
			semconv.NetworkProtocolName("HTTP"),
			semconv.NetworkProtocolVersion("1.1"),
		),
	)
	webSpan.AddEvent("request_received", trace.WithAttributes(
		semconv.EventName("http.request.received"),
		semconv.HTTPRequestBodySize(1024),
	))
	time.Sleep(time.Duration(r.IntN(50)) * time.Millisecond)
	webSpan.End()

	// Application output
	ctx, appSpan := tracer.Start(ctx, "app_output",
		trace.WithAttributes(
			semconv.ServiceNameKey.String(appServerServiceName),
			semconv.ServiceNameKey.String("data-service"),
			semconv.ServiceVersionKey.String("1.5.0"),
			semconv.ServiceInstanceIDKey.String("data-service-1"),
		),
	)
	appSpan.AddEvent("processing_started")
	time.Sleep(time.Duration(r.IntN(100)) * time.Millisecond)
	appSpan.AddEvent("processing_completed")
	appSpan.End()

	// Database Backend
	_, dbSpan := tracer.Start(ctx, "database_query",
		trace.WithAttributes(
			semconv.ServiceNameKey.String(dbServerServiceName),
			semconv.DBSystemKey.String("postgresql"),
			semconv.DBNamespace("users"),
			semconv.DBQueryText("SELECT * FROM users WHERE id = $1"),
			semconv.DBOperationName("SELECT"),
			semconv.DBSystemPostgreSQL,
		),
	)
	time.Sleep(time.Duration(r.IntN(75)) * time.Millisecond)
	dbSpan.End()

	rootSpan.SetStatus(codes.Ok, "")
	return nil
}
