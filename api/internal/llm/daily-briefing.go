package llm

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/notification"
)

// DailyBriefingConfig holds configuration for the daily briefing job.
type DailyBriefingConfig struct {
	Provider  LLMProvider
	DB        *gorm.DB
	NotifRepo *notification.Repository
	Logger    *slog.Logger
	// Hour is the UTC hour at which to run (default 8).
	Hour int
}

// StartDailyBriefing launches a background goroutine that fires daily at
// the configured UTC hour. For each opted-in user, it assembles context and
// delivers a briefing notification. This function returns immediately.
func StartDailyBriefing(cfg DailyBriefingConfig) {
	if cfg.Hour == 0 {
		cfg.Hour = 8
	}
	go runDailyBriefingLoop(cfg)
}

// runDailyBriefingLoop is the main loop for the daily briefing job.
func runDailyBriefingLoop(cfg DailyBriefingConfig) {
	for {
		now := time.Now().UTC()
		next := nextOccurrence(now, cfg.Hour)
		timer := time.NewTimer(time.Until(next))
		<-timer.C

		cfg.Logger.Info("daily briefing: starting run")
		runBriefingForOptedInUsers(cfg)
		cfg.Logger.Info("daily briefing: run complete")
	}
}

// nextOccurrence returns the next time.Time at the given UTC hour.
// If the current time is past that hour today, returns tomorrow.
func nextOccurrence(now time.Time, hour int) time.Time {
	target := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.UTC)
	if now.After(target) {
		target = target.Add(24 * time.Hour)
	}
	return target
}

// runBriefingForOptedInUsers finds opted-in users and generates briefings.
func runBriefingForOptedInUsers(cfg DailyBriefingConfig) {
	ctx := context.Background()

	// Find users who have opted in to daily_crm_brief.
	userIDs, err := getOptedInUsers(ctx, cfg.DB)
	if err != nil {
		cfg.Logger.Error("daily briefing: failed to find opted-in users", slog.String("error", err.Error()))
		return
	}

	if len(userIDs) == 0 {
		cfg.Logger.Info("daily briefing: no opted-in users")
		return
	}

	handler := NewCRMHandler(cfg.Provider, cfg.DB, "")

	for _, userID := range userIDs {
		opps, _ := handler.loadUserOpportunities(ctx, userID)
		tasks, _ := handler.loadUserTasks(ctx, userID)
		msgs, _ := handler.loadRecentMessages(ctx, userID)

		// Skip users with zero open records.
		if len(opps) == 0 && len(tasks) == 0 {
			continue
		}

		result, err := cfg.Provider.Briefing(ctx, userID, opps, tasks, msgs)
		if err != nil {
			cfg.Logger.Error("daily briefing: LLM call failed",
				slog.String("user_id", userID),
				slog.String("error", err.Error()))
			continue
		}

		// Deliver as in-app notification.
		notif := &models.Notification{
			UserID:     userID,
			Type:       "daily_crm_brief",
			Title:      "Daily CRM Briefing",
			Body:       result,
			EntityType: "briefing",
		}
		if err := cfg.NotifRepo.Create(ctx, notif); err != nil {
			cfg.Logger.Error("daily briefing: notification create failed",
				slog.String("user_id", userID),
				slog.String("error", err.Error()))
		}
	}
}

// getOptedInUsers returns user IDs with notification_preferences.daily_crm_brief enabled.
// Uses the NotificationPreference model: event_type="daily_crm_brief", channel="in_app", enabled=true.
func getOptedInUsers(ctx context.Context, db *gorm.DB) ([]string, error) {
	var prefs []models.NotificationPreference
	err := db.WithContext(ctx).
		Where("event_type = ? AND channel = ? AND enabled = ?", "daily_crm_brief", "in_app", true).
		Find(&prefs).Error
	if err != nil {
		return nil, err
	}
	userIDs := make([]string, len(prefs))
	for i, p := range prefs {
		userIDs[i] = p.UserID
	}
	return userIDs, nil
}

// RunBriefingOnce executes one iteration of the daily briefing for testing.
// It is exported to allow test code to invoke the briefing logic directly.
func RunBriefingOnce(cfg DailyBriefingConfig) {
	runBriefingForOptedInUsers(cfg)
}
