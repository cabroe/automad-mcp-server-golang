package starterkit

// fallback.go embeds a snapshot of a curated set of frequently used Starter
// Kit files (captured from the master branch when this package was written),
// used only when a live GitHub API request fails — e.g. the rate limit is
// exhausted or the network is unreachable. Fallback content may lag behind
// the upstream repository; callers are expected to flag results that came
// from here so a user (or an AI assistant) doesn't mistake it for a live
// fetch.

// fallbackFiles maps a repository-relative path to its last-known content.
var fallbackFiles = map[string]string{
	"theme.json": `{
  "name": "Starter Kit",
  "description": "The Automad theme starter kit for developers",
  "author": "Marc Anton Dahmen",
  "license": "MIT",
  "masks": {
    "page": [],
    "shared": ["+main"]
  },
  "fieldOrder": [
    "selectColorTheme",
    "selectPageDateFormat",
    "brand",
    "checkboxHidePreviousAndNextPageNavigation",
    "checkboxShowPageInNavbar",
    "checkboxShowPageInFooter"
  ],
  "labels": {
    "brand": "Brand logo (SVG, HTML or text)",
    "selectPagelistSort": "Pagelist: Sorting",
    "selectPagelistSubset": "Pagelist: Type of pagelist subset"
  },
  "options": {
    "selectColorTheme": {
      "switcher": "Show theme switcher, visitor can select theme",
      "light": "Light theme",
      "dark": "Dark theme"
    },
    "selectPageDateFormat": {
      "M Y": "Month and Year",
      "Y": "Year only",
      "j. M Y": "Long"
    },
    "selectPagelistSubset": {
      "all": "All pages",
      "children": "Show only children of the selected context",
      "siblings": "Show only siblings of the selected context"
    },
    "selectPagelistSort": {
      ":index asc": "Manual order",
      "date desc": "Recent first",
      "date asc": "Oldest first",
      "title asc": "Alphabetically"
    }
  },
  "tooltips": {
    "+main": "The main content block area",
    "+footer": "A custom footer section that can also be defined globally"
  }
}
`,

	"composer.json": `{
  "name": "automad/theme-starter-kit",
  "description": "The Automad theme starter kit for developers",
  "type": "automad-package",
  "keywords": [
    "automad",
    "theme",
    "starter-kit"
  ],
  "authors": [
    {
      "name": "Marc Anton Dahmen",
      "homepage": "https://marcdahmen.de"
    }
  ],
  "license": "MIT",
  "require": {
    "automad/package-installer": "^1.1 || dev-master"
  }
}
`,

	"default.php": `<#

The entire logic and markup of the default template is located inside the "page" component
that is simply included here.

#>
<@ components/page.php @>
`,

	"page_not_found.php": `<@ components/page.php @>
`,

	"pagelist.php": `<#

Also the "pagelist" template is based on the "page" component.
The only exception is the content of the "main" snippet that is deeply nested
inside the page component.

In order to inherit the "page" component and only changing the main snippet,
we can simply redefine it after the include statement.

#>
<@ components/page.php @>

<@~ snippet main ~@>
	<main class="kit-layout__main">
		<#

		The main content goes here.

		#>
		<@ components/content.php @>

		<#

		Before we can render a filter menu and the actual page previews,
		we have to define a pagelist that respects query string parameters and
		the user's configuration.

		#>
		<@ newPagelist {
			type: @{ selectPagelistSubset | def (false) },
			sort: @{ selectPagelistSort },
			context: @{ urlPagelistContext | def (false) },
			filter: @{ ?filter },
			page: @{ ?page | def (1) },
			limit: 8
		} @>

		<section class="am-block">
			<#

			The following section creates a little filter menu with clickable tags
			that filter the displayed pages.

			#>
			<div class="kit-filters">
				<a
					href="@{ url }"
					<@ if not @{ ?filter } @>class="active"<@ end @>
				>
					All
				</a>
				<@ foreach in filters @>
					<a
						href="?filter=@{ :filter }"
						<@ if @{ ?filter } = @{ :filter } @>class="active"<@ end @>
					>@{ :filter }</a>
				<@ end @>
			</div>

			<#

			The grid template for the page previews can also be used here to render the actual pagelist content.

			#>
			<@ blocks/pagelist/grid.php @>

			<#

			At the end we also add the pagination component.

			#>
			<@ components/pagination.php @>
		</section>
	</main>
<@~ end ~@>
`,

	"lib/functions.php": `<?php

use Automad\Core\Automad;

/**
 * This is a simple example how to define little helper function with plain PHP
 * that can be easily used inside templates.
 *
 * The "icon" function can be used in templates like "<@ icon { name: '...' } @>".
 */
func('icon', function (array $options, Automad $Automad): string {
	$icon = file_get_contents(__DIR__ . '/../icons/' . $options['name'] . '.svg');

	return $icon;
});
`,

	"components/page.php": `<#

This is the base page template that defines the main document structure.
The first thing that happens here is including the helper functions file.
You can define all sorts of helper in there.

#>
<@~ ../lib/functions.php ~@>

<#

Now we set the ":colorTheme" class that is added to the <html> element.

The actual option set for the "selectColorTheme" variable is defined in
the "theme.json" file under "options".

#>
<@~ if @{ selectColorTheme | def ('switcher') } != 'switcher' @>
	<@~ set { :colorTheme: ' @{ selectColorTheme }' } @>
<@~ end ~@>

<!DOCTYPE html>
<html lang="en" class="@{ template | sanitize }@{ :colorTheme }">
<head>
	<meta name="viewport" content="width=device-width, initial-scale=1">

	<#

	The "@{ theme }" variable can be used to automatically build the correct path to your theme
	directory even, when renaming it.

	#>
	<link href="/packages/@{ theme }/dist/main.css" rel="stylesheet">
	<script src="/packages/@{ theme }/dist/main.js" type="text/javascript"></script>
</head>
<body>
	<#

	The layout is included where the main content goes.

	#>
	<@ layout.php @>
</body>
</html>
`,

	"components/content.php": `<#

This is a reusable snippet for rendering the main content of a page.

#>

<#

We can wrap the entire title section and assign the "am-block" class in order
to nicely integrate the content with the content that is provided
by the "+main" block editor field below.

#>
<section class="am-block">
	<h1>@{ title }</h1>
	<@ if @{ date } or @{ tags } @>
		<div class="kit-subtitle">
			<#

			The next block formats a page date based on the selected format string
			and wraps it inside a <p> tag. This way we can add the wrapping <p>
			ONLY whenever the formatted date is NOT an empty string and therefore
			avoid empty lines showing up under the title.

			Also note the "selectPageDateFormat" variable. All variables using the
			"select*" prefix must have the actual option set defined inside the
			"theme.json" file under "options".

			#>
			@{ date |
				dateFormat (@{ selectPageDateFormat | def('M Y') }, @{ :lang }) |
				replace('/(.+)/', '<p>$1</p>')
			}

			<@ if @{ tags } @>
				<p>
					<#

					We can iterate all page tags by using the
					"foreach in tags" statement. ":i" is the current index
					inside that tags array and ":tag" refers to the current tag.

					#>
					<@ foreach in tags ~@>
						<@ if @{ :i } > 1 @>, <@ end @>@{ :tag }
					<@~ end @>
				</p>
			<@ end @>
		</div>
	<@ end @>
</section>

@{ +main }
`,

	"components/pagination.php": `<#

This is the pagination component.
The ":paginationCount" runtime variable can be used to test if
there is more than on page of items.

#>
<@ if @{ :paginationCount } > 1 @>
	<nav class="kit-pagination">
		<#

		We can use the "set" statement in order to define a runtime variable called ":page" that
		automatically combines a given value for the "page" parameter in the query string (?page)
		and a default value of 1. This makes reusing the current page number much shorter and easier.

		#>
		<@ set { :page: @{ ?page | def (1) } } @>
			<@ if @{ :page } > 1 @>
			<#

			The "queryStringMerge" statement is used to merge as set of parameters into the current
			query string. So here for example we want to update only the "page" parameter while not touching other
			potentially existing parameters.

			#>
			<a href="?<@ queryStringMerge { page: 1 } @>">
				<#

				The "icon" function is a custom helper function that is define in "lib/functions.php"
				and provides an easy way of embedding SVG files.

				#>
				<@ icon { name: 'chevron-double-left' } @>
			</a>
			<a href="?<@ queryStringMerge { page: @{ :page | -1 } } @>">
				<@ icon { name: 'chevron-left' } @>
			</a>
		<@ end @>
			<@ for @{ :page | -3 } to @{ :page | +3 } @>
				<@ if @{ :i } > 0 and @{ :i } <= @{ :paginationCount } @>
					<@ if @{ :i } = @{ :page } @>
						<span>@{ :i }</span>
					<@ else @>
						<a href="?<@ queryStringMerge { page: @{ :i } } @>">@{ :i }</a>
					<@ end @>
				<@ end @>
			<@ end @>
		<@ if @{ :page } < @{ :paginationCount } @>
			<a href="?<@ queryStringMerge { page: @{ :page | +1 } } @>">
				<@ icon { name: 'chevron-right' } @>
			</a>
			<a href="?<@ queryStringMerge { page: @{ :paginationCount } } @>">
				<@ icon { name: 'chevron-double-right' } @>
			</a>
		<@ end @>
	</nav>
<@ end @>
`,

	"blocks/pagelist/grid.php": `<#

This snippet file is responsible to render any given pagelist into a grid of previews
consisting of the first found image inside the main content area and the page's title.

#>

<@ if @{ :pagelistCount } @>
	<section class="kit-pagelist">
		<#

		All pages in a defined pagelist are iterated here.
		Pagelists can be defined using the "newPagelist" statement.
		An example can be found in the "pagelist.php" template in
		the root of the repository.

		#>
		<@ foreach in pagelist @>
			<#

			The ":img" variable is used in order to store the file path of the first found
			image in the "+main" field.

			Note that the leading colon ":" is used to mark a variable as an internal runtime
			variable that is not exposed to the user interface in the dashboard.

			#>
			<@ set { :img: @{ +main | findFirstImage | def(false) } } @>

			<a href="@{ url }">
				<#

				The "with" statement can be used to "do something" with a file or page
				as long it exists. So here we want to create a small preview image with a maximum
				width of 500px. The resized image can be referenced using the ":fileResized"
				runtime variable.

				#>
				<@ with @{ :img } { width: 500 } @>
					<img src="@{ :fileResized }" />
				<@ end @>

				<div><strong>@{ title }</strong></div>
				<#

				As preview text we can extract the first paragraph of text of the "+main"
				field using the "findFirstParagraph" function and shorten it then to 100 characters.

				#>
				<p>@{ +main | findFirstParagraph | 180 }</p>
			</a>
		<@ end @>
	</section>
<@ end @>
`,
}

// fallbackTopLevel is a snapshot of the repository's structure, used by
// ListFiles only when the live Git Trees API request fails. It reflects the
// top-level layout at the time this package was written and may be missing
// files added upstream since.
var fallbackTopLevel = []TreeEntry{
	{Path: "bin", Type: "tree"},
	{Path: "blocks", Type: "tree"},
	{Path: "blocks/pagelist", Type: "tree"},
	{Path: "blocks/pagelist/grid.php", Type: "blob"},
	{Path: "client", Type: "tree"},
	{Path: "client/styles", Type: "tree"},
	{Path: "client/styles/index.less", Type: "blob"},
	{Path: "client/index.ts", Type: "blob"},
	{Path: "components", Type: "tree"},
	{Path: "components/content.php", Type: "blob"},
	{Path: "components/page.php", Type: "blob"},
	{Path: "components/pagination.php", Type: "blob"},
	{Path: "i18n", Type: "tree"},
	{Path: "icons", Type: "tree"},
	{Path: "lib", Type: "tree"},
	{Path: "lib/functions.php", Type: "blob"},
	{Path: ".editorconfig", Type: "blob"},
	{Path: ".env.example", Type: "blob"},
	{Path: ".gitignore", Type: "blob"},
	{Path: ".php-cs-fixer.php", Type: "blob"},
	{Path: "LICENSE", Type: "blob"},
	{Path: "README.md", Type: "blob"},
	{Path: "composer.json", Type: "blob"},
	{Path: "default.php", Type: "blob"},
	{Path: "esbuild.js", Type: "blob"},
	{Path: "package-lock.json", Type: "blob"},
	{Path: "package.json", Type: "blob"},
	{Path: "page_not_found.php", Type: "blob"},
	{Path: "pagelist.php", Type: "blob"},
	{Path: "postcss.config.js", Type: "blob"},
	{Path: "theme.json", Type: "blob"},
	{Path: "tsconfig.json", Type: "blob"},
}

// fallbackTree returns a Tree built from fallbackTopLevel.
func fallbackTree() *Tree {
	return &Tree{Entries: fallbackTopLevel}
}
