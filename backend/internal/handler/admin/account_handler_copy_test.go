package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type accountCopyResponse struct {
	Code int `json:"code"`
	Data struct {
		ID                int64           `json:"id"`
		Name              string          `json:"name"`
		Type              string          `json:"type"`
		Credentials       map[string]any  `json:"credentials"`
		CredentialsStatus map[string]bool `json:"credentials_status"`
	} `json:"data"`
}

func setupAccountCopyRouter() (*gin.Engine, *stubAdminService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminSvc := newStubAdminService()

	h := NewAccountHandler(
		adminSvc,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	router.POST("/api/v1/admin/accounts/:id/copy", h.Copy)
	return router, adminSvc
}

func TestAccountHandlerCopy_RedactsRawAPIKey(t *testing.T) {
	router, adminSvc := setupAccountCopyRouter()
	adminSvc.copiedAccount = &service.Account{
		ID:          31,
		Name:        "prod-key (copy)",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-secret", "base_url": "https://api.example.com"},
		Status:      service.StatusActive,
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/3/copy", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotContains(t, rec.Body.String(), "sk-secret")
	require.Equal(t, int64(3), adminSvc.copiedAccountID)

	var resp accountCopyResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(31), resp.Data.ID)
	require.Equal(t, "prod-key (copy)", resp.Data.Name)
	require.Equal(t, service.AccountTypeAPIKey, resp.Data.Type)
	require.NotContains(t, resp.Data.Credentials, "api_key")
	require.Equal(t, "https://api.example.com", resp.Data.Credentials["base_url"])
	require.True(t, resp.Data.CredentialsStatus["has_api_key"])
}
