package models

import "time"

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

type Session struct {
	ID        int64
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type Page struct {
	ID              int64
	Slug            string
	Title           string
	MetaDescription string
	MetaKeywords    string
	OGImage         string
	UpdatedAt       time.Time
	Sections        map[string]PageSection
}

type PageSection struct {
	ID         int64
	PageID     int64
	SectionKey string
	Title      string
	Subtitle   string
	Body       string
}

type Skill struct {
	ID        int64
	Label     string
	SortOrder int
}

type Education struct {
	ID          int64
	Degree      string
	School      string
	Year        string
	Description string
	SortOrder   int
}

type Experience struct {
	ID          int64
	Role        string
	Company     string
	Period      string
	Description string
	SortOrder   int
}

type Project struct {
	ID               int64
	Slug             string
	Title            string
	ShortDescription string
	FullDescription  string
	ImageURL         string
	LiveURL          string
	GithubURL        string
	Featured         bool
	Published        bool
	SortOrder        int
	Tags             []string
	Features         []string
	TechStack        []string
}

type PrivacySection struct {
	ID        int64
	Heading   string
	Body      string
	SortOrder int
}

type ContactMessage struct {
	ID        int64
	Name      string
	Email     string
	Subject   string
	Message   string
	Notified  bool
	CreatedAt time.Time
}

type Blog struct {
	ID              int64
	Slug            string
	Title           string
	Excerpt         string
	Body            string // HTML
	CoverImage      string
	MetaTitle       string
	MetaDescription string
	MetaKeywords    string
	Published       bool
	ViewCount       int64
	PublishedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type FooterLink struct {
	ID        int64
	Label     string
	URL       string
	Icon      string // github | twitter | linkedin | link
	SortOrder int
	Enabled   bool
}

type Breadcrumb struct {
	Label string
	Href  string // empty = current page
}

type Pagination struct {
	Page       int
	PerPage    int
	Total      int
	TotalPages int
	HasPrev    bool
	HasNext    bool
	PrevPage   int
	NextPage   int
	Pages      []int
}

type DashboardStats struct {
	ProjectCount   int
	MessageCount   int
	UnreadCount    int
	PublishedCount int
	BlogCount      int
}
