// Package vaultdummy is an internal vault mocking package
package vaultdummy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
)

// DummyVaultServer runs a test server to simulate vault and sets up the env accordingly
func DummyVaultServer(handler func(r *http.Request) interface{}) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := handler(r)
		response := map[string]interface{}{
			"data": data,
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			panic(err)
		}
		w.Write(responseBytes)
	}))
	os.Setenv("VAULT_TOKEN", "dummy-token")
	os.Setenv("VAULT_ADDR", ts.URL)
	return ts
}

// DummyVaultServerSecretEcho dummy vault server to echo sectets
func DummyVaultServerSecretEcho() *httptest.Server {
	return DummyVaultServer(func(r *http.Request) interface{} {
		response := map[string]string{
			"token":    "weel-a-token",
			"name":     "call my name",
			"user":     "user-from" + r.URL.Path,
			"password": "dummy-password",
		}
		return response
	})
}
