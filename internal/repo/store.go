package repo

import (
	"context"

	"github.com/prakashniraula/portfolio-in-go/internal/models"
)

// Store is the data access interface. SQLite implements it; Turso/libsql can later.
type Store interface {
	// Auth
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	CreateUser(ctx context.Context, username, passwordHash string) (int64, error)
	CountUsers(ctx context.Context) (int, error)
	CreateSession(ctx context.Context, userID int64, tokenHash string, expiresAt string) error
	GetSessionUser(ctx context.Context, tokenHash string) (*models.User, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	UpdatePassword(ctx context.Context, userID int64, passwordHash string) error

	// Settings
	GetSetting(ctx context.Context, key string) (string, error)
	GetSettings(ctx context.Context) (map[string]string, error)
	SetSetting(ctx context.Context, key, value string) error
	SetSettings(ctx context.Context, settings map[string]string) error

	// Pages
	GetPageBySlug(ctx context.Context, slug string) (*models.Page, error)
	ListPages(ctx context.Context) ([]models.Page, error)
	UpsertPage(ctx context.Context, page models.Page) error
	UpsertPageSection(ctx context.Context, pageID int64, section models.PageSection) error

	// Skills / Education / Experience
	ListSkills(ctx context.Context) ([]models.Skill, error)
	ReplaceSkills(ctx context.Context, skills []models.Skill) error
	ListEducation(ctx context.Context) ([]models.Education, error)
	CreateEducation(ctx context.Context, e models.Education) (int64, error)
	UpdateEducation(ctx context.Context, e models.Education) error
	DeleteEducation(ctx context.Context, id int64) error
	ListExperiences(ctx context.Context) ([]models.Experience, error)
	CreateExperience(ctx context.Context, e models.Experience) (int64, error)
	UpdateExperience(ctx context.Context, e models.Experience) error
	DeleteExperience(ctx context.Context, id int64) error
	ReplaceExperiences(ctx context.Context, items []models.Experience) error

	// Projects
	ListProjects(ctx context.Context, featuredOnly bool) ([]models.Project, error)
	ListAllProjects(ctx context.Context) ([]models.Project, error)
	ListProjectsByFilter(ctx context.Context, filter string) ([]models.Project, error)
	GetProject(ctx context.Context, id int64) (*models.Project, error)
	GetProjectBySlug(ctx context.Context, slug string) (*models.Project, error)
	CreateProject(ctx context.Context, p models.Project) (int64, error)
	UpdateProject(ctx context.Context, p models.Project) error
	DeleteProject(ctx context.Context, id int64) error
	CountProjects(ctx context.Context) (models.DashboardStats, error)

	// Privacy
	ListPrivacySections(ctx context.Context) ([]models.PrivacySection, error)
	CreatePrivacySection(ctx context.Context, s models.PrivacySection) (int64, error)
	UpdatePrivacySection(ctx context.Context, s models.PrivacySection) error
	DeletePrivacySection(ctx context.Context, id int64) error

	// Contact
	CreateContactMessage(ctx context.Context, m models.ContactMessage) (int64, error)
	ListContactMessages(ctx context.Context) ([]models.ContactMessage, error)
	MarkMessageNotified(ctx context.Context, id int64) error
	DeleteContactMessage(ctx context.Context, id int64) error
	CountMessages(ctx context.Context) (int, error)

	// Blogs
	ListBlogs(ctx context.Context, publishedOnly bool) ([]models.Blog, error)
	ListBlogsPaged(ctx context.Context, publishedOnly bool, page, perPage int) ([]models.Blog, int, error)
	GetBlog(ctx context.Context, id int64) (*models.Blog, error)
	GetBlogBySlug(ctx context.Context, slug string) (*models.Blog, error)
	CreateBlog(ctx context.Context, b models.Blog) (int64, error)
	UpdateBlog(ctx context.Context, b models.Blog) error
	DeleteBlog(ctx context.Context, id int64) error
	IncrementBlogViews(ctx context.Context, id int64) error
	CountBlogs(ctx context.Context) (int, error)

	// Footer links
	ListFooterLinks(ctx context.Context, enabledOnly bool) ([]models.FooterLink, error)
	GetFooterLink(ctx context.Context, id int64) (*models.FooterLink, error)
	CreateFooterLink(ctx context.Context, l models.FooterLink) (int64, error)
	UpdateFooterLink(ctx context.Context, l models.FooterLink) error
	DeleteFooterLink(ctx context.Context, id int64) error
}
