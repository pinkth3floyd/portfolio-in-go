package db

import (
	"context"

	"github.com/prakashniraula/portfolio-in-go/internal/models"
	"github.com/prakashniraula/portfolio-in-go/internal/repo"
)

// SeedFooterLinksIfEmpty seeds GitHub/Twitter/LinkedIn from site settings when empty.
func SeedFooterLinksIfEmpty(ctx context.Context, store repo.Store) error {
	existing, err := store.ListFooterLinks(ctx, false)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}
	settings, _ := store.GetSettings(ctx)
	defaults := []models.FooterLink{
		{Label: "GitHub", URL: settings["github_url"], Icon: "github", SortOrder: 0, Enabled: true},
		{Label: "Twitter", URL: settings["twitter_url"], Icon: "twitter", SortOrder: 1, Enabled: true},
		{Label: "LinkedIn", URL: settings["linkedin_url"], Icon: "linkedin", SortOrder: 2, Enabled: true},
	}
	for _, l := range defaults {
		if l.URL == "" {
			switch l.Icon {
			case "github":
				l.URL = "https://github.com/PrakashNiraula"
			case "twitter":
				l.URL = "https://twitter.com/pinkth3floyd"
			case "linkedin":
				l.URL = "https://linkedin.com"
			}
		}
		if _, err := store.CreateFooterLink(ctx, l); err != nil {
			return err
		}
	}
	return nil
}
