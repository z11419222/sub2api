//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type copyAccountRepoStub struct {
	accountRepoStub
	source       *Account
	listAccounts []Account
	created      *Account
	boundAccount int64
	boundGroups  []int64
}

func (s *copyAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	if s.source == nil || s.source.ID != id {
		return nil, ErrAccountNotFound
	}
	return s.source, nil
}

func (s *copyAccountRepoStub) Create(_ context.Context, account *Account) error {
	account.ID = 9001
	s.created = account
	return nil
}

func (s *copyAccountRepoStub) BindGroups(_ context.Context, accountID int64, groupIDs []int64) error {
	s.boundAccount = accountID
	s.boundGroups = append([]int64(nil), groupIDs...)
	return nil
}

func (s *copyAccountRepoStub) ListWithFilters(_ context.Context, _ pagination.PaginationParams, _, _, _, _ string, _ int64, _ string) ([]Account, *pagination.PaginationResult, error) {
	return s.listAccounts, &pagination.PaginationResult{Total: int64(len(s.listAccounts))}, nil
}

func TestAdminService_CopyAccount_CopiesAPIKeyAccountWithoutRuntimeState(t *testing.T) {
	now := time.Now().UTC()
	rateMultiplier := 0.8
	loadFactor := 7
	proxyID := int64(44)
	expiresAt := now.Add(24 * time.Hour)
	note := "keeps operator notes"
	source := &Account{
		ID:       10,
		Name:     "prod-key",
		Notes:    &note,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Status:   StatusError,
		ProxyID:  &proxyID,
		GroupIDs: []int64{2, 5},
		Extra: map[string]any{
			"model_mapping":              map[string]any{"gpt-5.5": "gpt-5.5"},
			"quota_limit":                float64(100),
			"codex_primary_usage_5h":     float64(18),
			"codex_secondary_usage_5h":   float64(9),
			"codex_5h_window_start":      now.Format(time.RFC3339),
			"codex_7d_window_start":      now.Format(time.RFC3339),
			"codex_usage_updated_at":     now.Format(time.RFC3339),
			"passive_usage_last_seen_at": now.Format(time.RFC3339),
			"quota_used":                 float64(77),
			"quota_daily_used":           float64(12),
			"quota_daily_start":          now.Format(time.RFC3339),
			"quota_weekly_used":          float64(42),
			"quota_weekly_start":         now.Format(time.RFC3339),
			"quota_daily_reset_at":       now.Format(time.RFC3339),
			"quota_weekly_reset_at":      now.Format(time.RFC3339),
			"model_rate_limits":          map[string]any{"gpt-5.5": now.Format(time.RFC3339)},
			"session_window_utilization": float64(0.7),
		},
		Credentials: map[string]any{
			"api_key":  "sk-secret-source",
			"base_url": "https://api.example.test/v1",
		},
		Concurrency:             3,
		Priority:                9,
		RateMultiplier:          &rateMultiplier,
		LoadFactor:              &loadFactor,
		ErrorMessage:            "previous failure",
		LastUsedAt:              &now,
		ExpiresAt:               &expiresAt,
		AutoPauseOnExpired:      false,
		Schedulable:             false,
		RateLimitResetAt:        &now,
		OverloadUntil:           &now,
		TempUnschedulableUntil:  &now,
		TempUnschedulableReason: "runtime penalty",
		SessionWindowStart:      &now,
		SessionWindowEnd:        &now,
		SessionWindowStatus:     "active",
	}
	repo := &copyAccountRepoStub{
		source: source,
		listAccounts: []Account{
			{ID: 10, Name: "prod-key", Platform: PlatformOpenAI, Type: AccountTypeAPIKey},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	copied, err := svc.CopyAccount(context.Background(), source.ID)

	require.NoError(t, err)
	require.NotNil(t, copied)
	require.NotNil(t, repo.created)
	require.Equal(t, int64(9001), copied.ID)
	require.Equal(t, "prod-key (copy)", repo.created.Name)
	require.Equal(t, PlatformOpenAI, repo.created.Platform)
	require.Equal(t, AccountTypeAPIKey, repo.created.Type)
	require.Equal(t, StatusActive, repo.created.Status)
	require.True(t, repo.created.Schedulable)
	require.Empty(t, repo.created.ErrorMessage)
	require.Nil(t, repo.created.LastUsedAt)
	require.Nil(t, repo.created.RateLimitResetAt)
	require.Nil(t, repo.created.OverloadUntil)
	require.Nil(t, repo.created.TempUnschedulableUntil)
	require.Empty(t, repo.created.TempUnschedulableReason)
	require.Nil(t, repo.created.SessionWindowStart)
	require.Nil(t, repo.created.SessionWindowEnd)
	require.Empty(t, repo.created.SessionWindowStatus)
	require.Equal(t, []int64{2, 5}, repo.boundGroups)
	require.Equal(t, int64(9001), repo.boundAccount)
	require.Equal(t, "sk-secret-source", repo.created.Credentials["api_key"])
	require.Equal(t, "https://api.example.test/v1", repo.created.Credentials["base_url"])
	require.Equal(t, map[string]any{"gpt-5.5": "gpt-5.5"}, repo.created.Extra["model_mapping"])
	require.Equal(t, float64(100), repo.created.Extra["quota_limit"])
	for _, key := range []string{"quota_used", "quota_daily_used", "quota_daily_start", "quota_weekly_used", "quota_weekly_start", "quota_daily_reset_at", "quota_weekly_reset_at", "model_rate_limits", "session_window_utilization", "codex_primary_usage_5h", "codex_secondary_usage_5h", "codex_5h_window_start", "codex_7d_window_start", "codex_usage_updated_at", "passive_usage_last_seen_at"} {
		require.NotContains(t, repo.created.Extra, key)
	}
	require.Equal(t, proxyID, *repo.created.ProxyID)
	require.Equal(t, 3, repo.created.Concurrency)
	require.Equal(t, 9, repo.created.Priority)
	require.Equal(t, rateMultiplier, *repo.created.RateMultiplier)
	require.Equal(t, loadFactor, *repo.created.LoadFactor)
	require.Equal(t, expiresAt.Unix(), repo.created.ExpiresAt.Unix())
	require.False(t, repo.created.AutoPauseOnExpired)

	source.Credentials["api_key"] = "sk-mutated"
	source.Extra["model_mapping"].(map[string]any)["gpt-5.5"] = "mutated"
	require.Equal(t, "sk-secret-source", repo.created.Credentials["api_key"])
	require.Equal(t, map[string]any{"gpt-5.5": "gpt-5.5"}, repo.created.Extra["model_mapping"])
}

func TestAdminService_CopyAccount_GeneratesNextCopyName(t *testing.T) {
	source := &Account{ID: 10, Name: "prod-key", Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	repo := &copyAccountRepoStub{
		source: source,
		listAccounts: []Account{
			{ID: 10, Name: "prod-key", Platform: PlatformOpenAI, Type: AccountTypeAPIKey},
			{ID: 11, Name: "prod-key (copy)", Platform: PlatformOpenAI, Type: AccountTypeAPIKey},
			{ID: 12, Name: "prod-key (copy 2)", Platform: PlatformOpenAI, Type: AccountTypeAPIKey},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	_, err := svc.CopyAccount(context.Background(), source.ID)

	require.NoError(t, err)
	require.Equal(t, "prod-key (copy 3)", repo.created.Name)
}

func TestAdminService_CopyAccount_RejectsNonAPIKeyAccounts(t *testing.T) {
	repo := &copyAccountRepoStub{
		source: &Account{ID: 10, Name: "oauth-account", Platform: PlatformOpenAI, Type: AccountTypeOAuth},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	copied, err := svc.CopyAccount(context.Background(), 10)

	require.Nil(t, copied)
	require.Error(t, err)
	require.Nil(t, repo.created)
}
