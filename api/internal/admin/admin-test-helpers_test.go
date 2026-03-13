package admin

import (
	"context"
	"net/http"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/audit"
	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/gdpr"
	"github.com/abraderAI/crm-project/api/internal/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:?_journal_mode=WAL"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&models.Org{},
		&models.Space{},
		&models.Board{},
		&models.Thread{},
		&models.Message{},
		&models.OrgMembership{},
		&models.SpaceMembership{},
		&models.BoardMembership{},
		&models.AuditLog{},
		&models.PlatformAdmin{},
		&models.UserShadow{},
		&models.Vote{},
		&models.Notification{},
		&models.NotificationPreference{},
		&models.DigestSchedule{},
		&models.Upload{},
		&models.CallLog{},
		&models.Revision{},
		&models.APIKey{},
		&models.WebhookSubscription{},
		&models.WebhookDelivery{},
		&models.SystemSetting{},
		&models.FeatureFlag{},
		&models.AdminExport{},
		&models.APIUsageStat{},
		&models.LoginEvent{},
		&models.FailedAuth{},
		&models.LLMUsageLog{},
	))
	return db
}

func setupTestHandler(t *testing.T) (*Handler, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	svc := NewService(db)
	auditSvc := audit.NewService(db)
	gdprSvc := gdpr.NewService(db)
	h := NewHandler(svc, auditSvc, gdprSvc, nil)
	return h, db
}

func adminCtx() context.Context {
	return auth.SetUserContext(context.Background(), &auth.UserContext{
		UserID: "admin_user", AuthMethod: auth.AuthMethodJWT,
	})
}

func chiCtx(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}
