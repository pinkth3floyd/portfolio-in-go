package public

import (
	"encoding/json"
	"html/template"
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

func (h *Handler) baseData(r *http.Request) web.PageData {
	settings, _ := h.Svc.Store.GetSettings(r.Context())
	links, _ := h.Svc.Store.ListFooterLinks(r.Context(), true)
	return web.PageData{
		CurrentPath: r.URL.Path,
		Settings:    settings,
		CSRFToken:   middleware.CSRFFrom(r),
		User:        middleware.UserFrom(r),
		ShowNav:     true,
		IsHTMX:      middleware.IsHTMX(r),
		BaseURL:     h.Svc.Config.BaseURL,
		Extra:       map[string]any{},
		FooterLinks: links,
	}
}

func crumbs(items ...models.Breadcrumb) []models.Breadcrumb {
	return items
}

func makePagination(page, perPage, total int) *models.Pagination {
	if perPage < 1 {
		perPage = 6
	}
	if page < 1 {
		page = 1
	}
	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	pages := make([]int, 0, totalPages)
	for i := 1; i <= totalPages; i++ {
		pages = append(pages, i)
	}
	return &models.Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		PrevPage:   page - 1,
		NextPage:   page + 1,
		Pages:      pages,
	}
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	data := h.baseData(r)
	page, _ := h.Svc.Store.GetPageBySlug(r.Context(), "home")
	projects, _ := h.Svc.Store.ListProjects(r.Context(), true)
	data.Page = page
	data.Projects = projects
	if page != nil {
		data.Title = page.Title
		data.Description = page.MetaDescription
		data.Keywords = page.MetaKeywords
		data.OGImage = page.OGImage
	}
	data.JSONLD = personJSONLD(data)
	h.R.Render(w, "pages/home.html", data, http.StatusOK)
}

func (h *Handler) About(w http.ResponseWriter, r *http.Request) {
	data := h.baseData(r)
	page, _ := h.Svc.Store.GetPageBySlug(r.Context(), "about")
	skills, _ := h.Svc.Store.ListSkills(r.Context())
	edu, _ := h.Svc.Store.ListEducation(r.Context())
	exp, _ := h.Svc.Store.ListExperiences(r.Context())
	data.Page = page
	data.Skills = skills
	data.Education = edu
	data.Experiences = exp
	data.Breadcrumbs = crumbs(
		models.Breadcrumb{Label: "Home", Href: "/"},
		models.Breadcrumb{Label: "About"},
	)
	cells := make([]int, 400)
	for i := range cells {
		cells[i] = (i * 37) % 360
	}
	data.Extra["GridCells"] = cells
	if page != nil {
		data.Title = page.Title
		data.Description = page.MetaDescription
		data.Keywords = page.MetaKeywords
	}
	data.JSONLD = mergeJSONLD(personJSONLD(data), breadcrumbJSONLD(data))
	h.R.Render(w, "pages/about.html", data, http.StatusOK)
}

func (h *Handler) Projects(w http.ResponseWriter, r *http.Request) {
	data := h.baseData(r)
	filter := r.URL.Query().Get("tag")
	if filter == "" {
		filter = "All"
	}
	page, _ := h.Svc.Store.GetPageBySlug(r.Context(), "projects")
	projects, _ := h.Svc.Store.ListProjectsByFilter(r.Context(), filter)
	data.Page = page
	data.Projects = projects
	data.ActiveFilter = filter
	data.Filters = []string{"All", "Web", "3D", "Design"}
	data.Breadcrumbs = crumbs(
		models.Breadcrumb{Label: "Home", Href: "/"},
		models.Breadcrumb{Label: "Projects"},
	)
	if page != nil {
		data.Title = page.Title
		data.Description = page.MetaDescription
		data.Keywords = page.MetaKeywords
	}
	if middleware.IsHTMX(r) {
		h.R.RenderPartial(w, "partials/project-grid.html", data, http.StatusOK)
		return
	}
	data.JSONLD = mergeJSONLD(personJSONLD(data), breadcrumbJSONLD(data), projectsItemListJSONLD(data))
	h.R.Render(w, "pages/projects.html", data, http.StatusOK)
}

func (h *Handler) ProjectShow(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	p, err := h.Svc.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil || p == nil || !p.Published {
		h.NotFound(w, r)
		return
	}
	data := h.baseData(r)
	data.Project = p
	data.Breadcrumbs = crumbs(
		models.Breadcrumb{Label: "Home", Href: "/"},
		models.Breadcrumb{Label: "Projects", Href: "/projects"},
		models.Breadcrumb{Label: p.Title},
	)
	data.Title = p.Title + " | Project by Prakash Niraula"
	data.Description = p.ShortDescription
	if len(data.Description) < 80 && p.FullDescription != "" {
		data.Description = truncate(p.FullDescription, 155)
	}
	data.Keywords = strings.Join(append([]string{p.Title, "Prakash Niraula", "portfolio project"}, p.TechStack...), ", ")
	data.OGImage = p.ImageURL
	data.JSONLD = mergeJSONLD(projectJSONLD(data, p), breadcrumbJSONLD(data))
	h.R.Render(w, "pages/project-show.html", data, http.StatusOK)
}

func (h *Handler) ProjectPartial(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	p, err := h.Svc.Store.GetProject(r.Context(), id)
	if err != nil || p == nil {
		http.NotFound(w, r)
		return
	}
	data := h.baseData(r)
	data.Project = p
	h.R.RenderPartial(w, "partials/project-modal.html", data, http.StatusOK)
}

func (h *Handler) Contact(w http.ResponseWriter, r *http.Request) {
	data := h.baseData(r)
	page, _ := h.Svc.Store.GetPageBySlug(r.Context(), "contact")
	data.Page = page
	data.Breadcrumbs = crumbs(
		models.Breadcrumb{Label: "Home", Href: "/"},
		models.Breadcrumb{Label: "Contact"},
	)
	if page != nil {
		data.Title = page.Title
		data.Description = page.MetaDescription
		data.Keywords = page.MetaKeywords
	}
	data.JSONLD = mergeJSONLD(contactJSONLD(data), breadcrumbJSONLD(data), faqJSONLD())
	h.R.Render(w, "pages/contact.html", data, http.StatusOK)
}

func (h *Handler) ContactSubmit(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	data := h.baseData(r)
	page, _ := h.Svc.Store.GetPageBySlug(r.Context(), "contact")
	data.Page = page
	data.Breadcrumbs = crumbs(
		models.Breadcrumb{Label: "Home", Href: "/"},
		models.Breadcrumb{Label: "Contact"},
	)
	if page != nil {
		data.Title = page.Title
		data.Description = page.MetaDescription
	}
	err := h.Svc.SubmitContact(r.Context(),
		r.FormValue("name"),
		r.FormValue("email"),
		r.FormValue("subject"),
		r.FormValue("message"),
	)
	if err != nil {
		data.FlashError = err.Error()
		if middleware.IsHTMX(r) {
			h.R.RenderPartial(w, "partials/contact-result.html", data, http.StatusOK)
			return
		}
		h.R.Render(w, "pages/contact.html", data, http.StatusBadRequest)
		return
	}
	data.Flash = "Message sent — I'll get back to you soon!"
	data.JSONLD = mergeJSONLD(contactJSONLD(data), breadcrumbJSONLD(data), faqJSONLD())
	if middleware.IsHTMX(r) {
		h.R.RenderPartial(w, "partials/contact-result.html", data, http.StatusOK)
		return
	}
	h.R.Render(w, "pages/contact.html", data, http.StatusOK)
}

func (h *Handler) Privacy(w http.ResponseWriter, r *http.Request) {
	data := h.baseData(r)
	page, _ := h.Svc.Store.GetPageBySlug(r.Context(), "privacy")
	sections, _ := h.Svc.Store.ListPrivacySections(r.Context())
	data.Page = page
	data.PrivacySections = sections
	data.Breadcrumbs = crumbs(
		models.Breadcrumb{Label: "Home", Href: "/"},
		models.Breadcrumb{Label: "Privacy"},
	)
	if page != nil {
		data.Title = page.Title
		data.Description = page.MetaDescription
		data.Keywords = page.MetaKeywords
	}
	data.JSONLD = breadcrumbJSONLD(data)
	h.R.Render(w, "pages/privacy.html", data, http.StatusOK)
}

func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	data := h.baseData(r)
	data.Title = "404 | Page Not Found"
	data.Description = "The page you requested could not be found."
	data.Breadcrumbs = crumbs(
		models.Breadcrumb{Label: "Home", Href: "/"},
		models.Breadcrumb{Label: "Not Found"},
	)
	h.R.Render(w, "pages/notfound.html", data, http.StatusNotFound)
}

func (h *Handler) Robots(w http.ResponseWriter, r *http.Request) {
	base := h.Svc.Config.BaseURL
	body := "User-agent: *\nAllow: /\nDisallow: /admin\nDisallow: /login\nSitemap: " + base + "/sitemap.xml\n"
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(body))
}

func (h *Handler) Sitemap(w http.ResponseWriter, r *http.Request) {
	base := h.Svc.Config.BaseURL
	now := time.Now().UTC().Format("2006-01-02")
	type smURL struct {
		Loc     string
		LastMod string
	}
	urls := []smURL{
		{base + "/", now},
		{base + "/about", now},
		{base + "/projects", now},
		{base + "/blog", now},
		{base + "/contact", now},
		{base + "/privacy", now},
	}
	projects, _ := h.Svc.Store.ListProjects(r.Context(), false)
	for _, p := range projects {
		urls = append(urls, smURL{base + "/projects/" + p.Slug, now})
	}
	blogs, _ := h.Svc.Store.ListBlogs(r.Context(), true)
	for _, blog := range blogs {
		lm := now
		if blog.PublishedAt != nil {
			lm = blog.PublishedAt.UTC().Format("2006-01-02")
		} else if !blog.UpdatedAt.IsZero() {
			lm = blog.UpdatedAt.UTC().Format("2006-01-02")
		}
		urls = append(urls, smURL{base + "/blog/" + blog.Slug, lm})
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, u := range urls {
		b.WriteString("<url><loc>")
		b.WriteString(u.Loc)
		b.WriteString("</loc><lastmod>")
		b.WriteString(u.LastMod)
		b.WriteString("</lastmod></url>")
	}
	b.WriteString("</urlset>")
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write([]byte(b.String()))
}

func (h *Handler) BlogIndex(w http.ResponseWriter, r *http.Request) {
	data := h.baseData(r)
	pageNum, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if pageNum < 1 {
		pageNum = 1
	}
	const perPage = 6
	blogs, total, _ := h.Svc.Store.ListBlogsPaged(r.Context(), true, pageNum, perPage)
	data.Blogs = blogs
	data.Pagination = makePagination(pageNum, perPage, total)
	data.Breadcrumbs = crumbs(
		models.Breadcrumb{Label: "Home", Href: "/"},
		models.Breadcrumb{Label: "Blog"},
	)
	data.Title = "Blog | Prakash Niraula — System Engineering, Web & SaaS"
	data.Description = "Articles by Prakash Niraula (System Engineer in Sakai, Japan) on infrastructure, web development, Flutter, Laravel, Next.js, and products like Web Converter Tools and GOLO CRM."
	data.Keywords = "prakash niraula blog, system engineer sakai, web converter tools, golo crm, ai kensetsu, seo tools, flutter, laravel"
	data.JSONLD = mergeJSONLD(personJSONLD(data), breadcrumbJSONLD(data), blogIndexJSONLD(data))
	h.R.Render(w, "pages/blog.html", data, http.StatusOK)
}

func (h *Handler) BlogShow(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	blog, err := h.Svc.Store.GetBlogBySlug(r.Context(), slug)
	if err != nil || blog == nil || !blog.Published {
		h.NotFound(w, r)
		return
	}
	_ = h.Svc.Store.IncrementBlogViews(r.Context(), blog.ID)
	blog.ViewCount++
	data := h.baseData(r)
	data.Blog = blog
	data.ShareURL = data.BaseURL + "/blog/" + blog.Slug
	data.Breadcrumbs = crumbs(
		models.Breadcrumb{Label: "Home", Href: "/"},
		models.Breadcrumb{Label: "Blog", Href: "/blog"},
		models.Breadcrumb{Label: blog.Title},
	)
	if blog.MetaTitle != "" {
		data.Title = blog.MetaTitle
	} else {
		data.Title = blog.Title + " | Prakash Niraula"
	}
	data.Description = blog.MetaDescription
	if data.Description == "" {
		data.Description = blog.Excerpt
	}
	data.Keywords = blog.MetaKeywords
	data.OGImage = blog.CoverImage
	data.OGType = "article"
	data.JSONLD = mergeJSONLD(blogJSONLD(data, blog), breadcrumbJSONLD(data))
	h.R.Render(w, "pages/blog-show.html", data, http.StatusOK)
}

func absoluteURL(base, path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(base, "/") + path
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n-1]) + "…"
}

func mergeJSONLD(parts ...template.JS) template.JS {
	var graph []any
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		var raw any
		if err := json.Unmarshal([]byte(part), &raw); err != nil {
			continue
		}
		switch v := raw.(type) {
		case map[string]any:
			if g, ok := v["@graph"].([]any); ok {
				graph = append(graph, g...)
				continue
			}
			delete(v, "@context")
			graph = append(graph, v)
		}
	}
	if len(graph) == 0 {
		return ""
	}
	b, _ := json.Marshal(map[string]any{
		"@context": "https://schema.org",
		"@graph":   graph,
	})
	return template.JS(b)
}

func breadcrumbJSONLD(data web.PageData) template.JS {
	if len(data.Breadcrumbs) == 0 {
		return ""
	}
	items := make([]any, 0, len(data.Breadcrumbs))
	pos := 1
	for _, c := range data.Breadcrumbs {
		item := map[string]any{
			"@type":    "ListItem",
			"position": pos,
			"name":     c.Label,
		}
		if c.Href != "" {
			item["item"] = absoluteURL(data.BaseURL, c.Href)
		} else {
			item["item"] = data.Canonical
		}
		items = append(items, item)
		pos++
	}
	b, _ := json.Marshal(map[string]any{
		"@type":           "BreadcrumbList",
		"itemListElement": items,
	})
	return template.JS(b)
}

func blogJSONLD(data web.PageData, blog *models.Blog) template.JS {
	desc := blog.MetaDescription
	if desc == "" {
		desc = blog.Excerpt
	}
	img := absoluteURL(data.BaseURL, blog.CoverImage)
	published := blog.CreatedAt.Format(time.RFC3339)
	if blog.PublishedAt != nil {
		published = blog.PublishedAt.Format(time.RFC3339)
	}
	modified := blog.UpdatedAt.Format(time.RFC3339)
	if blog.UpdatedAt.IsZero() {
		modified = published
	}
	obj := map[string]any{
		"@type":            "BlogPosting",
		"headline":         blog.Title,
		"description":      desc,
		"image":            filterEmpty([]string{img}),
		"url":              data.ShareURL,
		"datePublished":    published,
		"dateModified":     modified,
		"inLanguage":       "en",
		"mainEntityOfPage": data.ShareURL,
		"keywords":         blog.MetaKeywords,
		"author": map[string]any{
			"@type": "Person",
			"name":  data.Settings["site_name"],
			"url":   data.BaseURL,
		},
		"publisher": map[string]any{
			"@type": "Person",
			"name":  data.Settings["site_name"],
			"url":   data.BaseURL,
			"image": absoluteURL(data.BaseURL, data.Settings["og_image"]),
		},
	}
	b, _ := json.Marshal(obj)
	return template.JS(b)
}

func blogIndexJSONLD(data web.PageData) template.JS {
	b, _ := json.Marshal(map[string]any{
		"@type":       "Blog",
		"name":        "Prakash Niraula Blog",
		"description": data.Description,
		"url":         data.BaseURL + "/blog",
		"author": map[string]any{
			"@type": "Person",
			"name":  data.Settings["site_name"],
			"url":   data.BaseURL,
		},
	})
	return template.JS(b)
}

func projectJSONLD(data web.PageData, p *models.Project) template.JS {
	url := data.BaseURL + "/projects/" + p.Slug
	obj := map[string]any{
		"@type":       "SoftwareApplication",
		"name":        p.Title,
		"description": p.FullDescription,
		"url":         url,
		"image":       absoluteURL(data.BaseURL, p.ImageURL),
		"applicationCategory": "WebApplication",
		"author": map[string]any{
			"@type": "Person",
			"name":  data.Settings["site_name"],
			"url":   data.BaseURL,
		},
		"offers": map[string]any{
			"@type":         "Offer",
			"price":         "0",
			"priceCurrency": "USD",
			"url":           firstNonEmpty(p.GithubURL, p.LiveURL, url),
		},
	}
	if len(p.TechStack) > 0 {
		obj["keywords"] = strings.Join(p.TechStack, ", ")
	}
	b, _ := json.Marshal(obj)
	return template.JS(b)
}

func projectsItemListJSONLD(data web.PageData) template.JS {
	if len(data.Projects) == 0 {
		return ""
	}
	items := make([]any, 0, len(data.Projects))
	for i, p := range data.Projects {
		items = append(items, map[string]any{
			"@type":    "ListItem",
			"position": i + 1,
			"url":      data.BaseURL + "/projects/" + p.Slug,
			"name":     p.Title,
		})
	}
	b, _ := json.Marshal(map[string]any{
		"@type":           "ItemList",
		"name":            "Projects by Prakash Niraula",
		"itemListElement": items,
	})
	return template.JS(b)
}

func contactJSONLD(data web.PageData) template.JS {
	b, _ := json.Marshal(map[string]any{
		"@type": "ContactPage",
		"name":  "Contact Prakash Niraula",
		"url":   data.BaseURL + "/contact",
		"about": map[string]any{
			"@type": "Person",
			"name":  data.Settings["site_name"],
			"email": data.Settings["email"],
			"url":   data.BaseURL,
		},
	})
	return template.JS(b)
}

func faqJSONLD() template.JS {
	faqs := []struct{ Q, A string }{
		{"What services do you offer?", "System engineering and IT infrastructure, plus full-stack web apps (Laravel, Next.js, Go/HTMX), Flutter mobile apps, CRM/SaaS products, news portals, SEO tooling, APIs, and cloud deployment."},
		{"Where are you based?", "Otori, Sakai City, Osaka, Japan — System Engineer at AI Kensetsu Co., Ltd. Also open to international remote collaboration."},
		{"Can you build products like Web Converter Tools or GOLO CRM?", "Yes. Those products were designed and built by Prakash Niraula. Share your workflow and receive a proposed stack and milestones."},
		{"How do I start a project?", "Send a short brief with goals, timeline, and budget via the contact form or email pinkth3floyd@gmail.com."},
	}
	entities := make([]any, 0, len(faqs))
	for _, f := range faqs {
		entities = append(entities, map[string]any{
			"@type": "Question",
			"name":  f.Q,
			"acceptedAnswer": map[string]any{
				"@type": "Answer",
				"text":  f.A,
			},
		})
	}
	b, _ := json.Marshal(map[string]any{
		"@type":      "FAQPage",
		"mainEntity": entities,
	})
	return template.JS(b)
}

func personJSONLD(data web.PageData) template.JS {
	name := data.Settings["site_name"]
	if name == "" {
		name = "Prakash Niraula"
	}
	jobTitle := data.Settings["job_title"]
	if jobTitle == "" {
		jobTitle = "System Engineer"
	}
	company := data.Settings["company"]
	if company == "" {
		company = "AI Kensetsu Co., Ltd"
	}
	image := absoluteURL(data.BaseURL, data.Settings["og_image"])
	sameAs := filterEmpty([]string{
		data.Settings["github_url"],
		data.Settings["twitter_url"],
		data.Settings["linkedin_url"],
		"https://webconvertertools.com/",
		"https://golocrm.com",
		"https://raidmedia.net/",
	})
	person := map[string]any{
		"@type":       "Person",
		"@id":         data.BaseURL + "/#person",
		"name":        name,
		"url":         data.BaseURL,
		"email":       data.Settings["email"],
		"telephone":   data.Settings["phone"],
		"image":       image,
		"jobTitle":    jobTitle,
		"description": "System Engineer at AI Kensetsu Co., Ltd in Sakai, Osaka, Japan. Full-stack experience building Web Converter Tools, GOLO CRM, Raid Media, and IT infrastructure/software solutions.",
		"worksFor": map[string]any{
			"@type": "Organization",
			"name":  company,
		},
		"knowsAbout": []string{
			"System Engineering", "IT Infrastructure", "Software Development",
			"Web Development", "Mobile Development", "Laravel", "Next.js", "React", "Flutter",
			"Go", "HTMX", "Cloud", "Linux", "REST API",
		},
		"address": map[string]any{
			"@type":           "PostalAddress",
			"streetAddress":   data.Settings["address"],
			"addressLocality": "Sakai",
			"addressRegion":   "Osaka",
			"addressCountry":  "JP",
		},
		"sameAs": sameAs,
	}
	service := map[string]any{
		"@type":       "ProfessionalService",
		"@id":         data.BaseURL + "/#service",
		"name":        "Prakash Niraula — System Engineering & Software Development",
		"url":         data.BaseURL,
		"image":       image,
		"description": "System engineering, IT infrastructure, and full-stack web/mobile/SaaS development based in Sakai, Japan — available for international collaboration.",
		"telephone":   data.Settings["phone"],
		"email":       data.Settings["email"],
		"priceRange":  "$$",
		"areaServed": []any{
			map[string]any{"@type": "City", "name": "Sakai"},
			map[string]any{"@type": "AdministrativeArea", "name": "Osaka"},
			map[string]any{"@type": "Country", "name": "Japan"},
			"Worldwide",
		},
		"serviceType": []string{
			"System Engineering", "IT Infrastructure", "Web Development", "Mobile App Development", "SaaS Development",
		},
		"provider": map[string]any{"@id": data.BaseURL + "/#person"},
	}
	website := map[string]any{
		"@type":       "WebSite",
		"@id":         data.BaseURL + "/#website",
		"name":        name,
		"url":         data.BaseURL,
		"description": data.Description,
		"inLanguage":  "en",
		"publisher":   map[string]any{"@id": data.BaseURL + "/#person"},
	}
	b, _ := json.Marshal(map[string]any{
		"@context": "https://schema.org",
		"@graph":   []any{website, person, service},
	})
	return template.JS(b)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" && v != "#" {
			return v
		}
	}
	return ""
}

func filterEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}
