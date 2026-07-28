package docs

// DocPage represents a single entry in the Automad documentation sitemap.
type DocPage struct {
	// Value is the display title of the page.
	Value string
	// URL is the relative path (e.g. "/user-guide/creating-pages").
	URL string
	// Parent is the parent section name (e.g. "User Guide").
	Parent string
}

// Sitemap returns all known Automad documentation pages, derived from the
// autocomplete index embedded in automad.org.
func Sitemap() []DocPage {
	return []DocPage{
		{Value: "Automad", URL: "/", Parent: ""},
		{Value: "Getting Started", URL: "/getting-started", Parent: "Automad"},
		{Value: "Local Installation", URL: "/getting-started/local-installation", Parent: "Getting Started"},
		{Value: "System Requirements", URL: "/getting-started/system-requirements", Parent: "Getting Started"},
		{Value: "Terms of Use", URL: "/getting-started/terms-of-use", Parent: "Getting Started"},
		{Value: "Live Demo", URL: "/try", Parent: "Automad"},
		{Value: "Headless Mode", URL: "/headless-mode", Parent: "Automad"},
		{Value: "Packages", URL: "/packages-1", Parent: "Automad"},
		{Value: "Discuss", URL: "/discuss", Parent: "Automad"},
		{Value: "☀ Version 2", URL: "/version-2", Parent: "Automad"},

		// User Guide
		{Value: "User Guide", URL: "/user-guide", Parent: "Automad"},
		{Value: "Creating Pages", URL: "/user-guide/creating-pages", Parent: "User Guide"},
		{Value: "General Data and Files", URL: "/user-guide/general-data-and-files", Parent: "User Guide"},
		{Value: "Key-Combos", URL: "/user-guide/key-combos", Parent: "User Guide"},
		{Value: "Linking to Files and Pages", URL: "/user-guide/linking-to-files-and-pages", Parent: "User Guide"},
		{Value: "Resizing Images", URL: "/user-guide/resizing-images", Parent: "User Guide"},
		{Value: "Using Blocks", URL: "/user-guide/using-blocks", Parent: "User Guide"},

		// Developer Guide
		{Value: "Developer Guide", URL: "/developer-guide", Parent: "Automad"},
		{Value: "API Reference", URL: "/developer-guide/api-reference", Parent: "Developer Guide"},
		{Value: "Building Themes", URL: "/developer-guide/building-themes", Parent: "Developer Guide"},
		{Value: "Block Layouts", URL: "/developer-guide/building-themes/block-layouts", Parent: "Building Themes"},
		{Value: "Customizing Blocks", URL: "/developer-guide/building-themes/customizing-blocks", Parent: "Building Themes"},
		{Value: "Plain PHP", URL: "/developer-guide/building-themes/plain-php", Parent: "Building Themes"},
		{Value: "theme.json", URL: "/developer-guide/building-themes/theme-json", Parent: "Building Themes"},

		// Template Language
		{Value: "Template Language", URL: "/developer-guide/building-themes/template-language", Parent: "Building Themes"},
		{Value: "Control Structures", URL: "/developer-guide/building-themes/template-language/control-structures", Parent: "Template Language"},
		{Value: "for", URL: "/developer-guide/building-themes/template-language/control-structures/for", Parent: "Control Structures"},
		{Value: "foreach", URL: "/developer-guide/building-themes/template-language/control-structures/foreach", Parent: "Control Structures"},
		{Value: "if", URL: "/developer-guide/building-themes/template-language/control-structures/if", Parent: "Control Structures"},
		{Value: "with", URL: "/developer-guide/building-themes/template-language/control-structures/with", Parent: "Control Structures"},
		{Value: "Includes", URL: "/developer-guide/building-themes/template-language/includes", Parent: "Template Language"},
		{Value: "Inheritance", URL: "/developer-guide/building-themes/template-language/inheritance", Parent: "Template Language"},
		{Value: "Multilingual Content", URL: "/developer-guide/building-themes/template-language/multilingual-content", Parent: "Template Language"},
		{Value: "Recursive Navigations", URL: "/developer-guide/building-themes/template-language/recursive-navigations", Parent: "Template Language"},
		{Value: "Snippets", URL: "/developer-guide/building-themes/template-language/snippets", Parent: "Template Language"},
		{Value: "Using Extensions", URL: "/developer-guide/building-themes/template-language/using-extensions", Parent: "Template Language"},
		{Value: "Working with Images", URL: "/developer-guide/building-themes/template-language/working-with-images", Parent: "Template Language"},

		// Objects
		{Value: "Objects", URL: "/developer-guide/building-themes/template-language/objects", Parent: "Template Language"},
		{Value: "Filelist", URL: "/developer-guide/building-themes/template-language/objects/filelist", Parent: "Objects"},
		{Value: "Filters", URL: "/developer-guide/building-themes/template-language/objects/filters", Parent: "Objects"},
		{Value: "Pagelist", URL: "/developer-guide/building-themes/template-language/objects/pagelist", Parent: "Objects"},
		{Value: "Tags", URL: "/developer-guide/building-themes/template-language/objects/tags", Parent: "Objects"},

		// Pipe Functions
		{Value: "Pipe", URL: "/developer-guide/building-themes/template-language/pipe", Parent: "Template Language"},
		{Value: "ceil", URL: "/developer-guide/building-themes/template-language/pipe/ceil", Parent: "Pipe"},
		{Value: "dateFormat", URL: "/developer-guide/building-themes/template-language/pipe/dateformat", Parent: "Pipe"},
		{Value: "def", URL: "/developer-guide/building-themes/template-language/pipe/def", Parent: "Pipe"},
		{Value: "escape", URL: "/developer-guide/building-themes/template-language/pipe/escape", Parent: "Pipe"},
		{Value: "findFirstImage", URL: "/developer-guide/building-themes/template-language/pipe/findfirstimage", Parent: "Pipe"},
		{Value: "findFirstParagraph", URL: "/developer-guide/building-themes/template-language/pipe/findfirstparagraph", Parent: "Pipe"},
		{Value: "floor", URL: "/developer-guide/building-themes/template-language/pipe/floor", Parent: "Pipe"},
		{Value: "markdown", URL: "/developer-guide/building-themes/template-language/pipe/markdown", Parent: "Pipe"},
		{Value: "match", URL: "/developer-guide/building-themes/template-language/pipe/match", Parent: "Pipe"},
		{Value: "replace", URL: "/developer-guide/building-themes/template-language/pipe/replace", Parent: "Pipe"},
		{Value: "round", URL: "/developer-guide/building-themes/template-language/pipe/round", Parent: "Pipe"},
		{Value: "sanitize", URL: "/developer-guide/building-themes/template-language/pipe/sanitize", Parent: "Pipe"},
		{Value: "shorten", URL: "/developer-guide/building-themes/template-language/pipe/shorten", Parent: "Pipe"},
		{Value: "stripEnd", URL: "/developer-guide/building-themes/template-language/pipe/stripend", Parent: "Pipe"},
		{Value: "stripStart", URL: "/developer-guide/building-themes/template-language/pipe/stripstart", Parent: "Pipe"},
		{Value: "stripTags", URL: "/developer-guide/building-themes/template-language/pipe/striptags", Parent: "Pipe"},
		{Value: "strlen", URL: "/developer-guide/building-themes/template-language/pipe/strlen", Parent: "Pipe"},
		{Value: "strtolower", URL: "/developer-guide/building-themes/template-language/pipe/strtolower", Parent: "Pipe"},
		{Value: "strtoupper", URL: "/developer-guide/building-themes/template-language/pipe/strtoupper", Parent: "Pipe"},
		{Value: "ucwords", URL: "/developer-guide/building-themes/template-language/pipe/ucwords", Parent: "Pipe"},

		// Toolbox
		{Value: "Toolbox", URL: "/developer-guide/building-themes/template-language/toolbox", Parent: "Template Language"},
		{Value: "breadcrumbs", URL: "/developer-guide/building-themes/template-language/toolbox/breadcrumbs", Parent: "Toolbox"},
		{Value: "filelist", URL: "/developer-guide/building-themes/template-language/toolbox/filelist", Parent: "Toolbox"},
		{Value: "img", URL: "/developer-guide/building-themes/template-language/toolbox/img", Parent: "Toolbox"},
		{Value: "nav", URL: "/developer-guide/building-themes/template-language/toolbox/nav", Parent: "Toolbox"},
		{Value: "navChildren", URL: "/developer-guide/building-themes/template-language/toolbox/navchildren", Parent: "Toolbox"},
		{Value: "navSiblings", URL: "/developer-guide/building-themes/template-language/toolbox/navsiblings", Parent: "Toolbox"},
		{Value: "navTop", URL: "/developer-guide/building-themes/template-language/toolbox/navtop", Parent: "Toolbox"},
		{Value: "navTree", URL: "/developer-guide/building-themes/template-language/toolbox/navtree", Parent: "Toolbox"},
		{Value: "newPagelist", URL: "/developer-guide/building-themes/template-language/toolbox/newpagelist", Parent: "Toolbox"},
		{Value: "pagelist", URL: "/developer-guide/building-themes/template-language/toolbox/pagelist", Parent: "Toolbox"},
		{Value: "queryStringMerge", URL: "/developer-guide/building-themes/template-language/toolbox/querystringmerge", Parent: "Toolbox"},
		{Value: "redirect", URL: "/developer-guide/building-themes/template-language/toolbox/redirect", Parent: "Toolbox"},
		{Value: "set", URL: "/developer-guide/building-themes/template-language/toolbox/set", Parent: "Toolbox"},

		// Variables
		{Value: "Variables", URL: "/developer-guide/building-themes/template-language/variables", Parent: "Template Language"},
		{Value: "Reserved Variables", URL: "/developer-guide/building-themes/template-language/variables/reserved-variables", Parent: "Variables"},
		{Value: "Runtime Variables", URL: "/developer-guide/building-themes/template-language/variables/runtime-variables", Parent: "Variables"},

		// Cheat Sheets
		{Value: "Cheat Sheets", URL: "/developer-guide/cheat-sheets", Parent: "Developer Guide"},
		{Value: "Creating Theme Packages", URL: "/developer-guide/cheat-sheets/creating-theme-packages", Parent: "Cheat Sheets"},
		{Value: "Plain PHP Snippets", URL: "/developer-guide/cheat-sheets/plain-php-snippets", Parent: "Cheat Sheets"},
		{Value: "Useful Template Snippets", URL: "/developer-guide/cheat-sheets/useful-template-snippets", Parent: "Cheat Sheets"},

		// Developing Extensions
		{Value: "Developing Extensions", URL: "/developer-guide/developing-extensions", Parent: "Developer Guide"},
		{Value: "Custom Pipe Functions", URL: "/developer-guide/developing-extensions/custom-pipe-functions", Parent: "Developing Extensions"},
		{Value: "Generic Extensions", URL: "/developer-guide/developing-extensions/generic-extensions", Parent: "Developing Extensions"},
		{Value: "Editor Plugins", URL: "/developer-guide/editor-plugins", Parent: "Developer Guide"},
		{Value: "Publishing Packages", URL: "/developer-guide/publishing-packages", Parent: "Developer Guide"},

		// System
		{Value: "System", URL: "/system", Parent: "Automad"},
		{Value: "Allowed File Types", URL: "/system/allowed-file-types", Parent: "System"},
		{Value: "Caching", URL: "/system/caching", Parent: "System"},
		{Value: "Console", URL: "/system/console", Parent: "System"},
		{Value: "Debugging", URL: "/system/debugging", Parent: "System"},
		{Value: "Installing Packages", URL: "/system/installing-packages", Parent: "System"},
		{Value: "Language", URL: "/system/language", Parent: "System"},
		{Value: "RSS Feed", URL: "/system/rss-feed", Parent: "System"},
		{Value: "Securing the Dashboard", URL: "/system/securing-the-dashboard", Parent: "System"},
		{Value: "sitemap.xml", URL: "/system/sitemap-xml", Parent: "System"},
		{Value: "Updating Automad", URL: "/system/updating-automad", Parent: "System"},
		{Value: "Users", URL: "/system/users", Parent: "System"},
	}
}

// SitemapByParent returns pages grouped by their parent section.
func SitemapByParent() map[string][]DocPage {
	grouped := make(map[string][]DocPage)
	for _, p := range Sitemap() {
		grouped[p.Parent] = append(grouped[p.Parent], p)
	}
	return grouped
}

// FindByURL returns the DocPage matching the given relative URL, or nil.
func FindByURL(url string) *DocPage {
	for _, p := range Sitemap() {
		if p.URL == url {
			return &p
		}
	}
	return nil
}
