package router

import (
	"path/filepath"
	"strings"
)

// Router maps files to their corresponding guides
type Router struct {
	// Mapping of file suffixes to guide names
	mapping map[string]string
}

// NewRouter creates a new Router instance
func NewRouter() *Router {
	return &Router{
		mapping: map[string]string{
			".page.ts":    "page.md",
			".page.tsx":   "page.md",
			".const.ts":   "const.md",
			".const.tsx":  "const.md",
			".utils.ts":   "utils.md",
			".utils.tsx":  "utils.md",
			".hooks.ts":   "hooks.md",
			".hooks.tsx":  "hooks.md",
			".list.ts":    "list.md",
			".list.tsx":   "list.md",
			".detail.ts":  "detail.md",
			".detail.tsx": "detail.md",
			".form.ts":    "form.md",
			".form.tsx":   "form.md",
			".table.ts":   "table.md",
			".table.tsx":  "table.md",
		},
	}
}

// GetGuide returns the guide filename for the given file
func (r *Router) GetGuide(filename string) string {
	base := filepath.Base(filename)
	
	for suffix, guide := range r.mapping {
		if strings.HasSuffix(base, suffix) {
			return guide
		}
	}
	
	return ""
}
