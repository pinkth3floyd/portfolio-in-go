package db

import (
	"context"
	"encoding/json"
	"os"

	"github.com/prakashniraula/portfolio-in-go/internal/models"
	"github.com/prakashniraula/portfolio-in-go/internal/repo"
)

// RefreshSEOFromFixtures updates page meta, settings, skills, experience,
// and featured project blurbs from seed.json even when the DB is populated.
func RefreshSEOFromFixtures(ctx context.Context, store repo.Store, fixturesPath string) error {
	raw, err := os.ReadFile(fixturesPath)
	if err != nil {
		return err
	}
	var seed seedFile
	if err := json.Unmarshal(raw, &seed); err != nil {
		return err
	}

	keys := []string{
		"linkedin_url", "tagline", "about_extra_1", "about_extra_2", "projects_intro",
		"location", "address", "og_image", "site_name", "site_url", "email", "phone",
		"github_url", "twitter_url", "twitter_handle", "job_title", "company", "hours",
	}
	patch := map[string]string{}
	for _, k := range keys {
		if v, ok := seed.Settings[k]; ok && v != "" {
			patch[k] = v
		}
	}
	if len(patch) > 0 {
		if err := store.SetSettings(ctx, patch); err != nil {
			return err
		}
	}

	for _, p := range seed.Pages {
		if err := store.UpsertPage(ctx, models.Page{
			Slug:            p.Slug,
			Title:           p.Title,
			MetaDescription: p.MetaDescription,
			MetaKeywords:    p.MetaKeywords,
		}); err != nil {
			return err
		}
		page, err := store.GetPageBySlug(ctx, p.Slug)
		if err != nil || page == nil {
			continue
		}
		for _, s := range p.Sections {
			if err := store.UpsertPageSection(ctx, page.ID, models.PageSection{
				SectionKey: s.Key, Title: s.Title, Subtitle: s.Subtitle, Body: s.Body,
			}); err != nil {
				return err
			}
		}
	}

	skills := make([]models.Skill, 0, len(seed.Skills))
	for i, label := range seed.Skills {
		skills = append(skills, models.Skill{Label: label, SortOrder: i})
	}
	if len(skills) > 0 {
		if err := store.ReplaceSkills(ctx, skills); err != nil {
			return err
		}
	}

	exps := make([]models.Experience, 0, len(seed.Experiences))
	for i, e := range seed.Experiences {
		exps = append(exps, models.Experience{
			Role: e.Role, Company: e.Company, Period: e.Period, Description: e.Description, SortOrder: i,
		})
	}
	if len(exps) > 0 {
		if err := store.ReplaceExperiences(ctx, exps); err != nil {
			return err
		}
	}

	for _, p := range seed.Projects {
		if !p.Featured {
			continue
		}
		existing, err := store.GetProjectBySlug(ctx, p.Slug)
		if err != nil || existing == nil {
			continue
		}
		existing.ShortDescription = p.ShortDescription
		existing.FullDescription = p.FullDescription
		if err := store.UpdateProject(ctx, *existing); err != nil {
			return err
		}
	}
	return nil
}
