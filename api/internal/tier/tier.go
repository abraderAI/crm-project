// Package tier implements user tier resolution and home preferences management.
package tier

// Tier represents a user's access tier level (1-6).
type Tier int

const (
	// TierAnonymous is for unauthenticated visitors (Tier 1).
	TierAnonymous Tier = 1
	// TierRegistered is for authenticated users with no org membership (Tier 2).
	TierRegistered Tier = 2
	// TierCustomer is for members of a paying customer org (Tier 3).
	TierCustomer Tier = 3
	// TierDeftEmployee is for members of the DEFT org (Tier 4).
	TierDeftEmployee Tier = 4
	// TierCustomerAdmin is for admin/owner of a paying customer org (Tier 5).
	TierCustomerAdmin Tier = 5
	// TierPlatformAdmin is for platform administrators (Tier 6).
	TierPlatformAdmin Tier = 6
)

// String returns the human-readable name for the tier.
func (t Tier) String() string {
	switch t {
	case TierAnonymous:
		return "anonymous"
	case TierRegistered:
		return "registered"
	case TierCustomer:
		return "customer"
	case TierDeftEmployee:
		return "deft_employee"
	case TierCustomerAdmin:
		return "customer_admin"
	case TierPlatformAdmin:
		return "platform_admin"
	default:
		return "unknown"
	}
}

// IsValid returns true if the tier is a recognized value.
func (t Tier) IsValid() bool {
	return t >= TierAnonymous && t <= TierPlatformAdmin
}

// SubType describes additional context within a tier (e.g., department for Tier 4).
type SubType string

const (
	SubTypeNone        SubType = ""
	SubTypeOrgOwner    SubType = "owner"
	SubTypeDeftSales   SubType = "sales"
	SubTypeDeftSupport SubType = "support"
	SubTypeDeftFinance SubType = "finance"
)

// TierResult holds the resolved tier information for a user.
type TierResult struct {
	Tier           Tier    `json:"tier"`
	SubType        SubType `json:"sub_type,omitempty"`
	OrgID          string  `json:"org_id,omitempty"`
	DeftDepartment string  `json:"deft_department,omitempty"`
}
