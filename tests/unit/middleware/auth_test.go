package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chrishham/xanthus/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAuthMiddleware_NoCookie(t *testing.T) {
	router := gin.New()
	router.Use(middleware.AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

func TestAuthMiddleware_EmptyCookie(t *testing.T) {
	router := gin.New()
	router.Use(middleware.AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "cf_token", Value: ""})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	router := gin.New()
	router.Use(middleware.AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "cf_token", Value: "invalid_token"})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	// Note: This test requires a valid Cloudflare token to pass
	// In a real scenario, you would mock the VerifyCloudflareToken function
	t.Skip("Requires valid Cloudflare token or mocking of VerifyCloudflareToken")

	router := gin.New()
	router.Use(middleware.AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		token := c.GetString("cf_token")
		c.JSON(http.StatusOK, gin.H{"message": "success", "token_set": token != ""})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "cf_token", Value: "valid_cloudflare_token"})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_TokenStoredInContext(t *testing.T) {
	// Note: This test requires mocking or a valid token
	t.Skip("Requires mocking of VerifyCloudflareToken")

	var contextToken string
	router := gin.New()
	router.Use(middleware.AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		contextToken = c.GetString("cf_token")
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	testToken := "test_token"
	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "cf_token", Value: testToken})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, testToken, contextToken)
}

func TestAPIAuthMiddleware_NoCookie(t *testing.T) {
	router := gin.New()
	router.Use(middleware.APIAuthMiddleware())
	router.GET("/api/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/api/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authentication required")
}

func TestAPIAuthMiddleware_EmptyCookie(t *testing.T) {
	router := gin.New()
	router.Use(middleware.APIAuthMiddleware())
	router.GET("/api/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.AddCookie(&http.Cookie{Name: "cf_token", Value: ""})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid authentication token")
}

func TestAPIAuthMiddleware_InvalidToken(t *testing.T) {
	router := gin.New()
	router.Use(middleware.APIAuthMiddleware())
	router.GET("/api/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.AddCookie(&http.Cookie{Name: "cf_token", Value: "invalid_token"})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid authentication token")
}

func TestAPIAuthMiddleware_ValidToken(t *testing.T) {
	// Note: This test requires a valid Cloudflare token to pass
	// In a real scenario, you would mock the VerifyCloudflareToken function
	t.Skip("Requires valid Cloudflare token or mocking of VerifyCloudflareToken")

	router := gin.New()
	router.Use(middleware.APIAuthMiddleware())
	router.GET("/api/protected", func(c *gin.Context) {
		token := c.GetString("cf_token")
		c.JSON(http.StatusOK, gin.H{"message": "success", "token_set": token != ""})
	})

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.AddCookie(&http.Cookie{Name: "cf_token", Value: "valid_cloudflare_token"})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIAuthMiddleware_TokenStoredInContext(t *testing.T) {
	// Note: This test requires mocking or a valid token
	t.Skip("Requires mocking of VerifyCloudflareToken")

	var contextToken string
	router := gin.New()
	router.Use(middleware.APIAuthMiddleware())
	router.GET("/api/protected", func(c *gin.Context) {
		contextToken = c.GetString("cf_token")
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	testToken := "test_token"
	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.AddCookie(&http.Cookie{Name: "cf_token", Value: testToken})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, testToken, contextToken)
}

// Benchmark tests
func BenchmarkAuthMiddleware_NoCookie(b *testing.B) {
	router := gin.New()
	router.Use(middleware.AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkAuthMiddleware_InvalidToken(b *testing.B) {
	router := gin.New()
	router.Use(middleware.AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "cf_token", Value: "invalid_token"})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkAPIAuthMiddleware_NoCookie(b *testing.B) {
	router := gin.New()
	router.Use(middleware.APIAuthMiddleware())
	router.GET("/api/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/protected", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkAPIAuthMiddleware_InvalidToken(b *testing.B) {
	router := gin.New()
	router.Use(middleware.APIAuthMiddleware())
	router.GET("/api/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/protected", nil)
		req.AddCookie(&http.Cookie{Name: "cf_token", Value: "invalid_token"})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}