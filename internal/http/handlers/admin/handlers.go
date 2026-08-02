package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prakashniraula/portfolio-in-go/internal/http/middleware"
	"github.com/prakashniraula/portfolio-in-go/internal/models"
	"github.com/prakashniraula/portfolio-in-go/internal/service"
	"github.com/prakashniraula/portfolio-in-go/internal/web"
)

type Handler struct {
	Svc *service.Services
	R   *web.Renderer
}

func (h *Handler) base(r *http.Request) web.PageData {
	settings, _ := h.Svc.Store.GetSettings(r.Context())
	return web.PageData{
		CurrentPath: r.URL.Path,
		Settings:    settings,
		CSRFToken:   middleware.CSRFFrom(r),
		User:        middleware.UserFrom(r),
		ShowNav:     false,
		BaseURL:     h.Svc.Config.BaseURL,
		Extra:       map[string]any{},
	}
}

func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if middleware.UserFrom(r) != nil {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	data := h.base(r)
	data.Title = "Admin Login | Prakash Niraula"
	data.Description = "Admin login"
	h.R.Render(w, "pages/login.html", data, http.StatusOK)
}

func (h *Handler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	token, err := h.Svc.Login(r.Context(), r.FormValue("username"), r.FormValue("password"))
	if err != nil {
		data := h.base(r)
		data.Title = "Admin Login | Prakash Niraula"
		data.FlashError = "Invalid username or password"
		h.R.Render(w, "pages/login.html", data, http.StatusUnauthorized)
		return
	}
	middleware.SetSessionCookie(w, token, h.Svc.Config)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	_ = h.Svc.Logout(r.Context(), middleware.SessionTokenFrom(r))
	middleware.ClearSessionCookie(w, h.Svc.Config)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	data := h.base(r)
	stats, _ := h.Svc.Store.CountProjects(r.Context())
	data.Stats = stats
	data.Title = "Admin Dashboard"
	h.R.Render(w, "admin/dashboard.html", data, http.StatusOK)
}

func (h *Handler) Pages(w http.ResponseWriter, r *http.Request) {
	data := h.base(r)
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		slug = r.URL.Query().Get("slug")
	}
	if slug == "" {
		slug = "home"
	}
	pages, _ := h.Svc.Store.ListPages(r.Context())
	page, _ := h.Svc.Store.GetPageBySlug(r.Context(), slug)
	data.Pages = pages
	data.Page = page
	data.Extra["ActiveSlug"] = slug
	data.Title = "Edit Pages"
	h.R.Render(w, "admin/pages.html", data, http.StatusOK)
}

func (h *Handler) SavePage(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	slug := r.FormValue("slug")
	page := models.Page{
		Slug:            slug,
		Title:           r.FormValue("title"),
		MetaDescription: r.FormValue("meta_description"),
		MetaKeywords:    r.FormValue("meta_keywords"),
		OGImage:         r.FormValue("og_image"),
	}
	_ = h.Svc.Store.UpsertPage(r.Context(), page)
	existing, _ := h.Svc.Store.GetPageBySlug(r.Context(), slug)
	if existing != nil {
		keys := []string{"hero", "featured", "intro", "skills"}
		for _, key := range keys {
			title := r.FormValue("section_" + key + "_title")
			subtitle := r.FormValue("section_" + key + "_subtitle")
			body := r.FormValue("section_" + key + "_body")
			if title == "" && subtitle == "" && body == "" && existing.Sections[key].ID == 0 {
				continue
			}
			if title == "" && subtitle == "" && body == "" {
				continue
			}
			_ = h.Svc.Store.UpsertPageSection(r.Context(), existing.ID, models.PageSection{
				SectionKey: key,
				Title:      title,
				Subtitle:   subtitle,
				Body:       body,
			})
		}
		// Always save posted section fields even if empty for known sections on page
		for key := range existing.Sections {
			_ = h.Svc.Store.UpsertPageSection(r.Context(), existing.ID, models.PageSection{
				SectionKey: key,
				Title:      r.FormValue("section_" + key + "_title"),
				Subtitle:   r.FormValue("section_" + key + "_subtitle"),
				Body:       r.FormValue("section_" + key + "_body"),
			})
		}
	}
	http.Redirect(w, r, "/admin/pages?slug="+slug+"&saved=1", http.StatusSeeOther)
}

func (h *Handler) Projects(w http.ResponseWriter, r *http.Request) {
	data := h.base(r)
	projects, _ := h.Svc.Store.ListAllProjects(r.Context())
	data.Projects = projects
	data.Title = "Manage Projects"
	if idStr := r.URL.Query().Get("edit"); idStr != "" {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		p, _ := h.Svc.Store.GetProject(r.Context(), id)
		data.Project = p
	}
	h.R.Render(w, "admin/projects.html", data, http.StatusOK)
}

func (h *Handler) SaveProject(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	slug := r.FormValue("slug")
	if slug == "" {
		slug = service.Slugify(r.FormValue("title"))
	}
	sortOrder, _ := strconv.Atoi(r.FormValue("sort_order"))
	p := models.Project{
		ID:               id,
		Slug:             slug,
		Title:            r.FormValue("title"),
		ShortDescription: r.FormValue("short_description"),
		FullDescription:  r.FormValue("full_description"),
		ImageURL:         r.FormValue("image_url"),
		LiveURL:          r.FormValue("live_url"),
		GithubURL:        r.FormValue("github_url"),
		Featured:         r.FormValue("featured") == "on" || r.FormValue("featured") == "1",
		Published:        r.FormValue("published") == "on" || r.FormValue("published") == "1",
		SortOrder:        sortOrder,
		Tags:             service.SplitCSV(r.FormValue("tags")),
		Features:         splitLines(r.FormValue("features")),
		TechStack:        service.SplitCSV(r.FormValue("tech_stack")),
	}
	if id == 0 && r.FormValue("published") == "" && r.FormValue("id") == "" {
		// new form defaults published checkbox checked in UI
	}
	if id > 0 {
		_ = h.Svc.Store.UpdateProject(r.Context(), p)
	} else {
		_, _ = h.Svc.Store.CreateProject(r.Context(), p)
	}
	http.Redirect(w, r, "/admin/projects?saved=1", http.StatusSeeOther)
}

func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_ = h.Svc.Store.DeleteProject(r.Context(), id)
	http.Redirect(w, r, "/admin/projects", http.StatusSeeOther)
}

func (h *Handler) Experience(w http.ResponseWriter, r *http.Request) {
	data := h.base(r)
	exp, _ := h.Svc.Store.ListExperiences(r.Context())
	edu, _ := h.Svc.Store.ListEducation(r.Context())
	skills, _ := h.Svc.Store.ListSkills(r.Context())
	data.Experiences = exp
	data.Education = edu
	data.Skills = skills
	data.Title = "Experience & Skills"
	h.R.Render(w, "admin/experience.html", data, http.StatusOK)
}

func (h *Handler) SaveExperience(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	sortOrder, _ := strconv.Atoi(r.FormValue("sort_order"))
	e := models.Experience{
		ID: id, Role: r.FormValue("role"), Company: r.FormValue("company"),
		Period: r.FormValue("period"), Description: r.FormValue("description"), SortOrder: sortOrder,
	}
	if id > 0 {
		_ = h.Svc.Store.UpdateExperience(r.Context(), e)
	} else {
		_, _ = h.Svc.Store.CreateExperience(r.Context(), e)
	}
	http.Redirect(w, r, "/admin/experience?saved=1", http.StatusSeeOther)
}

func (h *Handler) DeleteExperience(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_ = h.Svc.Store.DeleteExperience(r.Context(), id)
	http.Redirect(w, r, "/admin/experience", http.StatusSeeOther)
}

func (h *Handler) SaveEducation(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	sortOrder, _ := strconv.Atoi(r.FormValue("sort_order"))
	e := models.Education{
		ID: id, Degree: r.FormValue("degree"), School: r.FormValue("school"),
		Year: r.FormValue("year"), Description: r.FormValue("description"), SortOrder: sortOrder,
	}
	if id > 0 {
		_ = h.Svc.Store.UpdateEducation(r.Context(), e)
	} else {
		_, _ = h.Svc.Store.CreateEducation(r.Context(), e)
	}
	http.Redirect(w, r, "/admin/experience?saved=1", http.StatusSeeOther)
}

func (h *Handler) DeleteEducation(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_ = h.Svc.Store.DeleteEducation(r.Context(), id)
	http.Redirect(w, r, "/admin/experience", http.StatusSeeOther)
}

func (h *Handler) SaveSkills(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	labels := service.SplitCSV(r.FormValue("skills"))
	skills := make([]models.Skill, 0, len(labels))
	for i, l := range labels {
		skills = append(skills, models.Skill{Label: l, SortOrder: i})
	}
	_ = h.Svc.Store.ReplaceSkills(r.Context(), skills)
	http.Redirect(w, r, "/admin/experience?saved=1", http.StatusSeeOther)
}

func (h *Handler) Privacy(w http.ResponseWriter, r *http.Request) {
	data := h.base(r)
	sections, _ := h.Svc.Store.ListPrivacySections(r.Context())
	data.PrivacySections = sections
	data.Title = "Privacy Sections"
	if idStr := r.URL.Query().Get("edit"); idStr != "" {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		for _, s := range sections {
			if s.ID == id {
				sec := s
				data.PrivacyEdit = &sec
				break
			}
		}
	}
	h.R.Render(w, "admin/privacy.html", data, http.StatusOK)
}

func (h *Handler) SavePrivacy(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	sortOrder, _ := strconv.Atoi(r.FormValue("sort_order"))
	sec := models.PrivacySection{ID: id, Heading: r.FormValue("heading"), Body: r.FormValue("body"), SortOrder: sortOrder}
	if id > 0 {
		_ = h.Svc.Store.UpdatePrivacySection(r.Context(), sec)
	} else {
		_, _ = h.Svc.Store.CreatePrivacySection(r.Context(), sec)
	}
	http.Redirect(w, r, "/admin/privacy?saved=1", http.StatusSeeOther)
}

func (h *Handler) DeletePrivacy(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_ = h.Svc.Store.DeletePrivacySection(r.Context(), id)
	http.Redirect(w, r, "/admin/privacy", http.StatusSeeOther)
}

func (h *Handler) Settings(w http.ResponseWriter, r *http.Request) {
	data := h.base(r)
	data.Title = "Site Settings"
	h.R.Render(w, "admin/settings.html", data, http.StatusOK)
}

func (h *Handler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	keys := []string{
		"site_name", "site_url", "email", "phone", "location", "hours", "address",
		"github_url", "twitter_url", "linkedin_url", "twitter_handle",
		"ga_id", "ahrefs_key", "tagline", "og_image",
		"about_extra_1", "about_extra_2", "projects_intro",
		"job_title", "company",
	}
	m := map[string]string{}
	for _, k := range keys {
		m[k] = r.FormValue(k)
	}
	_ = h.Svc.Store.SetSettings(r.Context(), m)
	http.Redirect(w, r, "/admin/settings?saved=1", http.StatusSeeOther)
}

func (h *Handler) Messages(w http.ResponseWriter, r *http.Request) {
	data := h.base(r)
	msgs, _ := h.Svc.Store.ListContactMessages(r.Context())
	data.Messages = msgs
	data.Title = "Contact Messages"
	h.R.Render(w, "admin/messages.html", data, http.StatusOK)
}

func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_ = h.Svc.Store.DeleteContactMessage(r.Context(), id)
	http.Redirect(w, r, "/admin/messages", http.StatusSeeOther)
}

func (h *Handler) Account(w http.ResponseWriter, r *http.Request) {
	data := h.base(r)
	data.Title = "Account"
	h.R.Render(w, "admin/account.html", data, http.StatusOK)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	data := h.base(r)
	data.Title = "Account"
	user := middleware.UserFrom(r)
	err := h.Svc.ChangePasswordWithUser(r.Context(), user, r.FormValue("current_password"), r.FormValue("new_password"))
	if err != nil {
		data.FlashError = err.Error()
		h.R.Render(w, "admin/account.html", data, http.StatusBadRequest)
		return
	}
	data.Flash = "Password updated"
	h.R.Render(w, "admin/account.html", data, http.StatusOK)
}

func (h *Handler) Blogs(w http.ResponseWriter, r *http.Request) {
	data := h.base(r)
	blogs, _ := h.Svc.Store.ListBlogs(r.Context(), false)
	data.Blogs = blogs
	data.Title = "Manage Blogs"
	if idStr := r.URL.Query().Get("edit"); idStr != "" {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		b, _ := h.Svc.Store.GetBlog(r.Context(), id)
		data.Blog = b
	}
	h.R.Render(w, "admin/blogs.html", data, http.StatusOK)
}

func (h *Handler) SaveBlog(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	slug := r.FormValue("slug")
	if slug == "" {
		slug = service.Slugify(r.FormValue("title"))
	}
	published := r.FormValue("published") == "1" || r.FormValue("published") == "on"
	b := models.Blog{
		ID:              id,
		Slug:            slug,
		Title:           r.FormValue("title"),
		Excerpt:         r.FormValue("excerpt"),
		Body:            r.FormValue("body"),
		CoverImage:      r.FormValue("cover_image"),
		MetaTitle:       r.FormValue("meta_title"),
		MetaDescription: r.FormValue("meta_description"),
		MetaKeywords:    r.FormValue("meta_keywords"),
		Published:       published,
	}
	if published {
		now := time.Now().UTC()
		b.PublishedAt = &now
	}
	if id > 0 {
		existing, _ := h.Svc.Store.GetBlog(r.Context(), id)
		if existing != nil && existing.PublishedAt != nil {
			b.PublishedAt = existing.PublishedAt
		}
		_ = h.Svc.Store.UpdateBlog(r.Context(), b)
	} else {
		_, _ = h.Svc.Store.CreateBlog(r.Context(), b)
	}
	http.Redirect(w, r, "/admin/blogs?saved=1", http.StatusSeeOther)
}

func (h *Handler) DeleteBlog(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_ = h.Svc.Store.DeleteBlog(r.Context(), id)
	http.Redirect(w, r, "/admin/blogs", http.StatusSeeOther)
}

func (h *Handler) Links(w http.ResponseWriter, r *http.Request) {
	data := h.base(r)
	links, _ := h.Svc.Store.ListFooterLinks(r.Context(), false)
	data.FooterLinks = links
	data.Title = "Footer Links"
	if idStr := r.URL.Query().Get("edit"); idStr != "" {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		l, _ := h.Svc.Store.GetFooterLink(r.Context(), id)
		data.FooterLink = l
	}
	h.R.Render(w, "admin/links.html", data, http.StatusOK)
}

func (h *Handler) SaveLink(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	sortOrder, _ := strconv.Atoi(r.FormValue("sort_order"))
	l := models.FooterLink{
		ID:        id,
		Label:     r.FormValue("label"),
		URL:       r.FormValue("url"),
		Icon:      r.FormValue("icon"),
		SortOrder: sortOrder,
		Enabled:   r.FormValue("enabled") == "1" || r.FormValue("enabled") == "on",
	}
	if l.Icon == "" {
		l.Icon = "link"
	}
	if id > 0 {
		_ = h.Svc.Store.UpdateFooterLink(r.Context(), l)
	} else {
		_, _ = h.Svc.Store.CreateFooterLink(r.Context(), l)
	}
	http.Redirect(w, r, "/admin/links?saved=1", http.StatusSeeOther)
}

func (h *Handler) DeleteLink(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_ = h.Svc.Store.DeleteFooterLink(r.Context(), id)
	http.Redirect(w, r, "/admin/links", http.StatusSeeOther)
}

func splitLines(s string) []string {
	parts := strings.Split(s, "\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
