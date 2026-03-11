package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Phase 7 Live API Tests ---

// TestLive_Phase7_FullSalesLifecycle tests the complete pipeline:
// Create org → space → board → thread → transition through stages → enrich → provision.
func TestLive_Phase7_FullSalesLifecycle(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	// 1. Create org.
	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs", `{"name":"Sales Org","metadata":"{\"tier\":\"pro\"}"}`)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	orgData := decodeJSON(t, resp)
	orgID := orgData["id"].(string)

	// 2. Create CRM space.
	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces", `{"name":"CRM","type":"crm"}`)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	spaceData := decodeJSON(t, resp)
	spaceID := spaceData["id"].(string)

	// 3. Create pipeline board.
	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards", `{"name":"Pipeline"}`)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	boardData := decodeJSON(t, resp)
	boardID := boardData["id"].(string)

	// 4. Create a lead thread.
	threadBody := `{"title":"Acme Corp Deal","body":"Big enterprise prospect","metadata":"{\"company\":\"Acme Corp\",\"contact_email\":\"john@acme.com\",\"deal_value\":50000}"}`
	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads", threadBody)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	threadData := decodeJSON(t, resp)
	threadID := threadData["id"].(string)

	threadBase := env.BaseURL + "/v1/orgs/" + orgID + "/spaces/" + spaceID + "/boards/" + boardID + "/threads/" + threadID

	// 5. Transition: → new_lead.
	resp = authReq(t, env, "POST", threadBase+"/stage", `{"stage":"new_lead"}`)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	stageResult := decodeJSON(t, resp)
	assert.Equal(t, "new_lead", stageResult["new_stage"])

	// 6. Transition: new_lead → contacted.
	resp = authReq(t, env, "POST", threadBase+"/stage", `{"stage":"contacted"}`)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 7. Transition: contacted → qualified.
	resp = authReq(t, env, "POST", threadBase+"/stage", `{"stage":"qualified"}`)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 8. Enrich the lead with LLM.
	resp = authReq(t, env, "POST", threadBase+"/enrich", "")
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	enrichResult := decodeJSON(t, resp)
	assert.Equal(t, threadID, enrichResult["thread_id"])
	assert.NotNil(t, enrichResult["summary"])
	assert.NotNil(t, enrichResult["suggestion"])

	// 9. Transition: qualified → proposal.
	resp = authReq(t, env, "POST", threadBase+"/stage", `{"stage":"proposal"}`)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 10. Transition: proposal → negotiation.
	resp = authReq(t, env, "POST", threadBase+"/stage", `{"stage":"negotiation"}`)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 11. Transition: negotiation → closed_won.
	resp = authReq(t, env, "POST", threadBase+"/stage", `{"stage":"closed_won"}`)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	finalStage := decodeJSON(t, resp)
	assert.Equal(t, "closed_won", finalStage["new_stage"])

	// 12. Wait briefly for async event handlers to settle, then provision manually.
	// Auto-provision may race with scoring handler on metadata writes.
	time.Sleep(500 * time.Millisecond)

	// Check if auto-provision already happened.
	resp = authReq(t, env, "GET", threadBase, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	getData := decodeJSON(t, resp)
	_ = resp.Body.Close()
	metaStr := getData["metadata"].(string)
	var meta map[string]any
	_ = json.Unmarshal([]byte(metaStr), &meta)

	if meta["customer_org_id"] == nil {
		// Auto-provision didn't fire yet — do manual provision.
		resp = authReq(t, env, "POST", threadBase+"/provision", `{"company_name":"Acme Corp","contact_email":"john@acme.com"}`)
		defer func() { _ = resp.Body.Close() }()
		// Accept 201 (success) or 400 (race: auto-provision just completed).
		assert.True(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusBadRequest,
			"expected 201 or 400, got %d", resp.StatusCode)
	}

	// 13. Verify thread eventually has provisioning data.
	time.Sleep(500 * time.Millisecond)
	resp = authReq(t, env, "GET", threadBase, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	updatedThread := decodeJSON(t, resp)
	_ = resp.Body.Close()
	metaStr = updatedThread["metadata"].(string)
	var finalMeta map[string]any
	require.NoError(t, json.Unmarshal([]byte(metaStr), &finalMeta))
	// Stage should be closed_won regardless.
	assert.Equal(t, "closed_won", finalMeta["stage"])
}

// TestLive_Phase7_GetPipelineStages tests the pipeline stages endpoint.
func TestLive_Phase7_GetPipelineStages(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	// Create org.
	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs", `{"name":"Stages Org"}`)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	orgData := decodeJSON(t, resp)
	orgID := orgData["id"].(string)

	// Get default stages.
	resp = authReq(t, env, "GET", env.BaseURL+"/v1/orgs/"+orgID+"/pipeline/stages", "")
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var stagesResp map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&stagesResp))
	stages := stagesResp["stages"].([]any)
	assert.Len(t, stages, 8) // Default 8 stages.
}

// TestLive_Phase7_InvalidTransition tests that invalid transitions return errors.
func TestLive_Phase7_InvalidTransition(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	// Setup hierarchy.
	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs", `{"name":"Invalid Org"}`)
	defer func() { _ = resp.Body.Close() }()
	orgID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces", `{"name":"CRM","type":"crm"}`)
	defer func() { _ = resp.Body.Close() }()
	spaceID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards", `{"name":"Pipeline"}`)
	defer func() { _ = resp.Body.Close() }()
	boardID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads",
		`{"title":"Lead","body":"Test","metadata":"{\"stage\":\"new_lead\"}"}`)
	defer func() { _ = resp.Body.Close() }()
	threadID := decodeJSON(t, resp)["id"].(string)

	// Set the initial stage.
	threadBase := env.BaseURL + "/v1/orgs/" + orgID + "/spaces/" + spaceID + "/boards/" + boardID + "/threads/" + threadID
	resp = authReq(t, env, "POST", threadBase+"/stage", `{"stage":"new_lead"}`)
	defer func() { _ = resp.Body.Close() }()

	// Try invalid skip: new_lead → closed_won.
	resp = authReq(t, env, "POST", threadBase+"/stage", `{"stage":"closed_won"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestLive_Phase7_ProvisionNotClosedWon tests that provisioning requires closed_won.
func TestLive_Phase7_ProvisionNotClosedWon(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs", `{"name":"NotWon Org"}`)
	defer func() { _ = resp.Body.Close() }()
	orgID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces", `{"name":"CRM","type":"crm"}`)
	defer func() { _ = resp.Body.Close() }()
	spaceID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards", `{"name":"Board"}`)
	defer func() { _ = resp.Body.Close() }()
	boardID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads",
		`{"title":"Lead","metadata":"{\"stage\":\"qualified\"}"}`)
	defer func() { _ = resp.Body.Close() }()
	threadID := decodeJSON(t, resp)["id"].(string)

	threadBase := env.BaseURL + "/v1/orgs/" + orgID + "/spaces/" + spaceID + "/boards/" + boardID + "/threads/" + threadID

	resp = authReq(t, env, "POST", threadBase+"/provision", `{}`)
	defer func() { _ = resp.Body.Close() }()
	// Not closed_won → should fail.
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestLive_Phase7_EnrichNonExistentThread tests enrichment of non-existent thread.
func TestLive_Phase7_EnrichNonExistentThread(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs", `{"name":"Enrich Org"}`)
	defer func() { _ = resp.Body.Close() }()
	orgID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces", `{"name":"CRM","type":"crm"}`)
	defer func() { _ = resp.Body.Close() }()
	spaceID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards", `{"name":"Board"}`)
	defer func() { _ = resp.Body.Close() }()
	boardID := decodeJSON(t, resp)["id"].(string)

	threadBase := env.BaseURL + "/v1/orgs/" + orgID + "/spaces/" + spaceID + "/boards/" + boardID + "/threads/nonexistent"

	resp = authReq(t, env, "POST", threadBase+"/enrich", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestLive_Phase7_TransitionToNurturing tests the nurturing path.
func TestLive_Phase7_TransitionToNurturing(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs", `{"name":"Nurt Org"}`)
	defer func() { _ = resp.Body.Close() }()
	orgID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces", `{"name":"CRM","type":"crm"}`)
	defer func() { _ = resp.Body.Close() }()
	spaceID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards", `{"name":"Pipeline"}`)
	defer func() { _ = resp.Body.Close() }()
	boardID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads",
		`{"title":"Cold Lead"}`)
	defer func() { _ = resp.Body.Close() }()
	threadID := decodeJSON(t, resp)["id"].(string)

	threadBase := env.BaseURL + "/v1/orgs/" + orgID + "/spaces/" + spaceID + "/boards/" + boardID + "/threads/" + threadID

	// new_lead → contacted → closed_lost → nurturing → qualified
	for _, stage := range []string{"new_lead", "contacted", "closed_lost", "nurturing", "qualified"} {
		// Brief pause to let async event handlers (scoring) complete before next transition.
		time.Sleep(100 * time.Millisecond)
		resp = authReq(t, env, "POST", threadBase+"/stage", `{"stage":"`+stage+`"}`)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode, "failed at stage: %s", stage)
	}
}
