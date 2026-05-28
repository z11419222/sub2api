package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type backupTestSettingRepo struct {
	mu   sync.Mutex
	data map[string]string
}

func newBackupTestSettingRepo() *backupTestSettingRepo {
	return &backupTestSettingRepo{data: make(map[string]string)}
}

func (r *backupTestSettingRepo) Get(_ context.Context, key string) (*service.Setting, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.data[key]
	if !ok {
		return nil, service.ErrSettingNotFound
	}
	return &service.Setting{Key: key, Value: value}, nil
}

func (r *backupTestSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.data[key], nil
}

func (r *backupTestSettingRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[key] = value
	return nil
}

func (r *backupTestSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.data[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (r *backupTestSettingRepo) SetMultiple(_ context.Context, settings map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, value := range settings {
		r.data[key] = value
	}
	return nil
}

func (r *backupTestSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make(map[string]string, len(r.data))
	for key, value := range r.data {
		result[key] = value
	}
	return result, nil
}

func (r *backupTestSettingRepo) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.data, key)
	return nil
}

type backupTestEncryptor struct{}

func (e *backupTestEncryptor) Encrypt(plaintext string) (string, error) {
	return "ENC:" + plaintext, nil
}

func (e *backupTestEncryptor) Decrypt(ciphertext string) (string, error) {
	if strings.HasPrefix(ciphertext, "ENC:") {
		return strings.TrimPrefix(ciphertext, "ENC:"), nil
	}
	return ciphertext, fmt.Errorf("not encrypted")
}

type backupTestDumper struct{}

func (d *backupTestDumper) Dump(_ context.Context) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte("dump"))), nil
}

func (d *backupTestDumper) Restore(_ context.Context, data io.Reader) error {
	_, err := io.ReadAll(data)
	return err
}

type backupTestObjectStore struct {
	listedObjects []service.BackupObjectInfo
}

func (s *backupTestObjectStore) Upload(_ context.Context, _ string, body io.Reader, _ string) (int64, error) {
	data, err := io.ReadAll(body)
	return int64(len(data)), err
}

func (s *backupTestObjectStore) Download(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte("download"))), nil
}

func (s *backupTestObjectStore) Delete(_ context.Context, _ string) error {
	return nil
}

func (s *backupTestObjectStore) PresignURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://example.com/" + key, nil
}

func (s *backupTestObjectStore) HeadBucket(_ context.Context) error {
	return nil
}

func (s *backupTestObjectStore) List(_ context.Context, prefix string) ([]service.BackupObjectInfo, error) {
	var result []service.BackupObjectInfo
	for _, object := range s.listedObjects {
		if strings.HasPrefix(object.Key, prefix) {
			result = append(result, object)
		}
	}
	return result, nil
}

func TestBackupHandler_DiscoverBackups(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newBackupTestSettingRepo()
	cfg := service.BackupS3Config{
		Bucket:          "test-bucket",
		AccessKeyID:     "AKID",
		SecretAccessKey: "ENC:secret",
		Prefix:          "backups",
	}
	cfgData, err := json.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, repo.Set(context.Background(), "backup_s3_config", string(cfgData)))

	store := &backupTestObjectStore{listedObjects: []service.BackupObjectInfo{
		{
			Key:          "backups/2026/05/28/testdb_20260528_080910.sql.gz",
			SizeBytes:    2048,
			LastModified: time.Date(2026, 5, 28, 8, 9, 10, 0, time.UTC),
		},
	}}
	backupService := service.NewBackupService(
		repo,
		&config.Config{Database: config.DatabaseConfig{DBName: "testdb"}},
		&backupTestEncryptor{},
		func(context.Context, *service.BackupS3Config) (service.BackupObjectStore, error) { return store, nil },
		&backupTestDumper{},
	)
	handler := NewBackupHandler(backupService, nil)
	router := gin.New()
	router.POST("/api/v1/admin/backups/discover", handler.DiscoverBackups)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/backups/discover", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items    []service.BackupRecord `json:"items"`
			Scanned  int                    `json:"scanned"`
			Imported int                    `json:"imported"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)
	require.Equal(t, 1, body.Data.Scanned)
	require.Equal(t, 1, body.Data.Imported)
	require.Len(t, body.Data.Items, 1)
	require.Equal(t, "discovered", body.Data.Items[0].TriggeredBy)
	require.Equal(t, "testdb_20260528_080910.sql.gz", body.Data.Items[0].FileName)
}
