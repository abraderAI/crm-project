// Package seed provides idempotent database seeding for system-level entities.
package seed

import (
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
// It creates the system org with global spaces and the deft org with department spaces.
func Run(db *gorm.DB) error {
	if err := seedSystemOrg(db); err != nil {
		return fmt.Errorf("seeding system org: %w", err)
	}
	if err := seedDeftOrg(db); err != nil {
		return fmt.Errorf("seeding deft org: %w", err)
	}
	return nil
}

// seedSystemOrg creates the system org and its global spaces idempotently.
func seedSystemOrg(db *gorm.DB) error {
	org, err := findOrCreateOrg(db, SystemOrgSlug, "System", "System-owned organization for global spaces")
	if err != nil {
		return err
	}

	for _, sp := range globalSpaces {
		if err := findOrCreateSpace(db, org.ID, sp.Slug, sp.Name, sp.Description, sp.Type); err != nil {
			return fmt.Errorf("seeding space %s: %w", sp.Slug, err)
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
		if err := findOrCreateSpace(db, org.ID, sp.Slug, sp.Name, sp.Description, sp.Type); err != nil {
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

// findOrCreateSpace looks up a space by org ID and slug, creating it if missing.
func findOrCreateSpace(db *gorm.DB, orgID, slug, name, description string, spaceType models.SpaceType) error {
	var count int64
	if err := db.Model(&models.Space{}).Where("org_id = ? AND slug = ?", orgID, slug).Count(&count).Error; err != nil {
		return fmt.Errorf("checking space %s: %w", slug, err)
	}
	if count > 0 {
		return nil
	}

	space := models.Space{
		OrgID:       orgID,
		Name:        name,
		Slug:        slug,
		Description: description,
		Type:        spaceType,
		Metadata:    "{}",
	}
	if err := db.Create(&space).Error; err != nil {
		return fmt.Errorf("creating space %s: %w", slug, err)
	}
	return nil
}
