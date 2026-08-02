package httpx

import (
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prakashniraula/portfolio-in-go/internal/config"
	"github.com/prakashniraula/portfolio-in-go/internal/http/handlers/admin"
	"github.com/prakashniraula/portfolio-in-go/internal/http/handlers/public"
	"github.com/prakashniraula/portfolio-in-go/internal/http/middleware"
	"github.com/prakashniraula/portfolio-in-go/internal/service"
	"github.com/prakashniraula/portfolio-in-go/internal/web"
)

func NewRouter(cfg config.Config, svc *service.Services, renderer *web.Renderer) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.Session(svc, cfg))

	pub := &public.Handler{Svc: svc, R: renderer}
	adm := &admin.Handler{Svc: svc, R: renderer}

	fileServer := http.FileServer(http.Dir(cfg.StaticDir))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))
	r.Get("/favicon.ico", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, filepath.Join(cfg.StaticDir, "favicon.ico"))
	})

	r.Get("/robots.txt", pub.Robots)
	r.Get("/sitemap.xml", pub.Sitemap)

	r.Get("/", pub.Home)
	r.Get("/about", pub.About)
	r.Get("/projects", pub.Projects)
	r.Get("/projects/{slug}", pub.ProjectShow)
	r.Get("/partials/projects/{id}", pub.ProjectPartial)
	r.Get("/blog", pub.BlogIndex)
	r.Get("/blog/{slug}", pub.BlogShow)
	r.Get("/contact", pub.Contact)
	r.With(middleware.RequireCSRF).Post("/contact", pub.ContactSubmit)
	r.Get("/privacy", pub.Privacy)

	r.Get("/login", adm.LoginPage)
	r.With(middleware.RequireCSRF).Post("/login", adm.LoginSubmit)
	r.With(middleware.RequireCSRF).Post("/logout", adm.Logout)

	r.Route("/admin", func(ar chi.Router) {
		ar.Use(middleware.RequireAuth)
		ar.Use(middleware.RequireCSRF)
		ar.Get("/", adm.Dashboard)
		ar.Get("/pages", adm.Pages)
		ar.Get("/pages/{slug}", adm.Pages)
		ar.Post("/pages", adm.SavePage)
		ar.Get("/projects", adm.Projects)
		ar.Post("/projects", adm.SaveProject)
		ar.Post("/projects/{id}/delete", adm.DeleteProject)
		ar.Get("/experience", adm.Experience)
		ar.Post("/experience", adm.SaveExperience)
		ar.Post("/experience/{id}/delete", adm.DeleteExperience)
		ar.Post("/education", adm.SaveEducation)
		ar.Post("/education/{id}/delete", adm.DeleteEducation)
		ar.Post("/skills", adm.SaveSkills)
		ar.Get("/privacy", adm.Privacy)
		ar.Post("/privacy", adm.SavePrivacy)
		ar.Post("/privacy/{id}/delete", adm.DeletePrivacy)
		ar.Get("/settings", adm.Settings)
		ar.Post("/settings", adm.SaveSettings)
		ar.Get("/messages", adm.Messages)
		ar.Post("/messages/{id}/delete", adm.DeleteMessage)
		ar.Get("/blogs", adm.Blogs)
		ar.Post("/blogs", adm.SaveBlog)
		ar.Post("/blogs/{id}/delete", adm.DeleteBlog)
		ar.Get("/links", adm.Links)
		ar.Post("/links", adm.SaveLink)
		ar.Post("/links/{id}/delete", adm.DeleteLink)
		ar.Get("/account", adm.Account)
		ar.Post("/account/password", adm.ChangePassword)
	})

	r.NotFound(pub.NotFound)
	return r
}
