package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chrishham/xanthus/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestJSONSuccess(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.JSONSuccess(c, "Operation successful", map[string]interface{}{
			"id":   123,
			"name": "test-item",
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response utils.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Equal(t, "Operation successful", response.Message)
	assert.NotNil(t, response.Data)

	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(123), data["id"]) // JSON unmarshals numbers as float64
	assert.Equal(t, "test-item", data["name"])
}

func TestJSONSuccessSimple(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.JSONSuccessSimple(c, "Simple success message")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Simple success message", response["message"])
}

func TestJSONError(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.JSONError(c, http.StatusBadRequest, "Invalid request parameters")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response utils.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "Invalid request parameters", response.Error)
	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestJSONBadRequest(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.JSONBadRequest(c, "Bad request message")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response utils.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "Bad request message", response.Error)
	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestJSONUnauthorized(t *testing.T) {
	testCases := []struct {
		name            string
		message         string
		expectedMessage string
	}{
		{
			name:            "With custom message",
			message:         "Custom unauthorized message",
			expectedMessage: "Custom unauthorized message",
		},
		{
			name:            "With empty message",
			message:         "",
			expectedMessage: "Unauthorized",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := setupTestRouter()
			router.GET("/test", func(c *gin.Context) {
				utils.JSONUnauthorized(c, tc.message)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)

			var response utils.ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response.Success)
			assert.Equal(t, tc.expectedMessage, response.Error)
			assert.Equal(t, http.StatusUnauthorized, response.Code)
		})
	}
}

func TestJSONForbidden(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.JSONForbidden(c, "Access forbidden")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response utils.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "Access forbidden", response.Error)
	assert.Equal(t, http.StatusForbidden, response.Code)
}

func TestJSONNotFound(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.JSONNotFound(c, "Resource not found")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response utils.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "Resource not found", response.Error)
	assert.Equal(t, http.StatusNotFound, response.Code)
}

func TestJSONInternalServerError(t *testing.T) {
	testCases := []struct {
		name            string
		message         string
		expectedMessage string
	}{
		{
			name:            "With custom message",
			message:         "Database connection failed",
			expectedMessage: "Database connection failed",
		},
		{
			name:            "With empty message",
			message:         "",
			expectedMessage: "Internal server error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := setupTestRouter()
			router.GET("/test", func(c *gin.Context) {
				utils.JSONInternalServerError(c, tc.message)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusInternalServerError, w.Code)

			var response utils.ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response.Success)
			assert.Equal(t, tc.expectedMessage, response.Error)
			assert.Equal(t, http.StatusInternalServerError, response.Code)
		})
	}
}

func TestJSONServiceUnavailable(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.JSONServiceUnavailable(c, "Service temporarily unavailable")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response utils.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "Service temporarily unavailable", response.Error)
	assert.Equal(t, http.StatusServiceUnavailable, response.Code)
}

func TestJSONResponse(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		customData := map[string]interface{}{
			"custom": "response",
			"code":   201,
		}
		utils.JSONResponse(c, http.StatusCreated, customData)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "response", response["custom"])
	assert.Equal(t, float64(201), response["code"])
}

func TestHTMLError(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.HTMLError(c, "HTML error message")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html", w.Header().Get("Content-Type"))
	assert.Equal(t, "❌ HTML error message", w.Body.String())
}

func TestHTMLSuccess(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.HTMLSuccess(c, "HTML success message")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html", w.Header().Get("Content-Type"))
	assert.Equal(t, "✅ HTML success message", w.Body.String())
}

func TestHTMXRedirect(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.HTMXRedirect(c, "/dashboard")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "/dashboard", w.Header().Get("HX-Redirect"))
}

func TestHTMXRefresh(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.HTMXRefresh(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "true", w.Header().Get("HX-Refresh"))
}

func TestJSONValidationError(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		errors := []utils.ValidationError{
			{Field: "email", Message: "Email is required"},
			{Field: "password", Message: "Password must be at least 8 characters"},
		}
		utils.JSONValidationError(c, errors)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	assert.Equal(t, "Validation failed", response["error"])
	
	errors, ok := response["errors"].([]interface{})
	require.True(t, ok)
	assert.Len(t, errors, 2)

	firstError := errors[0].(map[string]interface{})
	assert.Equal(t, "email", firstError["field"])
	assert.Equal(t, "Email is required", firstError["message"])

	secondError := errors[1].(map[string]interface{})
	assert.Equal(t, "password", secondError["field"])
	assert.Equal(t, "Password must be at least 8 characters", secondError["message"])
}

func TestVPSCreationSuccess(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.VPSCreationSuccess(c, "test-server")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Server created successfully with K3s, Helm, and ArgoCD", response["message"])
}

func TestVPSDeletionSuccess(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.VPSDeletionSuccess(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Server deleted successfully and configuration cleaned up", response["message"])
}

func TestVPSConfigurationSuccess(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.VPSConfigurationSuccess(c, "example.com")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, "VPS successfully configured with SSL certificates for example.com", response["message"])
}

func TestApplicationSuccess(t *testing.T) {
	testCases := []struct {
		action          string
		expectedMessage string
	}{
		{"create", "Application created successfully"},
		{"upgrade", "Application upgraded successfully"},
		{"delete", "Application deleted successfully"},
		{"unknown", "Operation completed successfully"},
	}

	for _, tc := range testCases {
		t.Run(tc.action, func(t *testing.T) {
			router := setupTestRouter()
			router.GET("/test", func(c *gin.Context) {
				utils.ApplicationSuccess(c, tc.action, "test-app")
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.True(t, response["success"].(bool))
			assert.Equal(t, tc.expectedMessage, response["message"])
			assert.Equal(t, "test-app", response["application_id"])
		})
	}
}

func TestDNSConfigurationSuccess(t *testing.T) {
	testCases := []struct {
		action          string
		expectedMessage string
	}{
		{"configure", "SSL certificate configured successfully"},
		{"remove", "SSL configuration removed successfully"},
		{"unknown", "DNS operation completed successfully"},
	}

	for _, tc := range testCases {
		t.Run(tc.action, func(t *testing.T) {
			router := setupTestRouter()
			router.GET("/test", func(c *gin.Context) {
				utils.DNSConfigurationSuccess(c, "example.com", tc.action)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response utils.SuccessResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.True(t, response.Success)
			assert.Equal(t, tc.expectedMessage, response.Message)
			
			data, ok := response.Data.(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "example.com", data["domain"])
		})
	}
}

func TestSetupSuccess(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.SetupSuccess(c, "Cloudflare")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Cloudflare configuration saved successfully", response["message"])
}

// Benchmarks
func BenchmarkJSONSuccess(b *testing.B) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.JSONSuccess(c, "Benchmark message", map[string]interface{}{"key": "value"})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}
}

func BenchmarkJSONError(b *testing.B) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		utils.JSONError(c, http.StatusBadRequest, "Benchmark error message")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}
}