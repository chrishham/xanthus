package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/chrishham/xanthus/internal/handlers"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func TestHandleRoot(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus int
		expectedHeader string
	}{
		{
			name:           "should redirect to login page",
			expectedStatus: http.StatusTemporaryRedirect,
			expectedHeader: "/login",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			authHandler := handlers.NewAuthHandler()

			router.GET("/", authHandler.HandleRoot)

			req, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedHeader, w.Header().Get("Location"))
		})
	}
}

func TestHandleLoginPage(t *testing.T) {
	t.Run("should call HTML method with correct parameters", func(t *testing.T) {
		router := setupTestRouter()
		authHandler := handlers.NewAuthHandler()

		// Set up a simple template to avoid nil pointer panic
		router.SetHTMLTemplate(template.Must(template.New("login.html").Parse("<html>Login Page</html>")))
		router.GET("/login", authHandler.HandleLoginPage)

		req, err := http.NewRequest("GET", "/login", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 200 with our simple template
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Login Page")
	})
}

func TestHandleLogin(t *testing.T) {
	tests := []struct {
		name           string
		formData       url.Values
		expectedStatus int
		expectedBody   string
		expectRedirect bool
	}{
		{
			name:           "empty token should return 400",
			formData:       url.Values{"cf_token": {""}},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "API token is required",
			expectRedirect: false,
		},
		{
			name:           "missing token field should return 400",
			formData:       url.Values{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "API token is required",
			expectRedirect: false,
		},
		{
			name:           "invalid token should return error message",
			formData:       url.Values{"cf_token": {"invalid_token"}},
			expectedStatus: http.StatusOK,
			expectedBody:   "❌ Invalid Cloudflare API token",
			expectRedirect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			authHandler := handlers.NewAuthHandler()

			router.POST("/login", authHandler.HandleLogin)

			formData := tt.formData.Encode()
			req, err := http.NewRequest("POST", "/login", strings.NewReader(formData))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			if tt.expectRedirect {
				assert.Equal(t, "/main", w.Header().Get("HX-Redirect"))
			}
		})
	}
}

func TestHandleLogin_ValidToken(t *testing.T) {
	// This test would require mocking the Cloudflare API calls
	// For now, we'll skip the implementation as it requires extensive mocking
	// of external dependencies like utils.VerifyCloudflareToken, utils.CheckKVNamespaceExists, etc.
	t.Skip("Requires mocking of external Cloudflare API dependencies")
}

func TestHandleLogout(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus int
		expectedHeader string
		checkCookie    bool
	}{
		{
			name:           "should clear cookie and redirect to login",
			expectedStatus: http.StatusTemporaryRedirect,
			expectedHeader: "/login",
			checkCookie:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			authHandler := handlers.NewAuthHandler()

			router.GET("/logout", authHandler.HandleLogout)

			req, err := http.NewRequest("GET", "/logout", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedHeader, w.Header().Get("Location"))

			if tt.checkCookie {
				cookies := w.Result().Cookies()
				found := false
				for _, cookie := range cookies {
					if cookie.Name == "cf_token" {
						found = true
						assert.Equal(t, "", cookie.Value)
						assert.Equal(t, -1, cookie.MaxAge)
						break
					}
				}
				assert.True(t, found, "cf_token cookie should be present and cleared")
			}
		})
	}
}

func TestHandleHealth(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:           "should return healthy status",
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]interface{}{"status": "healthy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			authHandler := handlers.NewAuthHandler()

			router.GET("/health", authHandler.HandleHealth)

			req, err := http.NewRequest("GET", "/health", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedBody, response)
		})
	}
}

// Benchmark tests for performance measurement
func BenchmarkHandleRoot(b *testing.B) {
	router := setupTestRouter()
	authHandler := handlers.NewAuthHandler()
	router.GET("/", authHandler.HandleRoot)

	req, _ := http.NewRequest("GET", "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkHandleHealth(b *testing.B) {
	router := setupTestRouter()
	authHandler := handlers.NewAuthHandler()
	router.GET("/health", authHandler.HandleHealth)

	req, _ := http.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
