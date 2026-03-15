package server

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/reporting"
)

// --- Reporting Live API Test Helpers ---

// reportAdminToken returns a signed JWT for a user who is admin of the given org.
func reportAdminToken(t *testing.T, env *liveAuthEnv, orgID string) string {
	t.Helper()
	userID := "rpt_admin_" + t.Name()
	require.NoError(t, env.DB.Create(&models.OrgMembership{
		OrgID: orgID, UserID: userID, Role: models.RoleAdmin,
	}).Error)
	return env.SignToken(auth.JWTClaims{
		Subject:   userID,
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})
}

// reportViewerToken returns a signed JWT for a user who is viewer of the given org.
func reportViewerToken(t *testing.T, env *liveAuthEnv, orgID string) string {
	t.Helper()
	userID := "rpt_viewer_" + t.Name()
	require.NoError(t, env.DB.Create(&models.OrgMembership{
		OrgID: orgID, UserID: userID, Role: models.RoleViewer,
	}).Error)
	return env.SignToken(auth.JWTClaims{
		Subject:   userID,
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})
}

// seedReportingData creates an org with CRM/support spaces, boards, threads, and audit logs.
func seedReportingData(t *testing.T, env *liveAuthEnv) string {
	t.Helper()
	org := &models.Org{Name: "Report Org", Slug: "report-org-" + t.Name(), Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	// CRM space and board.
	crmSpace := &models.Space{OrgID: org.ID, Name: "CRM", Slug: "crm", Type: models.SpaceTypeCRM, Metadata: "{}"}
	require.NoError(t, env.DB.Create(crmSpace).Error)
	board := &models.Board{SpaceID: crmSpace.ID, Name: "Leads", Slug: "leads", Metadata: "{}"}
	require.NoError(t, env.DB.Create(board).Error)

	now := time.Now()
	stages := []string{"new_lead", "qualified", "proposal", "negotiation", "closed_won", "closed_lost"}

	threadIDs := make([]string, 0, 15)
	for i := 0; i < 15; i++ {
		stage := stages[i%len(stages)]
		assignee := fmt.Sprintf("rep_%d", i%3)
		score := (i * 7) % 100
		dealValue := 5000 + (i * 1000)
		meta := fmt.Sprintf(`{"stage":"%s","assigned_to":"%s","score":%d,"deal_value":%d}`, stage, assignee, score, dealValue)
		slug := fmt.Sprintf("lead-%d-%s", i, t.Name())
		thread := &models.Thread{
			BoardID:  board.ID,
			Title:    fmt.Sprintf("Lead %d", i),
			Slug:     slug,
			Metadata: meta,
			AuthorID: "author1",
		}
		require.NoError(t, env.DB.Create(thread).Error)
		env.DB.Exec("UPDATE threads SET created_at = ? WHERE id = ?", now.Add(-time.Duration(i)*24*time.Hour), thread.ID)
		threadIDs = append(threadIDs, thread.ID)
	}

	// Seed audit_log stage-change records for first 3 threads.
	for i := 0; i < 3; i++ {
		seedReportAuditLog(t, env, threadIDs[i], "new_lead", "qualified", now.Add(-time.Duration(i+5)*24*time.Hour))
		seedReportAuditLog(t, env, threadIDs[i], "qualified", "proposal", now.Add(-time.Duration(i+3)*24*time.Hour))
	}

	// Support space.
	suppSpace := &models.Space{OrgID: org.ID, Name: "Support", Slug: "support", Type: models.SpaceTypeSupport, Metadata: "{}"}
	require.NoError(t, env.DB.Create(suppSpace).Error)
	suppBoard := &models.Board{SpaceID: suppSpace.ID, Name: "Tickets", Slug: "tickets", Metadata: "{}"}
	require.NoError(t, env.DB.Create(suppBoard).Error)

	for i := 0; i < 5; i++ {
		status := []string{"open", "in_progress", "resolved"}[i%3]
		meta := fmt.Sprintf(`{"status":"%s","priority":"medium","assigned_to":"agent_%d"}`, status, i%2)
		thread := &models.Thread{
			BoardID:  suppBoard.ID,
			Title:    fmt.Sprintf("Ticket %d", i),
			Slug:     fmt.Sprintf("ticket-%d-%s", i, t.Name()),
			Metadata: meta,
			AuthorID: "author1",
		}
		require.NoError(t, env.DB.Create(thread).Error)
		env.DB.Exec("UPDATE threads SET created_at = ? WHERE id = ?", now.Add(-time.Duration(i)*24*time.Hour), thread.ID)
	}

	return org.ID
}

func seedReportAuditLog(t *testing.T, env *liveAuthEnv, threadID, fromStage, toStage string, createdAt time.Time) {
	t.Helper()
	al := &models.AuditLog{
		UserID:      "user1",
		Action:      "thread.updated",
		EntityType:  "thread",
		EntityID:    threadID,
		BeforeState: fmt.Sprintf(`{"stage":"%s"}`, fromStage),
		AfterState:  fmt.Sprintf(`{"stage":"%s"}`, toStage),
	}
	require.NoError(t, env.DB.Create(al).Error)
	env.DB.Exec("UPDATE audit_logs SET created_at = ? WHERE id = ?", createdAt, al.ID)
}

// --- Reporting Live API Tests ---

func TestLive_SalesMetrics_200(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	orgID := seedReportingData(t, env)
	token := reportAdminToken(t, env, orgID)

	from := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	to := time.Now().Add(24 * time.Hour).Format("2006-01-02")

	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/v1/orgs/%s/reports/sales?from=%s&to=%s", env.BaseURL, orgID, from, to), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var metrics reporting.SalesMetrics
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&metrics))
	assert.NotEmpty(t, metrics.PipelineFunnel)
	assert.NotEmpty(t, metrics.LeadVelocity)
	assert.NotNil(t, metrics.AvgDealValue)
	assert.NotEmpty(t, metrics.LeadsByAssignee)
	assert.NotEmpty(t, metrics.ScoreDistribution)
	assert.NotEmpty(t, metrics.StageConversionRates)
	assert.NotEmpty(t, metrics.AvgTimeInStage)
}

func TestLive_SalesExport_200(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	orgID := seedReportingData(t, env)
	token := reportAdminToken(t, env, orgID)

	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/v1/orgs/%s/reports/sales/export", env.BaseURL, orgID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/csv", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	reader := csv.NewReader(strings.NewReader(string(body)))
	records, err := reader.ReadAll()
	require.NoError(t, err)
	// Header + data rows.
	assert.Greater(t, len(records), 1)
	assert.Equal(t, []string{"id", "title", "stage", "assigned_to", "deal_value", "score", "created_at"}, records[0])
}

func TestLive_SalesMetrics_403_ViewerRole(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	orgID := seedReportingData(t, env)
	token := reportViewerToken(t, env, orgID)

	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/v1/orgs/%s/reports/sales", env.BaseURL, orgID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestLive_SupportMetrics_200(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	orgID := seedReportingData(t, env)
	token := reportAdminToken(t, env, orgID)

	from := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	to := time.Now().Add(24 * time.Hour).Format("2006-01-02")

	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/v1/orgs/%s/reports/support?from=%s&to=%s", env.BaseURL, orgID, from, to), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var metrics reporting.SupportMetrics
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&metrics))
	assert.NotEmpty(t, metrics.StatusBreakdown)
}

func TestLive_SupportExport_200(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	orgID := seedReportingData(t, env)
	token := reportAdminToken(t, env, orgID)

	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/v1/orgs/%s/reports/support/export", env.BaseURL, orgID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/csv", resp.Header.Get("Content-Type"))
}

func TestLive_SalesMetrics_EmptyAuditLog(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	// Create org with CRM threads but NO audit log entries.
	org := &models.Org{Name: "Empty Audit Org", Slug: "empty-audit-" + t.Name(), Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)
	token := reportAdminToken(t, env, org.ID)

	space := &models.Space{OrgID: org.ID, Name: "CRM", Slug: "crm", Type: models.SpaceTypeCRM, Metadata: "{}"}
	require.NoError(t, env.DB.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "Board", Slug: "board", Metadata: "{}"}
	require.NoError(t, env.DB.Create(board).Error)
	thread := &models.Thread{BoardID: board.ID, Title: "Lead", Slug: "lead-" + t.Name(), Metadata: `{"stage":"new_lead"}`, AuthorID: "a1"}
	require.NoError(t, env.DB.Create(thread).Error)

	from := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	to := time.Now().Add(24 * time.Hour).Format("2006-01-02")

	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/v1/orgs/%s/reports/sales?from=%s&to=%s", env.BaseURL, org.ID, from, to), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var metrics reporting.SalesMetrics
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&metrics))
	assert.Empty(t, metrics.StageConversionRates)
	assert.Empty(t, metrics.AvgTimeInStage)
}

func TestLive_SalesMetrics_InvalidDate(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	orgID := seedReportingData(t, env)
	token := reportAdminToken(t, env, orgID)

	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/v1/orgs/%s/reports/sales?from=invalid", env.BaseURL, orgID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
