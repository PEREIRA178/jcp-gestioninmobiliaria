package services

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type ufResponse struct {
	Serie []struct {
		Valor float64 `json:"valor"`
	} `json:"serie"`
}

var (
	ufMu      sync.RWMutex
	ufValue   float64
	ufFetched time.Time
)

// GetUF returns the current UF value in CLP, cached for 1 hour.
func GetUF() float64 {
	ufMu.RLock()
	if time.Since(ufFetched) < 1*time.Hour && ufValue > 0 {
		v := ufValue
		ufMu.RUnlock()
		return v
	}
	ufMu.RUnlock()

	ufMu.Lock()
	defer ufMu.Unlock()
	if time.Since(ufFetched) < 1*time.Hour && ufValue > 0 {
		return ufValue
	}
	if v := fetchUF(); v > 0 {
		ufValue = v
		ufFetched = time.Now()
	}
	return ufValue
}

func fetchUF() float64 {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://mindicador.cl/api/uf")
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	var data ufResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || len(data.Serie) == 0 {
		return 0
	}
	return data.Serie[0].Valor
}
