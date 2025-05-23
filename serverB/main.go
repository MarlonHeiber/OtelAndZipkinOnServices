package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ViaCEP struct {
	Cep        string          `json:"cep"`
	Localidade string          `json:"localidade"`
	Erro       json.RawMessage `json:"erro"`
}

type WeatherApi struct {
	Location struct {
		Name string `json:"name"`
	} `json:"location"`
	Current struct {
		TempC float64 `json:"temp_c"`
		TempF float64 `json:"temp_f"`
	} `json:"current"`
}

type WeatherResponse struct {
	City  string  `json:"City"`
	TempC float64 `json:"Temp_C"`
	TempF float64 `json:"Temp_F"`
	TempK float64 `json:"Temp_K"`
}

func init() {
	viper.AutomaticEnv()
}

func initProvider(serviceName, collectorURL string) (func(context.Context) error, error) {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, collectorURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tracerProvider.Shutdown, nil
}

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	shutdown, err := initProvider(viper.GetString("OTEL_SERVICE_NAME"), viper.GetString("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Fatal("failed to shutdown TracerProvider: %w", err)
		}
	}()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		showTemperatureByCep(r.Context(), w, r)
	})
	handler := otelhttp.NewHandler(mux, "serverB")
	http.ListenAndServe(":8080", handler)

}
func showTemperatureByCep(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	cepParam := r.URL.Query().Get("cep")
	if cepParam == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println("Bad Request")
		return
	}

	viaCep, err := BuscaCEP(ctx, cepParam)
	if err != nil {
		switch err.Error() {
		case "invalid zipcode":
			w.WriteHeader(http.StatusUnprocessableEntity) // 422
			json.NewEncoder(w).Encode(map[string]string{
				"error": "invalid zipcode",
			})
		case "can not find zipcode":
			w.WriteHeader(http.StatusNotFound) // 404
			json.NewEncoder(w).Encode(map[string]string{
				"error": "can not find zipcode",
			})
		default:
			w.WriteHeader(http.StatusInternalServerError) // fallback
			json.NewEncoder(w).Encode(map[string]string{
				"error": "internal error",
			})
		}
		return
	}

	temperare, err := getWeatherFromCityName(ctx, viaCep.Localidade)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println("StatusInternalServerError")
		json.NewEncoder(w).Encode(map[string]string{
			"error": "internal error",
		})
		return
	}

	response := WeatherResponse{
		City:  temperare.Location.Name,
		TempC: temperare.Current.TempC,
		TempF: temperare.Current.TempF, //Não converti pois já existe na API
		TempK: temperare.Current.TempC + 273,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	// w.Write([]byte(fmt.Sprint(viaCep)))
}

func BuscaCEP(ctx context.Context, cep string) (ViaCEP, error) {
	ctx, span := otel.Tracer("serverB").Start(ctx, "BuscaCEP")
	defer span.End()
	req, err := http.Get("http://viacep.com.br/ws/" + cep + "/json/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao fazer requisição: %v \n", err)
		return ViaCEP{}, err
	}
	defer req.Body.Close()

	if req.StatusCode == http.StatusBadRequest {
		return ViaCEP{}, fmt.Errorf("invalid zipcode")
	}

	response, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao ler resposta: %v \n", err)
		return ViaCEP{}, err
	}
	var data ViaCEP
	err = json.Unmarshal(response, &data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao fazer parce da resposta: %v \n", err)
		return ViaCEP{}, err
	}

	if len(data.Erro) > 0 {
		return ViaCEP{}, fmt.Errorf("can not find zipcode")
	}
	return data, nil

}

func getWeatherFromCityName(ctx context.Context, cityName string) (WeatherApi, error) {
	ctx, span := otel.Tracer("serverB").Start(ctx, "getWeatherFromCityName")
	defer span.End()
	weatherUrl := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?q=%s&key=12b01999d1844295996195139252304", cityName)
	req, err := http.Get(weatherUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao fazer requisição: %v \n", err)
		return WeatherApi{}, err
	}
	defer req.Body.Close()

	response, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao ler resposta: %v \n", err)
		return WeatherApi{}, err
	}
	var data WeatherApi
	err = json.Unmarshal(response, &data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao fazer parse da resposta: %v \n", err)
		return WeatherApi{}, err
	}
	return data, nil

}
