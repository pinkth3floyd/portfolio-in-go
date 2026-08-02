package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/prakashniraula/portfolio-in-go/internal/models"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	u := &models.User{}
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT id, username, password_hash, created_at FROM users WHERE username = ?`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
	return u, nil
}

func (s *SQLiteStore) CreateUser(ctx context.Context, username, passwordHash string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO users (username, password_hash) VALUES (?, ?)`, username, passwordHash)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&n)
	return n, err
}

func (s *SQLiteStore) CreateSession(ctx context.Context, userID int64, tokenHash string, expiresAt string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions (user_id, token_hash, expires_at) VALUES (?, ?, ?)`, userID, tokenHash, expiresAt)
	return err
}

func (s *SQLiteStore) GetSessionUser(ctx context.Context, tokenHash string) (*models.User, error) {
	u := &models.User{}
	var created string
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.password_hash, u.created_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > datetime('now')
	`, tokenHash).Scan(&u.ID, &u.Username, &u.PasswordHash, &created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
	return u, nil
}

func (s *SQLiteStore) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}

func (s *SQLiteStore) UpdatePassword(ctx context.Context, userID int64, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, userID)
	return err
}

func (s *SQLiteStore) GetSetting(ctx context.Context, key string) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM site_settings WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

func (s *SQLiteStore) GetSettings(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM site_settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

func (s *SQLiteStore) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO site_settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func (s *SQLiteStore) SetSettings(ctx context.Context, settings map[string]string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO site_settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for k, v := range settings {
		if _, err := stmt.ExecContext(ctx, k, v); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) GetPageBySlug(ctx context.Context, slug string) (*models.Page, error) {
	p := &models.Page{Sections: map[string]models.PageSection{}}
	var updated string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, slug, title, meta_description, meta_keywords, og_image, updated_at
		FROM pages WHERE slug = ?`, slug).
		Scan(&p.ID, &p.Slug, &p.Title, &p.MetaDescription, &p.MetaKeywords, &p.OGImage, &updated)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updated)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, page_id, section_key, title, subtitle, body FROM page_sections WHERE page_id = ?`, p.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var sec models.PageSection
		if err := rows.Scan(&sec.ID, &sec.PageID, &sec.SectionKey, &sec.Title, &sec.Subtitle, &sec.Body); err != nil {
			return nil, err
		}
		p.Sections[sec.SectionKey] = sec
	}
	return p, rows.Err()
}

func (s *SQLiteStore) ListPages(ctx context.Context) ([]models.Page, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, slug, title, meta_description, meta_keywords, og_image, updated_at FROM pages ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pages []models.Page
	for rows.Next() {
		var p models.Page
		var updated string
		if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.MetaDescription, &p.MetaKeywords, &p.OGImage, &updated); err != nil {
			return nil, err
		}
		p.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updated)
		p.Sections = map[string]models.PageSection{}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

func (s *SQLiteStore) UpsertPage(ctx context.Context, page models.Page) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pages (slug, title, meta_description, meta_keywords, og_image, updated_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(slug) DO UPDATE SET
			title = excluded.title,
			meta_description = excluded.meta_description,
			meta_keywords = excluded.meta_keywords,
			og_image = CASE WHEN excluded.og_image = '' THEN pages.og_image ELSE excluded.og_image END,
			updated_at = datetime('now')
	`, page.Slug, page.Title, page.MetaDescription, page.MetaKeywords, page.OGImage)
	return err
}

func (s *SQLiteStore) UpsertPageSection(ctx context.Context, pageID int64, section models.PageSection) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO page_sections (page_id, section_key, title, subtitle, body)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(page_id, section_key) DO UPDATE SET
			title = excluded.title,
			subtitle = excluded.subtitle,
			body = excluded.body
	`, pageID, section.SectionKey, section.Title, section.Subtitle, section.Body)
	return err
}

func (s *SQLiteStore) ListSkills(ctx context.Context) ([]models.Skill, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, label, sort_order FROM skills ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Skill
	for rows.Next() {
		var sk models.Skill
		if err := rows.Scan(&sk.ID, &sk.Label, &sk.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, sk)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ReplaceSkills(ctx context.Context, skills []models.Skill) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM skills`); err != nil {
		_ = tx.Rollback()
		return err
	}
	for i, sk := range skills {
		if _, err := tx.ExecContext(ctx, `INSERT INTO skills (label, sort_order) VALUES (?, ?)`, sk.Label, i); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) ListEducation(ctx context.Context) ([]models.Education, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, degree, school, year, description, sort_order FROM education ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Education
	for rows.Next() {
		var e models.Education
		if err := rows.Scan(&e.ID, &e.Degree, &e.School, &e.Year, &e.Description, &e.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) CreateEducation(ctx context.Context, e models.Education) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO education (degree, school, year, description, sort_order) VALUES (?, ?, ?, ?, ?)`,
		e.Degree, e.School, e.Year, e.Description, e.SortOrder)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) UpdateEducation(ctx context.Context, e models.Education) error {
	_, err := s.db.ExecContext(ctx, `UPDATE education SET degree=?, school=?, year=?, description=?, sort_order=? WHERE id=?`,
		e.Degree, e.School, e.Year, e.Description, e.SortOrder, e.ID)
	return err
}

func (s *SQLiteStore) DeleteEducation(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM education WHERE id=?`, id)
	return err
}

func (s *SQLiteStore) ListExperiences(ctx context.Context) ([]models.Experience, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, role, company, period, description, sort_order FROM experiences ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Experience
	for rows.Next() {
		var e models.Experience
		if err := rows.Scan(&e.ID, &e.Role, &e.Company, &e.Period, &e.Description, &e.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ReplaceExperiences(ctx context.Context, items []models.Experience) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM experiences`); err != nil {
		return err
	}
	for i, e := range items {
		if _, err := tx.ExecContext(ctx, `INSERT INTO experiences (role, company, period, description, sort_order) VALUES (?, ?, ?, ?, ?)`,
			e.Role, e.Company, e.Period, e.Description, i); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) CreateExperience(ctx context.Context, e models.Experience) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO experiences (role, company, period, description, sort_order) VALUES (?, ?, ?, ?, ?)`,
		e.Role, e.Company, e.Period, e.Description, e.SortOrder)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) UpdateExperience(ctx context.Context, e models.Experience) error {
	_, err := s.db.ExecContext(ctx, `UPDATE experiences SET role=?, company=?, period=?, description=?, sort_order=? WHERE id=?`,
		e.Role, e.Company, e.Period, e.Description, e.SortOrder, e.ID)
	return err
}

func (s *SQLiteStore) DeleteExperience(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM experiences WHERE id=?`, id)
	return err
}

func (s *SQLiteStore) loadProjectChildren(ctx context.Context, p *models.Project) error {
	tagRows, err := s.db.QueryContext(ctx, `SELECT tag FROM project_tags WHERE project_id=?`, p.ID)
	if err != nil {
		return err
	}
	defer tagRows.Close()
	for tagRows.Next() {
		var t string
		if err := tagRows.Scan(&t); err != nil {
			return err
		}
		p.Tags = append(p.Tags, t)
	}
	featRows, err := s.db.QueryContext(ctx, `SELECT feature FROM project_features WHERE project_id=? ORDER BY sort_order, id`, p.ID)
	if err != nil {
		return err
	}
	defer featRows.Close()
	for featRows.Next() {
		var f string
		if err := featRows.Scan(&f); err != nil {
			return err
		}
		p.Features = append(p.Features, f)
	}
	techRows, err := s.db.QueryContext(ctx, `SELECT tech FROM project_tech WHERE project_id=? ORDER BY sort_order, id`, p.ID)
	if err != nil {
		return err
	}
	defer techRows.Close()
	for techRows.Next() {
		var t string
		if err := techRows.Scan(&t); err != nil {
			return err
		}
		p.TechStack = append(p.TechStack, t)
	}
	return nil
}

func scanProject(rows interface {
	Scan(dest ...any) error
}) (models.Project, error) {
	var p models.Project
	var featured, published int
	err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.ShortDescription, &p.FullDescription,
		&p.ImageURL, &p.LiveURL, &p.GithubURL, &featured, &published, &p.SortOrder)
	p.Featured = featured == 1
	p.Published = published == 1
	return p, err
}

const projectSelect = `SELECT id, slug, title, short_description, full_description, image_url, live_url, github_url, featured, published, sort_order FROM projects`

func (s *SQLiteStore) ListProjects(ctx context.Context, featuredOnly bool) ([]models.Project, error) {
	q := projectSelect + ` WHERE published = 1`
	if featuredOnly {
		q += ` AND featured = 1`
	}
	q += ` ORDER BY sort_order, id`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	var out []models.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for i := range out {
		if err := s.loadProjectChildren(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *SQLiteStore) ListAllProjects(ctx context.Context) ([]models.Project, error) {
	rows, err := s.db.QueryContext(ctx, projectSelect+` ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	var out []models.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for i := range out {
		if err := s.loadProjectChildren(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *SQLiteStore) ListProjectsByFilter(ctx context.Context, filter string) ([]models.Project, error) {
	filter = strings.TrimSpace(filter)
	if filter == "" || strings.EqualFold(filter, "All") {
		return s.ListProjects(ctx, false)
	}
	var tagList []string
	switch strings.ToLower(filter) {
	case "web":
		tagList = []string{"React", "Web", "UI/UX", "Template"}
	case "3d":
		tagList = []string{"3D", "WebGL", "VR", "ThreeJS"}
	case "design":
		tagList = []string{"Design", "Animation", "UI/UX"}
	default:
		tagList = []string{filter}
	}
	placeholders := make([]string, len(tagList))
	args := make([]any, len(tagList))
	for i, t := range tagList {
		placeholders[i] = "?"
		args[i] = t
	}
	q := fmt.Sprintf(`
		SELECT DISTINCT p.id, p.slug, p.title, p.short_description, p.full_description, p.image_url, p.live_url, p.github_url, p.featured, p.published, p.sort_order
		FROM projects p
		JOIN project_tags t ON t.project_id = p.id
		WHERE p.published = 1 AND t.tag IN (%s)
		ORDER BY p.sort_order, p.id
	`, strings.Join(placeholders, ","))
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var out []models.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for i := range out {
		if err := s.loadProjectChildren(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *SQLiteStore) GetProject(ctx context.Context, id int64) (*models.Project, error) {
	row := s.db.QueryRowContext(ctx, projectSelect+` WHERE id = ?`, id)
	p, err := scanProject(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := s.loadProjectChildren(ctx, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *SQLiteStore) GetProjectBySlug(ctx context.Context, slug string) (*models.Project, error) {
	row := s.db.QueryRowContext(ctx, projectSelect+` WHERE slug = ?`, slug)
	p, err := scanProject(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := s.loadProjectChildren(ctx, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *SQLiteStore) saveProjectChildren(ctx context.Context, tx *sql.Tx, projectID int64, p models.Project) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_tags WHERE project_id=?`, projectID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_features WHERE project_id=?`, projectID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_tech WHERE project_id=?`, projectID); err != nil {
		return err
	}
	for _, t := range p.Tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO project_tags (project_id, tag) VALUES (?, ?)`, projectID, t); err != nil {
			return err
		}
	}
	for i, f := range p.Features {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO project_features (project_id, feature, sort_order) VALUES (?, ?, ?)`, projectID, f, i); err != nil {
			return err
		}
	}
	for i, t := range p.TechStack {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO project_tech (project_id, tech, sort_order) VALUES (?, ?, ?)`, projectID, t, i); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) CreateProject(ctx context.Context, p models.Project) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	feat, pub := 0, 0
	if p.Featured {
		feat = 1
	}
	if p.Published {
		pub = 1
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO projects (slug, title, short_description, full_description, image_url, live_url, github_url, featured, published, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Slug, p.Title, p.ShortDescription, p.FullDescription, p.ImageURL, p.LiveURL, p.GithubURL, feat, pub, p.SortOrder)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if err := s.saveProjectChildren(ctx, tx, id, p); err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	return id, tx.Commit()
}

func (s *SQLiteStore) UpdateProject(ctx context.Context, p models.Project) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	feat, pub := 0, 0
	if p.Featured {
		feat = 1
	}
	if p.Published {
		pub = 1
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE projects SET slug=?, title=?, short_description=?, full_description=?, image_url=?, live_url=?, github_url=?,
		featured=?, published=?, sort_order=?, updated_at=datetime('now') WHERE id=?`,
		p.Slug, p.Title, p.ShortDescription, p.FullDescription, p.ImageURL, p.LiveURL, p.GithubURL, feat, pub, p.SortOrder, p.ID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := s.saveProjectChildren(ctx, tx, p.ID, p); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) DeleteProject(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id=?`, id)
	return err
}

func (s *SQLiteStore) CountProjects(ctx context.Context) (models.DashboardStats, error) {
	var st models.DashboardStats
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1), COALESCE(SUM(CASE WHEN published=1 THEN 1 ELSE 0 END), 0) FROM projects`).Scan(&st.ProjectCount, &st.PublishedCount)
	if err != nil {
		return st, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM contact_messages`).Scan(&st.MessageCount)
	if err != nil {
		return st, err
	}
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM blogs`).Scan(&st.BlogCount)
	return st, nil
}

func (s *SQLiteStore) ListPrivacySections(ctx context.Context) ([]models.PrivacySection, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, heading, body, sort_order FROM privacy_sections ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.PrivacySection
	for rows.Next() {
		var sec models.PrivacySection
		if err := rows.Scan(&sec.ID, &sec.Heading, &sec.Body, &sec.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, sec)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) CreatePrivacySection(ctx context.Context, sec models.PrivacySection) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO privacy_sections (heading, body, sort_order) VALUES (?, ?, ?)`, sec.Heading, sec.Body, sec.SortOrder)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) UpdatePrivacySection(ctx context.Context, sec models.PrivacySection) error {
	_, err := s.db.ExecContext(ctx, `UPDATE privacy_sections SET heading=?, body=?, sort_order=? WHERE id=?`, sec.Heading, sec.Body, sec.SortOrder, sec.ID)
	return err
}

func (s *SQLiteStore) DeletePrivacySection(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM privacy_sections WHERE id=?`, id)
	return err
}

func (s *SQLiteStore) CreateContactMessage(ctx context.Context, m models.ContactMessage) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO contact_messages (name, email, subject, message) VALUES (?, ?, ?, ?)`,
		m.Name, m.Email, m.Subject, m.Message)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) ListContactMessages(ctx context.Context) ([]models.ContactMessage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, email, subject, message, notified, created_at FROM contact_messages ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ContactMessage
	for rows.Next() {
		var m models.ContactMessage
		var notified int
		var created string
		if err := rows.Scan(&m.ID, &m.Name, &m.Email, &m.Subject, &m.Message, &notified, &created); err != nil {
			return nil, err
		}
		m.Notified = notified == 1
		m.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) MarkMessageNotified(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE contact_messages SET notified=1 WHERE id=?`, id)
	return err
}

func (s *SQLiteStore) DeleteContactMessage(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM contact_messages WHERE id=?`, id)
	return err
}

func (s *SQLiteStore) CountMessages(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM contact_messages`).Scan(&n)
	return n, err
}

func scanBlog(scanner interface{ Scan(dest ...any) error }) (models.Blog, error) {
	var b models.Blog
	var published int
	var publishedAt, created, updated sql.NullString
	err := scanner.Scan(
		&b.ID, &b.Slug, &b.Title, &b.Excerpt, &b.Body, &b.CoverImage,
		&b.MetaTitle, &b.MetaDescription, &b.MetaKeywords,
		&published, &b.ViewCount, &publishedAt, &created, &updated,
	)
	if err != nil {
		return b, err
	}
	b.Published = published == 1
	if publishedAt.Valid && publishedAt.String != "" {
		if t, e := time.Parse("2006-01-02 15:04:05", publishedAt.String); e == nil {
			b.PublishedAt = &t
		}
	}
	b.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created.String)
	b.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updated.String)
	return b, nil
}

const blogSelect = `SELECT id, slug, title, excerpt, body, cover_image, meta_title, meta_description, meta_keywords, published, view_count, published_at, created_at, updated_at FROM blogs`

func (s *SQLiteStore) ListBlogs(ctx context.Context, publishedOnly bool) ([]models.Blog, error) {
	blogs, _, err := s.ListBlogsPaged(ctx, publishedOnly, 1, 1000)
	return blogs, err
}

func (s *SQLiteStore) ListBlogsPaged(ctx context.Context, publishedOnly bool, page, perPage int) ([]models.Blog, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 6
	}
	where := ""
	if publishedOnly {
		where = ` WHERE published = 1`
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM blogs`+where).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * perPage
	q := blogSelect + where + ` ORDER BY COALESCE(published_at, created_at) DESC, id DESC LIMIT ? OFFSET ?`
	rows, err := s.db.QueryContext(ctx, q, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.Blog
	for rows.Next() {
		b, err := scanBlog(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, b)
	}
	return out, total, rows.Err()
}

func (s *SQLiteStore) GetBlog(ctx context.Context, id int64) (*models.Blog, error) {
	b, err := scanBlog(s.db.QueryRowContext(ctx, blogSelect+` WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *SQLiteStore) GetBlogBySlug(ctx context.Context, slug string) (*models.Blog, error) {
	b, err := scanBlog(s.db.QueryRowContext(ctx, blogSelect+` WHERE slug = ?`, slug))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *SQLiteStore) CreateBlog(ctx context.Context, b models.Blog) (int64, error) {
	pub := 0
	if b.Published {
		pub = 1
	}
	var publishedAt any
	if b.PublishedAt != nil {
		publishedAt = b.PublishedAt.UTC().Format("2006-01-02 15:04:05")
	} else if b.Published {
		publishedAt = time.Now().UTC().Format("2006-01-02 15:04:05")
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO blogs (slug, title, excerpt, body, cover_image, meta_title, meta_description, meta_keywords, published, view_count, published_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.Slug, b.Title, b.Excerpt, b.Body, b.CoverImage, b.MetaTitle, b.MetaDescription, b.MetaKeywords, pub, b.ViewCount, publishedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) UpdateBlog(ctx context.Context, b models.Blog) error {
	pub := 0
	if b.Published {
		pub = 1
	}
	var publishedAt any
	if b.PublishedAt != nil {
		publishedAt = b.PublishedAt.UTC().Format("2006-01-02 15:04:05")
	} else if b.Published {
		publishedAt = time.Now().UTC().Format("2006-01-02 15:04:05")
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE blogs SET slug=?, title=?, excerpt=?, body=?, cover_image=?, meta_title=?, meta_description=?, meta_keywords=?,
		published=?, published_at=COALESCE(?, published_at), updated_at=datetime('now') WHERE id=?`,
		b.Slug, b.Title, b.Excerpt, b.Body, b.CoverImage, b.MetaTitle, b.MetaDescription, b.MetaKeywords, pub, publishedAt, b.ID)
	return err
}

func (s *SQLiteStore) DeleteBlog(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM blogs WHERE id=?`, id)
	return err
}

func (s *SQLiteStore) IncrementBlogViews(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE blogs SET view_count = view_count + 1 WHERE id=?`, id)
	return err
}

func (s *SQLiteStore) CountBlogs(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM blogs`).Scan(&n)
	return n, err
}

func (s *SQLiteStore) ListFooterLinks(ctx context.Context, enabledOnly bool) ([]models.FooterLink, error) {
	q := `SELECT id, label, url, icon, sort_order, enabled FROM footer_links`
	if enabledOnly {
		q += ` WHERE enabled = 1`
	}
	q += ` ORDER BY sort_order, id`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.FooterLink
	for rows.Next() {
		var l models.FooterLink
		var enabled int
		if err := rows.Scan(&l.ID, &l.Label, &l.URL, &l.Icon, &l.SortOrder, &enabled); err != nil {
			return nil, err
		}
		l.Enabled = enabled == 1
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetFooterLink(ctx context.Context, id int64) (*models.FooterLink, error) {
	var l models.FooterLink
	var enabled int
	err := s.db.QueryRowContext(ctx, `SELECT id, label, url, icon, sort_order, enabled FROM footer_links WHERE id=?`, id).
		Scan(&l.ID, &l.Label, &l.URL, &l.Icon, &l.SortOrder, &enabled)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	l.Enabled = enabled == 1
	return &l, nil
}

func (s *SQLiteStore) CreateFooterLink(ctx context.Context, l models.FooterLink) (int64, error) {
	en := 0
	if l.Enabled {
		en = 1
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO footer_links (label, url, icon, sort_order, enabled) VALUES (?, ?, ?, ?, ?)`,
		l.Label, l.URL, l.Icon, l.SortOrder, en)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) UpdateFooterLink(ctx context.Context, l models.FooterLink) error {
	en := 0
	if l.Enabled {
		en = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE footer_links SET label=?, url=?, icon=?, sort_order=?, enabled=? WHERE id=?`,
		l.Label, l.URL, l.Icon, l.SortOrder, en, l.ID)
	return err
}

func (s *SQLiteStore) DeleteFooterLink(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM footer_links WHERE id=?`, id)
	return err
}
