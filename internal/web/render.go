package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/prakashniraula/portfolio-in-go/internal/config"
	"github.com/prakashniraula/portfolio-in-go/internal/models"
	"github.com/prakashniraula/portfolio-in-go/internal/service"
)

type Renderer struct {
	mu       sync.RWMutex
	tmpl     *template.Template
	dir      string
	baseURL  string
	services *service.Services
}

func NewRenderer(dir, baseURL string, svc *service.Services) (*Renderer, error) {
	r := &Renderer{dir: dir, baseURL: baseURL, services: svc}
	if err := r.Reload(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Renderer) Reload() error {
	funcs := template.FuncMap{
		"year": func() int { return time.Now().Year() },
		"section": func(p *models.Page, key string) models.PageSection {
			if p == nil || p.Sections == nil {
				return models.PageSection{}
			}
			return p.Sections[key]
		},
		"setting": func(m map[string]string, key string) string {
			if m == nil {
				return ""
			}
			return m[key]
		},
		"eq":        func(a, b any) bool { return a == b },
		"join":      strings.Join,
		"hasPrefix": strings.HasPrefix,
		"rawHTML":   func(s string) template.HTML { return template.HTML(s) },
		"safeHTML": func(s string) template.HTML {
			lines := strings.Split(s, "\n")
			var b strings.Builder
			inList := false
			flushP := func(p string) {
				p = strings.TrimSpace(p)
				if p == "" {
					return
				}
				b.WriteString("<p class=\"text-cyber-lavender/90 mb-4\">")
				b.WriteString(template.HTMLEscapeString(p))
				b.WriteString("</p>")
			}
			var para strings.Builder
			closeList := func() {
				if inList {
					b.WriteString("</ul>")
					inList = false
				}
			}
			for _, line := range lines {
				trim := strings.TrimSpace(line)
				if strings.HasPrefix(trim, "- ") {
					flushP(para.String())
					para.Reset()
					if !inList {
						b.WriteString(`<ul class="list-disc pl-6 space-y-2 text-cyber-lavender/90 mb-4">`)
						inList = true
					}
					b.WriteString("<li>")
					b.WriteString(template.HTMLEscapeString(strings.TrimPrefix(trim, "- ")))
					b.WriteString("</li>")
					continue
				}
				if trim == "" {
					closeList()
					flushP(para.String())
					para.Reset()
					continue
				}
				closeList()
				if para.Len() > 0 {
					para.WriteByte(' ')
				}
				para.WriteString(trim)
			}
			closeList()
			flushP(para.String())
			return template.HTML(b.String())
		},
		"json": func(v any) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"csrfField": func(token string) template.HTML {
			return template.HTML(fmt.Sprintf(`<input type="hidden" name="csrf_token" value="%s">`, template.HTMLEscapeString(token)))
		},
		"activeNav": func(current, path string) string {
			if current == path {
				return "text-cyber-cyan after:w-full"
			}
			return "text-cyber-text"
		},
		"adminActive": func(current, prefix string) bool {
			if prefix == "/admin" {
				return current == "/admin" || current == "/admin/"
			}
			return current == prefix || strings.HasPrefix(current, prefix+"/") || strings.HasPrefix(current, prefix+"?")
		},
		"isLast": func(i, n int) bool { return i == n-1 },
		"add":    func(a, b int) int { return a + b },
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict requires even number of args")
			}
			m := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				m[key] = values[i+1]
			}
			return m, nil
		},
		"printf": fmt.Sprintf,
		"urlquery": func(s string) string {
			return template.URLQueryEscaper(s)
		},
		"formatDate": func(t *time.Time) string {
			if t == nil {
				return ""
			}
			return t.Format("January 2, 2006")
		},
		"formatDateTime": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("Jan 2, 2006")
		},
		"seq": func(n int) []int {
			if n < 1 {
				return nil
			}
			out := make([]int, n)
			for i := 0; i < n; i++ {
				out[i] = i + 1
			}
			return out
		},
	}
	root := template.New("root").Funcs(funcs)
	err := filepath.WalkDir(r.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		rel, err := filepath.Rel(r.dir, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = root.New(name).Parse(string(b))
		return err
	})
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.tmpl = root
	r.mu.Unlock()
	return nil
}

type PageData struct {
	Title           string
	Description     string
	Keywords        string
	Canonical       string
	OGImage         string
	OGType          string
	CurrentPath     string
	BaseURL         string
	Settings        map[string]string
	Page            *models.Page
	Projects        []models.Project
	Project         *models.Project
	Skills          []models.Skill
	Education       []models.Education
	Experiences     []models.Experience
	PrivacySections []models.PrivacySection
	ActiveFilter    string
	Filters         []string
	CSRFToken       string
	Flash           string
	FlashError      string
	User            *models.User
	Year            int
	JSONLD          template.JS
	ShowNav         bool
	IsHTMX          bool
	Extra           map[string]any
	Stats           models.DashboardStats
	Messages        []models.ContactMessage
	Pages           []models.Page
	PrivacyEdit     *models.PrivacySection
	Blogs           []models.Blog
	Blog            *models.Blog
	ShareURL        string
	Breadcrumbs     []models.Breadcrumb
	Pagination      *models.Pagination
	FooterLinks     []models.FooterLink
	FooterLink      *models.FooterLink
	GAID            string
	AhrefsKey       string
}

func (r *Renderer) Render(w http.ResponseWriter, name string, data PageData, status int) {
	r.mu.RLock()
	tmpl := r.tmpl
	r.mu.RUnlock()
	if data.BaseURL == "" {
		data.BaseURL = r.baseURL
	}
	if data.Year == 0 {
		data.Year = time.Now().Year()
	}
	if data.OGType == "" {
		data.OGType = "website"
	}
	if !strings.HasPrefix(data.CurrentPath, "/admin") && data.CurrentPath != "/login" {
		data.ShowNav = true
	} else {
		data.ShowNav = false
	}
	if data.Settings != nil {
		data.GAID = data.Settings["ga_id"]
		data.AhrefsKey = data.Settings["ahrefs_key"]
		if data.OGImage == "" {
			data.OGImage = data.Settings["og_image"]
		}
	}
	if data.Canonical == "" && data.CurrentPath != "" {
		data.Canonical = data.BaseURL + data.CurrentPath
		if data.CurrentPath == "/" {
			data.Canonical = data.BaseURL + "/"
		}
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func (r *Renderer) RenderPartial(w http.ResponseWriter, name string, data PageData, status int) {
	r.Render(w, name, data, status)
}

func LoadConfigTemplates(cfg config.Config, svc *service.Services) (*Renderer, error) {
	return NewRenderer(cfg.TemplatesDir, cfg.BaseURL, svc)
}
