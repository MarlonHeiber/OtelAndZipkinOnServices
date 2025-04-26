package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"
)

func main() {
	shutdown := initTracer("serverA")
	defer shutdown()
	mux := http.NewServeMux()
	mux.HandleFunc("/", validateCepAndSendToServiceB)

	handler := otelhttp.NewHandler(mux, "serverA")
	http.ListenAndServe(":8081", handler)
	// http.ListenAndServe(":8081", mux)

}
func validateCepAndSendToServiceB(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	cepParam := r.URL.Query().Get("cep")
	if cepParam == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println("CEP não informado")
		json.NewEncoder(w).Encode(map[string]string{
			"error": "cep not informed",
		})
		return
	}
	if len(cepParam) != 8 || !isOnlyDigits(cepParam) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid zipcode",
		})
		return
	}

	urlServerB := fmt.Sprintf("http://serverB:8080/?cep=%s", cepParam)

	// req, err := http.NewRequest(http.MethodPost, urlServerB, nil)
	req, err := http.NewRequest(http.MethodGet, urlServerB, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Não foi possível criar a requisição para o servidor B",
		})
		return
	}

	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	resp, err := client.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Não foi possível enviar a requisição para o servidor B",
		})
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, resp.Body)

}

func isOnlyDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func initTracer(serviceName string) func() {
	ctx := context.Background()

	conn, err := grpc.DialContext(ctx, os.Getenv("OTEL_COLLECTOR_ENDPOINT"), grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}

	exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		log.Fatal(err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			attribute.String("service.name", serviceName),
		)),
	)

	otel.SetTracerProvider(tp)

	return func() {
		_ = tp.Shutdown(ctx)
	}
}
