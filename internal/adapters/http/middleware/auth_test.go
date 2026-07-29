package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuth_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Auth("secret"))
	r.GET("/v1/users/me", func(c *gin.Context) {
		t.Fatal("should not reach inner handler when token is missing")
	})

	req := httptest.NewRequest("GET", "/v1/users/me", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}

	var errResp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp["code"] != "UNAUTHENTICATED" {
		t.Errorf("expected status UNAUTHENTICATED, got %v", errResp["code"])
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Auth("secret"))
	r.GET("/v1/users/me", func(c *gin.Context) {
		t.Fatal("should not reach inner handler when token is invalid")
	})

	req := httptest.NewRequest("GET", "/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.string")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestAuth_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSecret := "secret"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "user-999",
		"email": "test@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, _ := token.SignedString([]byte(jwtSecret))

	var extractedID string
	r := gin.New()
	r.Use(Auth(jwtSecret))
	r.GET("/v1/users/me", func(c *gin.Context) {
		extractedID = c.GetString("userID")
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if extractedID != "user-999" {
		t.Errorf("expected c.GetString(\"userID\") to return user-999, got %s", extractedID)
	}
}
