package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type ViaCEP struct {
	Cep         string          `json:"cep"`
	Logradouro  string          `json:"logradouro"`
	Complemento string          `json:"complemento"`
	Unidade     string          `json:"unidade"`
	Bairro      string          `json:"bairro"`
	Localidade  string          `json:"localidade"`
	Uf          string          `json:"uf"`
	Estado      string          `json:"estado"`
	Regiao      string          `json:"regiao"`
	Ibge        string          `json:"ibge"`
	Gia         string          `json:"gia"`
	Ddd         string          `json:"ddd"`
	Siafi       string          `json:"siafi"`
	Erro        json.RawMessage `json:"erro"`
}

type WeatherApi struct {
	Location struct {
		Name           string  `json:"name"`
		Region         string  `json:"region"`
		Country        string  `json:"country"`
		Lat            float64 `json:"lat"`
		Lon            float64 `json:"lon"`
		TzID           string  `json:"tz_id"`
		LocaltimeEpoch int     `json:"localtime_epoch"`
		Localtime      string  `json:"localtime"`
	} `json:"location"`
	Current struct {
		LastUpdatedEpoch int     `json:"last_updated_epoch"`
		LastUpdated      string  `json:"last_updated"`
		TempC            float64 `json:"temp_c"`
		TempF            float64 `json:"temp_f"`
		IsDay            int     `json:"is_day"`
		Condition        struct {
			Text string `json:"text"`
			Icon string `json:"icon"`
			Code int    `json:"code"`
		} `json:"condition"`
		WindMph    float64 `json:"wind_mph"`
		WindKph    float64 `json:"wind_kph"`
		WindDegree int     `json:"wind_degree"`
		WindDir    string  `json:"wind_dir"`
		PressureMb float64 `json:"pressure_mb"`
		PressureIn float64 `json:"pressure_in"`
		PrecipMm   float64 `json:"precip_mm"`
		PrecipIn   float64 `json:"precip_in"`
		Humidity   int     `json:"humidity"`
		Cloud      int     `json:"cloud"`
		FeelslikeC float64 `json:"feelslike_c"`
		FeelslikeF float64 `json:"feelslike_f"`
		WindchillC float64 `json:"windchill_c"`
		WindchillF float64 `json:"windchill_f"`
		HeatindexC float64 `json:"heatindex_c"`
		HeatindexF float64 `json:"heatindex_f"`
		DewpointC  float64 `json:"dewpoint_c"`
		DewpointF  float64 `json:"dewpoint_f"`
		VisKm      float64 `json:"vis_km"`
		VisMiles   float64 `json:"vis_miles"`
		Uv         float64 `json:"uv"`
		GustMph    float64 `json:"gust_mph"`
		GustKph    float64 `json:"gust_kph"`
	} `json:"current"`
}

type WeatherResponse struct {
	City  string  `json:"City"`
	TempC float64 `json:"Temp_C"`
	TempF float64 `json:"Temp_F"`
	TempK float64 `json:"Temp_K"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", showTemperatureByCep)
	http.ListenAndServe(":8080", mux)

}
func showTemperatureByCep(w http.ResponseWriter, r *http.Request) {
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

	viaCep, err := BuscaCEP(cepParam)
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

	temperare, err := getWeatherFromCityName(viaCep.Localidade)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println("StatusInternalServerError")
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

func BuscaCEP(cep string) (ViaCEP, error) {
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

func getWeatherFromCityName(cityName string) (WeatherApi, error) {
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
