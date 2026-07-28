package starterkit

// SnippetMeta describes a single well-known, frequently reused Starter Kit
// file exposed via the get_template_snippet tool.
//
// Note: the Starter Kit does not have separate "header.php"/"footer.php"
// files. Its "page" component (components/page.php) covers the full HTML
// document shell (both the <head> and the <body> wrapper) in one file, and
// is the closest equivalent. The keys below reflect the repository's actual
// structure rather than a generic theme's file names.
type SnippetMeta struct {
	// Key is the short identifier passed to get_template_snippet.
	Key string
	// Path is the file's path relative to the repository root.
	Path string
	// Title is a human-readable name for the snippet.
	Title string
	// Description explains what the file does and when to look at it.
	Description string
}

// knownSnippets is the curated registry backing get_template_snippet.
var knownSnippets = []SnippetMeta{
	{
		Key:   "page",
		Path:  "components/page.php",
		Title: "Page shell (components/page.php)",
		Description: "The base HTML document structure included by every template: doctype, <head> with " +
			"the theme's compiled CSS/JS links, and <body> including layout.php. This is the closest " +
			"equivalent to a combined header+footer in a classic theme.",
	},
	{
		Key:   "content",
		Path:  "components/content.php",
		Title: "Content block (components/content.php)",
		Description: "Renders the page title, an optional date/tags subtitle, and the '+main' block editor " +
			"field. Included by the default page layout.",
	},
	{
		Key:   "pagination",
		Path:  "components/pagination.php",
		Title: "Pagination (components/pagination.php)",
		Description: "Renders previous/next and numbered pagination links for a pagelist, using the " +
			"':paginationCount' runtime variable and the 'queryStringMerge' statement.",
	},
	{
		Key:   "pagelist-grid",
		Path:  "blocks/pagelist/grid.php",
		Title: "Pagelist grid (blocks/pagelist/grid.php)",
		Description: "Renders a pagelist as a grid of preview cards, each showing the first image and first " +
			"paragraph found in the page's '+main' field.",
	},
	{
		Key:   "default-template",
		Path:  "default.php",
		Title: "Default template (default.php)",
		Description: "The simplest possible template: a root-level .php file that just includes " +
			"components/page.php.",
	},
	{
		Key:   "pagelist-template",
		Path:  "pagelist.php",
		Title: "Pagelist template (pagelist.php)",
		Description: "Extends the page component, overrides its 'main' snippet, builds a pagelist with " +
			"newPagelist, renders a filter menu, the pagelist grid block, and pagination.",
	},
	{
		Key:   "page-not-found",
		Path:  "page_not_found.php",
		Title: "404 template (page_not_found.php)",
		Description: "Template rendered for unresolved URLs. Reuses the page component, same as any other template.",
	},
	{
		Key:   "functions",
		Path:  "lib/functions.php",
		Title: "Helper functions (lib/functions.php)",
		Description: "Example of defining a custom PHP helper function (icon()) that becomes callable from " +
			"templates, e.g. '<@ icon { name: ... } @>'.",
	},
	{
		Key:   "theme-config",
		Path:  "theme.json",
		Title: "Theme configuration (theme.json)",
		Description: "Dashboard-facing theme metadata: field order, labels, select-field option sets, field " +
			"masks, and tooltips.",
	},
}

// KnownSnippets returns the full snippet registry.
func KnownSnippets() []SnippetMeta {
	return knownSnippets
}

// FindSnippet looks up a snippet by key (case-sensitive, matching the keys
// returned by KnownSnippets).
func FindSnippet(key string) (SnippetMeta, bool) {
	for _, m := range knownSnippets {
		if m.Key == key {
			return m, true
		}
	}
	return SnippetMeta{}, false
}
