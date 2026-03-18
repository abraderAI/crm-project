package reporting_test

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/reporting"
)

// --- Helpers ---

// seedAdminReportingData creates 3 orgs with varied support + CRM data.
func seedAdminReportingData(t *testing.T, env *liveAuthEnv) (org1ID, org2ID, org3ID string) {
	t.Helper()

	now := time.Now()

	// Org 1: 3 support tickets, 2 CRM leads.
	o1 := createTestOrg(t, env.DB, "admin-rpt-org1")
	s1 := createSupportSpace(t, env.DB, o1.ID)
	b1 := createBoard(t, env.DB, s1.ID)
	createThread(t, env.DB, b1.ID, "S1-T1", "author1", `{"status":"open","priority":"high"}`, now)
	createThread(t, env.DB, b1.ID, "S1-T2", "author2", `{"status":"resolved","priority":"low"}`, now.Add(-24*time.Hour))
	t1 := createThread(t, env.DB, b1.ID, "S1-T3", "author3", `{"status":"closed","priority":"medium"}`, now.Add(-48*time.Hour))
	env.DB.Exec("UPDATE threads SET updated_at = datetime(created_at, '+8 hours') WHERE id = ?", t1.ID)

	crm1 := &models.Space{OrgID: o1.ID, Name: "CRM", Slug: "crm-" + o1.ID[:8], Type: models.SpaceTypeCRM, Metadata: "{}"}
	require.NoError(t, env.DB.Create(crm1).Error)
	cb1 := createBoard(t, env.DB, crm1.ID)
	createThread(t, env.DB, cb1.ID, "C1-L1", "rep1", `{"stage":"new_lead","deal_value":10000}`, now)
	createThread(t, env.DB, cb1.ID, "C1-L2", "rep2", `{"stage":"closed_won","deal_value":20000}`, now)

	// Org 2: 2 support tickets, 3 CRM leads.
	o2 := createTestOrg(t, env.DB, "admin-rpt-org2")
	s2 := createSupportSpace(t, env.DB, o2.ID)
	b2 := createBoard(t, env.DB, s2.ID)
	createThread(t, env.DB, b2.ID, "S2-T1", "author4", `{"status":"open","priority":"high"}`, now)
	createThread(t, env.DB, b2.ID, "S2-T2", "author5", `{"status":"in_progress","priority":"medium"}`, now)

	crm2 := &models.Space{OrgID: o2.ID, Name: "CRM", Slug: "crm-" + o2.ID[:8], Type: models.SpaceTypeCRM, Metadata: "{}"}
	require.NoError(t, env.DB.Create(crm2).Error)
	cb2 := createBoard(t, env.DB, crm2.ID)
	createThread(t, env.DB, cb2.ID, "C2-L1", "rep3", `{"stage":"qualified","deal_value":15000}`, now)
	createThread(t, env.DB, cb2.ID, "C2-L2", "rep4", `{"stage":"closed_lost","deal_value":5000}`, now)
	createThread(t, env.DB, cb2.ID, "C2-L3", "rep5", `{"stage":"new_lead","deal_value":25000}`, now)

	// Org 3: 1 support ticket, 1 CRM lead.
	o3 := createTestOrg(t, env.DB, "admin-rpt-org3")
	s3 := createSupportSpace(t, env.DB, o3.ID)
	b3 := createBoard(t, env.DB, s3.ID)
	createThread(t, env.DB, b3.ID, "S3-T1", "author6", `{"status":"open","priority":"low"}`, now)

	crm3 := &models.Space{OrgID: o3.ID, Name: "CRM", Slug: "crm-" + o3.ID[:8], Type: models.SpaceTypeCRM, Metadata: "{}"}
	require.NoError(t, env.DB.Create(crm3).Error)
	cb3 := createBoard(t, env.DB, crm3.ID)
	createThread(t, env.DB, cb3.ID, "C3-L1", "rep6", `{"stage":"new_lead","deal_value":30000}`, now)

	return o1.ID, o2.ID, o3.ID
}

// createPlatformAdmin inserts a platform admin record directly.
func createPlatformAdmin(t *testing.T, env *liveAuthEnv, userID string) string {
	t.Helper()
	admin := models.PlatformAdmin{
		UserID:    userID,
		GrantedBy: "test",
		IsActive:  true,
	}
	require.NoError(t, env.DB.Create(&admin).Error)

	return env.SignToken(auth.JWTClaims{
		Subject:   userID,
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})
}

// --- Live API Tests ---

func TestLive_AdminSupportMetrics_OK(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	seedAdminReportingData(t, env)
	token := createPlatformAdmin(t, env, "admin-report-user")

	from := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	to := time.Now().Format("2006-01-02")

	req, err := http.NewRequest(http.MethodGet,
		env.BaseURL+"/v1/admin/reports/support?from="+from+"&to="+to, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var metrics reporting.AdminSupportMetrics
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&metrics))

	// 3 orgs seeded → 3 in breakdown.
	assert.Len(t, metrics.OrgBreakdown, 3)

	// Platform totals should match sum of breakdown.
	platformTotal := int64(0)
	for _, v := range metrics.StatusBreakdown {
		platformTotal += v
	}
	breakdownTotal := int64(0)
	for _, b := range metrics.OrgBreakdown {
		breakdownTotal += b.TotalInRange
	}
	assert.Equal(t, platformTotal, breakdownTotal)
}

func TestLive_AdminSalesMetrics_OK(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	seedAdminReportingData(t, env)
	token := createPlatformAdmin(t, env, "admin-sales-user")

	from := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	to := time.Now().Format("2006-01-02")

	req, err := http.NewRequest(http.MethodGet,
		env.BaseURL+"/v1/admin/reports/sales?from="+from+"&to="+to, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var metrics reporting.AdminSalesMetrics
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&metrics))

	assert.NotEmpty(t, metrics.OrgBreakdown)
	assert.NotEmpty(t, metrics.PipelineFunnel)
}

func TestLive_AdminSupportMetrics_Forbidden_NonPlatformAdmin(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	// Create an org admin (not platform admin).
	org := createTestOrg(t, env.DB, "admin-forbidden-org")
	createAdminMembership(t, env.DB, org.ID, "org-admin-user")

	token := env.SignToken(auth.JWTClaims{
		Subject:   "org-admin-user",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	req, err := http.NewRequest(http.MethodGet,
		env.BaseURL+"/v1/admin/reports/support", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestLive_AdminSupportExport_CSV(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	seedAdminReportingData(t, env)
	token := createPlatformAdmin(t, env, "admin-export-user")

	req, err := http.NewRequest(http.MethodGet,
		env.BaseURL+"/v1/admin/reports/support/export", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/csv", resp.Header.Get("Content-Type"))

	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Header + data rows.
	assert.Greater(t, len(records), 1)

	// First column is org_id.
	assert.Equal(t, "org_id", records[0][0])
	assert.Equal(t, "org_slug", records[0][1])
}

func TestLive_AdminSalesExport_CSV(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	seedAdminReportingData(t, env)
	token := createPlatformAdmin(t, env, "admin-sales-export")

	req, err := http.NewRequest(http.MethodGet,
		env.BaseURL+"/v1/admin/reports/sales/export", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/csv", resp.Header.Get("Content-Type"))

	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	assert.Greater(t, len(records), 1)
	assert.Equal(t, "org_id", records[0][0])
	assert.Equal(t, "org_slug", records[0][1])
}

func TestLive_AdminSalesMetrics_Forbidden(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	token := env.SignToken(auth.JWTClaims{
		Subject:   "regular-user",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	req, err := http.NewRequest(http.MethodGet,
		env.BaseURL+"/v1/admin/reports/sales", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestLive_AdminSupportMetrics_Unauthenticated(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	resp, err := http.Get(env.BaseURL + "/v1/admin/reports/support")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestLive_AdminSupportMetrics_BadDate(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	token := createPlatformAdmin(t, env, "admin-baddate")

	req, err := http.NewRequest(http.MethodGet,
		env.BaseURL+"/v1/admin/reports/support?from=invalid", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestLive_AdminSupportExport_Forbidden(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	token := env.SignToken(auth.JWTClaims{
		Subject:   "non-admin-export",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	endpoints := []string{
		"/v1/admin/reports/support/export",
		"/v1/admin/reports/sales/export",
	}

	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, env.BaseURL+ep, nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		})
	}
}

// Verify the admin support CSV export has data from multiple orgs.
func TestLive_AdminSupportExport_MultiOrgData(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	seedAdminReportingData(t, env)
	token := createPlatformAdmin(t, env, "admin-multi-export")

	req, err := http.NewRequest(http.MethodGet,
		env.BaseURL+"/v1/admin/reports/support/export", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Collect unique org_ids from data rows.
	orgIDs := make(map[string]bool)
	for _, row := range records[1:] {
		orgIDs[row[0]] = true
	}
	// All 3 orgs should appear.
	assert.GreaterOrEqual(t, len(orgIDs), 3)
}

// --- Fuzz tests ---

func FuzzAdminDateParams(f *testing.F) {
	f.Add("2026-03-01", "2026-03-15")
	f.Add("", "")
	f.Add("not-a-date", "2026-01-01")
	f.Add("2026-01-01", "garbage")
	f.Add("99999", "00-00-00")
	for i := 0; i < 50; i++ {
		f.Add(strings.Repeat("a", i), strings.Repeat("z", i))
	}

	f.Fuzz(func(t *testing.T, fromStr, toStr string) {
		db := setupTestDB(t)
		handler := reporting.NewHandler(reporting.NewService(reporting.NewRepository(db)), db)

		req, err := http.NewRequest(http.MethodGet, "/v1/admin/reports/support", nil)
		require.NoError(t, err)
		q := req.URL.Query()
		if fromStr != "" {
			q.Set("from", fromStr)
		}
		if toStr != "" {
			q.Set("to", toStr)
		}
		req.URL.RawQuery = q.Encode()

		w := httptest.NewRecorder()
		handler.GetAdminSupportMetrics(w, req)

		// Should not panic; response must be 200 or 400.
		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, w.Code)
	})
}
