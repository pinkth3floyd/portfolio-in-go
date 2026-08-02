package db

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/prakashniraula/portfolio-in-go/internal/models"
	"github.com/prakashniraula/portfolio-in-go/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

type seedFile struct {
	Settings         map[string]string `json:"settings"`
	Pages            []seedPage        `json:"pages"`
	Skills           []string          `json:"skills"`
	Education        []seedEducation   `json:"education"`
	Experiences      []seedExperience  `json:"experiences"`
	Projects         []seedProject     `json:"projects"`
	PrivacySections  []seedPrivacy     `json:"privacy_sections"`
}

type seedPage struct {
	Slug            string        `json:"slug"`
	Title           string        `json:"title"`
	MetaDescription string        `json:"meta_description"`
	MetaKeywords    string        `json:"meta_keywords"`
	Sections        []seedSection `json:"sections"`
}

type seedSection struct {
	Key      string `json:"key"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Body     string `json:"body"`
}

type seedEducation struct {
	Degree      string `json:"degree"`
	School      string `json:"school"`
	Year        string `json:"year"`
	Description string `json:"description"`
}

type seedExperience struct {
	Role        string `json:"role"`
	Company     string `json:"company"`
	Period      string `json:"period"`
	Description string `json:"description"`
}

type seedProject struct {
	Slug             string   `json:"slug"`
	Title            string   `json:"title"`
	ShortDescription string   `json:"short_description"`
	FullDescription  string   `json:"full_description"`
	ImageURL         string   `json:"image_url"`
	LiveURL          string   `json:"live_url"`
	GithubURL        string   `json:"github_url"`
	Featured         bool     `json:"featured"`
	Tags             []string `json:"tags"`
	Features         []string `json:"features"`
	TechStack        []string `json:"tech_stack"`
}

type seedPrivacy struct {
	Heading string `json:"heading"`
	Body    string `json:"body"`
}

// SeedIfEmpty loads fixtures when pages table is empty.
func SeedIfEmpty(ctx context.Context, store repo.Store, fixturesPath, adminUser, adminPassword string) error {
	pages, err := store.ListPages(ctx)
	if err != nil {
		return err
	}
	if len(pages) > 0 {
		if err := ensureAdmin(ctx, store, adminUser, adminPassword); err != nil {
			return err
		}
		if err := SeedBlogsIfEmpty(ctx, store); err != nil {
			return err
		}
		if err := SeedExtraBlogsIfMissing(ctx, store); err != nil {
			return err
		}
		if err := SeedFooterLinksIfEmpty(ctx, store); err != nil {
			return err
		}
		return RefreshSEOFromFixtures(ctx, store, fixturesPath)
	}
	data, err := os.ReadFile(fixturesPath)
	if err != nil {
		return fmt.Errorf("read fixtures: %w", err)
	}
	var seed seedFile
	if err := json.Unmarshal(data, &seed); err != nil {
		return fmt.Errorf("parse fixtures: %w", err)
	}
	if err := store.SetSettings(ctx, seed.Settings); err != nil {
		return err
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
			return fmt.Errorf("load page %s: %w", p.Slug, err)
		}
		for _, sec := range p.Sections {
			if err := store.UpsertPageSection(ctx, page.ID, models.PageSection{
				SectionKey: sec.Key,
				Title:      sec.Title,
				Subtitle:   sec.Subtitle,
				Body:       sec.Body,
			}); err != nil {
				return err
			}
		}
	}
	skills := make([]models.Skill, 0, len(seed.Skills))
	for i, label := range seed.Skills {
		skills = append(skills, models.Skill{Label: label, SortOrder: i})
	}
	if err := store.ReplaceSkills(ctx, skills); err != nil {
		return err
	}
	for i, e := range seed.Education {
		if _, err := store.CreateEducation(ctx, models.Education{
			Degree: e.Degree, School: e.School, Year: e.Year, Description: e.Description, SortOrder: i,
		}); err != nil {
			return err
		}
	}
	for i, e := range seed.Experiences {
		if _, err := store.CreateExperience(ctx, models.Experience{
			Role: e.Role, Company: e.Company, Period: e.Period, Description: e.Description, SortOrder: i,
		}); err != nil {
			return err
		}
	}
	for i, p := range seed.Projects {
		if _, err := store.CreateProject(ctx, models.Project{
			Slug:             p.Slug,
			Title:            p.Title,
			ShortDescription: p.ShortDescription,
			FullDescription:  p.FullDescription,
			ImageURL:         p.ImageURL,
			LiveURL:          p.LiveURL,
			GithubURL:        p.GithubURL,
			Featured:         p.Featured,
			Published:        true,
			SortOrder:        i,
			Tags:             p.Tags,
			Features:         p.Features,
			TechStack:        p.TechStack,
		}); err != nil {
			return err
		}
	}
	for i, sec := range seed.PrivacySections {
		if _, err := store.CreatePrivacySection(ctx, models.PrivacySection{
			Heading: sec.Heading, Body: sec.Body, SortOrder: i,
		}); err != nil {
			return err
		}
	}
	if err := ensureAdmin(ctx, store, adminUser, adminPassword); err != nil {
		return err
	}
	if err := SeedBlogsIfEmpty(ctx, store); err != nil {
		return err
	}
	if err := SeedExtraBlogsIfMissing(ctx, store); err != nil {
		return err
	}
	if err := SeedFooterLinksIfEmpty(ctx, store); err != nil {
		return err
	}
	return RefreshSEOFromFixtures(ctx, store, fixturesPath)
}

func ensureAdmin(ctx context.Context, store repo.Store, username, password string) error {
	n, err := store.CountUsers(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	username = strings.TrimSpace(username)
	if username == "" {
		username = "admin"
	}
	if password == "" {
		password = "changeme"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = store.CreateUser(ctx, username, string(hash))
	return err
}
