package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", validadeCepAndSendToServiceB)
	http.ListenAndServe(":8081", mux)

}
func validadeCepAndSendToServiceB(w http.ResponseWriter, r *http.Request) {
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

	urlServerB := fmt.Sprintf("http://localhost:8080/?cep=%s", cepParam)

	req, err := http.NewRequest(http.MethodPost, urlServerB, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Não foi possível criar a requisição para o servidor B",
		})
		return
	}

	resp, err := http.DefaultClient.Do(req)
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
