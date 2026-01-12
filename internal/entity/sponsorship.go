package entity

import (
	"time"

	"github.com/google/uuid"
)

type SponsorshipTier string

const (
	TierFeatured SponsorshipTier = "featured"
	TierPromoted SponsorshipTier = "promoted"
	TierBoosted  SponsorshipTier = "boosted"
)

type CampaignStatus string

const (
	CampaignActive    CampaignStatus = "active"
	CampaignPaused    CampaignStatus = "paused"
	CampaignCompleted CampaignStatus = "completed"
	CampaignCancelled CampaignStatus = "cancelled"
)

type PlacementType string

const (
	PlacementHomepageHero PlacementType = "homepage_hero"
	PlacementSearchTop    PlacementType = "search_top"
	PlacementSearchResult PlacementType = "search_results"
	PlacementSidebar      PlacementType = "sidebar"
)

type SponsorshipCampaign struct {
	ID            uuid.UUID      `json:"id" db:"id"`
	ServerID      uuid.UUID      `json:"server_id" db:"server_id"`
	SponsorUserID uuid.UUID      `json:"sponsor_user_id" db:"sponsor_user_id"`
	Tier          string         `json:"tier" db:"tier"`
	StartDate     time.Time      `json:"start_date" db:"start_date"`
	EndDate       time.Time      `json:"end_date" db:"end_date"`
	BudgetCents   int64          `json:"budget_cents" db:"budget_cents"`
	SpentCents    int64          `json:"spent_cents" db:"spent_cents"`
	Impressions   int64          `json:"impressions" db:"impressions"`
	Clicks        int64          `json:"clicks" db:"clicks"`
	Installs      int64          `json:"installs" db:"installs"`
	Status        CampaignStatus `json:"status" db:"status"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at" db:"updated_at"`
}

type SponsorshipImpression struct {
	ID         uuid.UUID     `json:"id" db:"id"`
	CampaignID uuid.UUID     `json:"campaign_id" db:"campaign_id"`
	UserID     *uuid.UUID    `json:"user_id,omitempty" db:"user_id"`
	Placement  PlacementType `json:"placement" db:"placement"`
	Clicked    bool          `json:"clicked" db:"clicked"`
	Installed  bool          `json:"installed" db:"installed"`
	Timestamp  time.Time     `json:"timestamp" db:"timestamp"`
}

type SponsorshipPricing struct {
	Tier                    string    `json:"tier" db:"tier"`
	Name                    string    `json:"name" db:"name"`
	Description             string    `json:"description" db:"description"`
	PricePerMonthCents      int64     `json:"price_per_month_cents" db:"price_per_month_cents"`
	PricePerImpressionCents *int      `json:"price_per_impression_cents,omitempty" db:"price_per_impression_cents"`
	MaxPriority             int       `json:"max_priority" db:"max_priority"`
	Placements              []string  `json:"placements" db:"placements"`
	CreatedAt               time.Time `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time `json:"updated_at" db:"updated_at"`
}

type CampaignAnalytics struct {
	CampaignID      uuid.UUID `json:"campaign_id"`
	Impressions     int64     `json:"impressions"`
	Clicks          int64     `json:"clicks"`
	Installs        int64     `json:"installs"`
	CTR             float64   `json:"ctr"`
	ConversionRate  float64   `json:"conversion_rate"`
	SpentCents      int64     `json:"spent_cents"`
	BudgetCents     int64     `json:"budget_cents"`
	BudgetRemaining int64     `json:"budget_remaining"`
	CostPerClick    float64   `json:"cost_per_click"`
	CostPerInstall  float64   `json:"cost_per_install"`
	DaysRemaining   int       `json:"days_remaining"`
	EstimatedReach  int64     `json:"estimated_reach"`
}
