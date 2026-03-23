// Package seed provides idempotent database seeding for system-level entities.
package seed

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// SystemOrgSlug is the slug for the system-owned org that holds global spaces.
const SystemOrgSlug = "_system"

// DeftOrgSlug is the slug for the DEFT organization.
const DeftOrgSlug = "deft"

// globalSpaceDef defines a global space to seed.
type globalSpaceDef struct {
	Slug        string
	Name        string
	Description string
	Type        models.SpaceType
}

// globalSpaces lists the four system-seeded global spaces.
var globalSpaces = []globalSpaceDef{
	{Slug: "global-docs", Name: "Documentation", Description: "Public documentation, wiki, tutorials", Type: models.SpaceTypeKnowledgeBase},
	{Slug: "global-forum", Name: "Community Forum", Description: "Public community forum", Type: models.SpaceTypeCommunity},
	{Slug: "global-support", Name: "Support", Description: "Support tickets (org-filtered for customers)", Type: models.SpaceTypeSupport},
	{Slug: "global-leads", Name: "Leads", Description: "Lead records (DEFT-internal)", Type: models.SpaceTypeCRM},
}

// defaultBoardSlug is the slug for the single default board in each global space.
const defaultBoardSlug = "default"

// deftSpaceDef defines a DEFT department space to seed.
type deftSpaceDef struct {
	Slug        string
	Name        string
	Description string
	Type        models.SpaceType
}

// deftSpaces lists the three DEFT department spaces.
var deftSpaces = []deftSpaceDef{
	{Slug: "deft-sales", Name: "DEFT Sales", Description: "DEFT sales department", Type: models.SpaceTypeCRM},
	{Slug: "deft-support", Name: "DEFT Support", Description: "DEFT support department", Type: models.SpaceTypeSupport},
	{Slug: "deft-finance", Name: "DEFT Finance", Description: "DEFT finance department", Type: models.SpaceTypeGeneral},
}

// Run executes all seed operations idempotently.
// It creates the system org with global spaces, the deft org with department spaces,
// and seeds forum threads for the community.
func Run(db *gorm.DB) error {
	if err := seedSystemOrg(db); err != nil {
		return fmt.Errorf("seeding system org: %w", err)
	}
	if err := seedDeftOrg(db); err != nil {
		return fmt.Errorf("seeding deft org: %w", err)
	}
	if err := seedForum(db); err != nil {
		return fmt.Errorf("seeding forum: %w", err)
	}
	return nil
}

// seedForum seeds forum threads into the existing default board of global-forum.
// The default board is created by seedSystemOrg — we must not create a second board
// because FindDefaultBoard uses First() and multiple boards cause ambiguity.
func seedForum(db *gorm.DB) error {
	// Look up the global-forum space.
	var space models.Space
	err := db.Joins("JOIN orgs ON orgs.id = spaces.org_id").
		Where("orgs.slug = ? AND spaces.slug = ?", SystemOrgSlug, "global-forum").
		First(&space).Error
	if err != nil {
		return fmt.Errorf("finding global-forum space: %w", err)
	}

	// Use the existing default board (seeded by seedSystemOrg).
	var board models.Board
	if err := db.Where("space_id = ? AND slug = ?", space.ID, defaultBoardSlug).First(&board).Error; err != nil {
		return fmt.Errorf("finding forum default board: %w", err)
	}

	// Clean up stale "general-discussion" board from earlier deploy if it exists.
	// Move any threads that ended up there into the default board.
	var staleBoard models.Board
	if err := db.Where("space_id = ? AND slug = ?", space.ID, "general-discussion").First(&staleBoard).Error; err == nil {
		_ = db.Model(&models.Thread{}).Where("board_id = ?", staleBoard.ID).Update("board_id", board.ID).Error
		_ = db.Delete(&staleBoard).Error
	}

	return seedForumThreads(db, board.ID)
}

// seedSystemOrg creates the system org, its global spaces, and a default
// board in each space idempotently.
func seedSystemOrg(db *gorm.DB) error {
	org, err := findOrCreateOrg(db, SystemOrgSlug, "System", "System-owned organization for global spaces")
	if err != nil {
		return err
	}

	for _, sp := range globalSpaces {
		spaceID, err := findOrCreateSpaceID(db, org.ID, sp.Slug, sp.Name, sp.Description, sp.Type)
		if err != nil {
			return fmt.Errorf("seeding space %s: %w", sp.Slug, err)
		}
		if err := findOrCreateBoard(db, spaceID, defaultBoardSlug, "Default"); err != nil {
			return fmt.Errorf("seeding board for %s: %w", sp.Slug, err)
		}
	}
	return nil
}

// seedDeftOrg creates the deft org and its department spaces idempotently.
func seedDeftOrg(db *gorm.DB) error {
	org, err := findOrCreateOrg(db, DeftOrgSlug, "DEFT", "DEFT organization")
	if err != nil {
		return err
	}

	for _, sp := range deftSpaces {
		if _, err := findOrCreateSpaceID(db, org.ID, sp.Slug, sp.Name, sp.Description, sp.Type); err != nil {
			return fmt.Errorf("seeding space %s: %w", sp.Slug, err)
		}
	}
	return nil
}

// findOrCreateOrg looks up an org by slug and creates it if missing.
func findOrCreateOrg(db *gorm.DB, slug, name, description string) (*models.Org, error) {
	var org models.Org
	result := db.Where("slug = ?", slug).First(&org)
	if result.Error == nil {
		return &org, nil
	}
	if result.Error != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("looking up org %s: %w", slug, result.Error)
	}

	org = models.Org{
		Name:        name,
		Slug:        slug,
		Description: description,
		Metadata:    "{}",
	}
	if err := db.Create(&org).Error; err != nil {
		return nil, fmt.Errorf("creating org %s: %w", slug, err)
	}
	return &org, nil
}

// findOrCreateSpaceID looks up a space by org ID and slug (creating it if missing)
// and returns the space ID.
func findOrCreateSpaceID(db *gorm.DB, orgID, slug, name, description string, spaceType models.SpaceType) (string, error) {
	var space models.Space
	err := db.Where("org_id = ? AND slug = ?", orgID, slug).First(&space).Error
	if err == nil {
		return space.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("checking space %s: %w", slug, err)
	}

	space = models.Space{
		OrgID:       orgID,
		Name:        name,
		Slug:        slug,
		Description: description,
		Type:        spaceType,
		Metadata:    "{}",
	}
	if err := db.Create(&space).Error; err != nil {
		return "", fmt.Errorf("creating space %s: %w", slug, err)
	}
	return space.ID, nil
}

// findOrCreateBoard looks up a board by space ID and slug, creating it if missing.
func findOrCreateBoard(db *gorm.DB, spaceID, slug, name string) error {
	var count int64
	if err := db.Model(&models.Board{}).Where("space_id = ? AND slug = ?", spaceID, slug).Count(&count).Error; err != nil {
		return fmt.Errorf("checking board %s: %w", slug, err)
	}
	if count > 0 {
		return nil
	}

	board := models.Board{
		SpaceID:  spaceID,
		Name:     name,
		Slug:     slug,
		Metadata: "{}",
	}
	if err := db.Create(&board).Error; err != nil {
		return fmt.Errorf("creating board %s: %w", slug, err)
	}
	return nil
}
