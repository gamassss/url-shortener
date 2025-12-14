package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gamassss/url-shortener/internal/domain"
	"github.com/gamassss/url-shortener/tests/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestShortenURL_Success(t *testing.T) {
	mockService := new(mocks.MockShortenerService)
	handler := NewShortenerHandler(mockService)
	router := setupTestRouter()
	router.POST("/api/shorten", handler.ShortenURL)

	reqBody := `{"url": "https://example.com"}`
	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mockURL := &domain.URL{
		ID:          1,
		ShortCode:   "abc1234",
		OriginalURL: "https://example.com",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockService.On("ShortenURL", mock.Anything, mock.MatchedBy(func(req *domain.CreatedURLRequest) bool {
		return req.OriginalURL == "https://example.com"
	})).Return(mockURL, nil).Once()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/abc1234", response["short_url"])
	assert.Equal(t, "abc1234", response["short_code"])
	assert.Equal(t, "https://example.com", response["original_url"])

	mockService.AssertExpectations(t)
}

func TestShortenURL_InvalidJSON(t *testing.T) {
	mockService := new(mocks.MockShortenerService)
	handler := NewShortenerHandler(mockService)
	router := setupTestRouter()
	router.POST("/api/shorten", handler.ShortenURL)

	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(`{invalid json}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockService.AssertNotCalled(t, "ShortenURL")
}

func TestShortenURL_MissingURL(t *testing.T) {
	mockService := new(mocks.MockShortenerService)
	handler := NewShortenerHandler(mockService)
	router := setupTestRouter()
	router.POST("/api/shorten", handler.ShortenURL)

	reqBody := `{"custom_alias": "mylink"}`
	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "required")

	mockService.AssertNotCalled(t, "ShortenURL")
}

func TestShortenURL_InvalidURLFormat(t *testing.T) {
	mockService := new(mocks.MockShortenerService)
	handler := NewShortenerHandler(mockService)
	router := setupTestRouter()
	router.POST("/api/shorten", handler.ShortenURL)

	reqBody := `{"url": "not-a-valid-url"}`
	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "url")

	mockService.AssertNotCalled(t, "ShortenURL")
}

func TestShortenURL_ServiceError(t *testing.T) {
	mockService := new(mocks.MockShortenerService)
	handler := NewShortenerHandler(mockService)
	router := setupTestRouter()
	router.POST("/api/shorten", handler.ShortenURL)

	reqBody := `{"url": "https://example.com"}`
	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mockService.On("ShortenURL", mock.Anything, mock.Anything).
		Return(nil, errors.New("database error")).Once()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "database error")

	mockService.AssertExpectations(t)
}

func TestShortenURL_WithCustomAlias(t *testing.T) {
	mockService := new(mocks.MockShortenerService)
	handler := NewShortenerHandler(mockService)
	router := setupTestRouter()
	router.POST("/api/shorten", handler.ShortenURL)

	reqBody := `{"url": "https://example.com", "custom_alias": "mylink"}`
	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mockURL := &domain.URL{
		ShortCode:   "mylink",
		OriginalURL: "https://example.com",
		IsActive:    true,
	}

	mockService.On("ShortenURL", mock.Anything, mock.MatchedBy(func(req *domain.CreatedURLRequest) bool {
		return req.CustomAlias == "mylink"
	})).Return(mockURL, nil).Once()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "mylink", response["short_code"])

	mockService.AssertExpectations(t)
}

func TestShortenURL_WithExpiry(t *testing.T) {
	mockService := new(mocks.MockShortenerService)
	handler := NewShortenerHandler(mockService)
	router := setupTestRouter()
	router.POST("/api/shorten", handler.ShortenURL)

	reqBody := `{"url": "https://example.com", "expiry_hours": 24}`
	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	expiresAt := time.Now().Add(24 * time.Hour)
	mockURL := &domain.URL{
		ShortCode:   "abc1234",
		OriginalURL: "https://example.com",
		IsActive:    true,
		ExpiresAt:   &expiresAt,
	}

	mockService.On("ShortenURL", mock.Anything, mock.MatchedBy(func(req *domain.CreatedURLRequest) bool {
		return req.ExpiryHours == 24
	})).Return(mockURL, nil).Once()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotNil(t, response["expires_at"])

	mockService.AssertExpectations(t)
}

func TestRedirect_Success(t *testing.T) {
	mockService := new(mocks.MockShortenerService)
	handler := NewShortenerHandler(mockService)
	router := setupTestRouter()
	router.GET("/:shortCode", handler.Redirect)

	req := httptest.NewRequest("GET", "/abc1234", nil)
	w := httptest.NewRecorder()

	mockURL := &domain.URL{
		ShortCode:   "abc1234",
		OriginalURL: "https://example.com",
		IsActive:    true,
	}

	mockService.On("GetOriginalURL", mock.Anything, "abc1234").
		Return(mockURL, nil).Once()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMovedPermanently, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Location"))

	mockService.AssertExpectations(t)
}

func TestRedirect_NotFound(t *testing.T) {
	mockService := new(mocks.MockShortenerService)
	handler := NewShortenerHandler(mockService)
	router := setupTestRouter()
	router.GET("/:shortCode", handler.Redirect)

	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()

	mockService.On("GetOriginalURL", mock.Anything, "notfound").
		Return(nil, errors.New("URL not found")).Once()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "URL not found")

	mockService.AssertExpectations(t)
}

func TestRedirect_ServiceError(t *testing.T) {
	mockService := new(mocks.MockShortenerService)
	handler := NewShortenerHandler(mockService)
	router := setupTestRouter()
	router.GET("/:shortCode", handler.Redirect)

	req := httptest.NewRequest("GET", "/abc1234", nil)
	w := httptest.NewRecorder()

	mockService.On("GetOriginalURL", mock.Anything, "abc1234").
		Return(nil, errors.New("database connection failed")).Once()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	mockService.AssertExpectations(t)
}
