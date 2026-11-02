package migrations

// ProjectDomainType adds the domain_type column to the projects table so that
// each project can declare which domain agent (writing, coding, investment,
// general) manages it. This replaces hardcoded observer path registration in
// app.go with a DB-driven lookup at boot (AIA-15 §23/§24, thinking.md §一/§三).
const ProjectDomainType = `
ALTER TABLE projects ADD COLUMN domain_type TEXT NOT NULL DEFAULT 'general';
CREATE INDEX IF NOT EXISTS idx_projects_domain_type ON projects(domain_type);
`
