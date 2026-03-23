package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// --- ChannelType ---

func TestChannelType_IsValid(t *testing.T) {
	assert.True(t, models.ChannelTypeEmail.IsValid())
	assert.True(t, models.ChannelTypeVoice.IsValid())
	assert.True(t, models.ChannelTypeChat.IsValid())
	assert.False(t, models.ChannelType("sms").IsValid())
	assert.False(t, models.ChannelType("").IsValid())
}

func TestValidChannelTypes(t *testing.T) {
	types := models.ValidChannelTypes()
	assert.Len(t, types, 3)
	assert.Contains(t, types, models.ChannelTypeEmail)
	assert.Contains(t, types, models.ChannelTypeVoice)
	assert.Contains(t, types, models.ChannelTypeChat)
}

// --- DLQStatus ---

func TestDLQStatus_IsValid(t *testing.T) {
	assert.True(t, models.DLQStatusFailed.IsValid())
	assert.True(t, models.DLQStatusRetrying.IsValid())
	assert.True(t, models.DLQStatusResolved.IsValid())
	assert.True(t, models.DLQStatusDismissed.IsValid())
	assert.False(t, models.DLQStatus("unknown").IsValid())
	assert.False(t, models.DLQStatus("").IsValid())
}

func TestValidDLQStatuses(t *testing.T) {
	statuses := models.ValidDLQStatuses()
	assert.Len(t, statuses, 4)
	assert.Contains(t, statuses, models.DLQStatusFailed)
	assert.Contains(t, statuses, models.DLQStatusRetrying)
	assert.Contains(t, statuses, models.DLQStatusResolved)
	assert.Contains(t, statuses, models.DLQStatusDismissed)
}

// --- RoutingAction ---

func TestRoutingAction_IsValid(t *testing.T) {
	assert.True(t, models.RoutingActionSupportTicket.IsValid())
	assert.True(t, models.RoutingActionSalesLead.IsValid())
	assert.True(t, models.RoutingActionGeneral.IsValid())
	assert.False(t, models.RoutingAction("unknown").IsValid())
	assert.False(t, models.RoutingAction("").IsValid())
}

func TestValidRoutingActions(t *testing.T) {
	actions := models.ValidRoutingActions()
	assert.GreaterOrEqual(t, len(actions), 3)
	assert.Contains(t, actions, models.RoutingActionSupportTicket)
	assert.Contains(t, actions, models.RoutingActionSalesLead)
	assert.Contains(t, actions, models.RoutingActionGeneral)
}

// --- FlagStatus ---

func TestFlagStatus_IsValid(t *testing.T) {
	assert.True(t, models.FlagStatusOpen.IsValid())
	assert.True(t, models.FlagStatusResolved.IsValid())
	assert.True(t, models.FlagStatusDismissed.IsValid())
	assert.False(t, models.FlagStatus("pending").IsValid())
	assert.False(t, models.FlagStatus("").IsValid())
}

// --- LLMUsageLog.BeforeCreate ---

func TestLLMUsageLog_BeforeCreate_GeneratesID(t *testing.T) {
	log := &models.LLMUsageLog{}
	require.NoError(t, log.BeforeCreate(nil))
	assert.NotEmpty(t, log.ID)
}

func TestLLMUsageLog_BeforeCreate_PreservesExistingID(t *testing.T) {
	existing := "existing-llm-id"
	log := &models.LLMUsageLog{ID: existing}
	require.NoError(t, log.BeforeCreate(nil))
	assert.Equal(t, existing, log.ID)
}

// --- LoginEvent.BeforeCreate ---

func TestLoginEvent_BeforeCreate_GeneratesID(t *testing.T) {
	event := &models.LoginEvent{}
	require.NoError(t, event.BeforeCreate(nil))
	assert.NotEmpty(t, event.ID)
}

func TestLoginEvent_BeforeCreate_PreservesExistingID(t *testing.T) {
	existing := "existing-login-id"
	event := &models.LoginEvent{ID: existing}
	require.NoError(t, event.BeforeCreate(nil))
	assert.Equal(t, existing, event.ID)
}

// --- AdminExport.BeforeCreate ---

func TestAdminExport_BeforeCreate_GeneratesID(t *testing.T) {
	export := &models.AdminExport{}
	require.NoError(t, export.BeforeCreate(nil))
	assert.NotEmpty(t, export.ID)
}

func TestAdminExport_BeforeCreate_PreservesExistingID(t *testing.T) {
	existing := "existing-export-id"
	export := &models.AdminExport{ID: existing}
	require.NoError(t, export.BeforeCreate(nil))
	assert.Equal(t, existing, export.ID)
}
