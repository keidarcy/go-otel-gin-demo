package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestHelloEndpoint(t *testing.T) {
	// Setup minimal tracer for testing
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	// Get the router from our actual application code
	router := setupRouter()

	// Create a test request to /hello
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/hello", nil)

	// Perform the request
	router.ServeHTTP(w, req)

	// Check response status code
	assert.Equal(t, http.StatusOK, w.Code)

	// Check response body contains expected text
	assert.Contains(t, w.Body.String(), "Hello, traced Gin")
}

func TestNonExistentEndpoint(t *testing.T) {
	// Setup minimal tracer for testing
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(nil)

	// Get the router from our actual application code
	router := setupRouter()

	// Create a test request to a non-existent endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/not-found", nil)

	// Perform the request
	router.ServeHTTP(w, req)

	// Check response status code - should be 404
	assert.Equal(t, http.StatusNotFound, w.Code)
}
