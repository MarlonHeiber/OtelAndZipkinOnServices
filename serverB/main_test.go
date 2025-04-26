package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Teste de integração de sucesso para o handler HTTP principal
func TestShowTemperatureByCep_Success(t *testing.T) {
	req := httptest.NewRequest("GET", "/?cep=89239899", nil)
	w := httptest.NewRecorder()

	showTemperatureByCep(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var body WeatherResponse
	err := json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		t.Fatalf("error decoding JSON: %v", err)
	}

	if body.City == "" || body.TempC == 0 || body.TempF == 0 || body.TempK == 0 {
		t.Errorf("unexpected empty fields in response: %+v", body)
	}
}

// Teste de integração de falha em caso de cep inválido
func TestShowTemperatureByCep_InvalidCep(t *testing.T) {
	req := httptest.NewRequest("GET", "/?cep=8923989", nil)
	w := httptest.NewRecorder()

	showTemperatureByCep(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", resp.StatusCode)
	}
}

// Teste de integração de falha em caso de cep não encontrado
func TestShowTemperatureByCep_NotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/?cep=89239898", nil)
	w := httptest.NewRecorder()

	showTemperatureByCep(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}
