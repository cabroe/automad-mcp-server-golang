package automad

import (
	"context"
	"net/url"
	"regexp"
	"strings"
)

// PagesInput selects a page action and carries its parameters. Optional scalar
// fields use pointers so "unset" is distinguishable from a zero value.
type PagesInput struct {
	Action string

	// URL is the page the action targets (a slug like "/blog/post").
	URL string
	// TargetURL is the destination parent for create and move.
	TargetURL string

	Title    string
	Template string
	Private  *bool
	Tags     []string
	// Fields are arbitrary additional page data fields to set on create/update.
	Fields map[string]any
	// Publish controls whether create/update publishes; nil means publish.
	Publish *bool
	// Layout is a JSON array of sibling URLs controlling move ordering.
	Layout string
	// HistoryID selects a revision for history_restore.
	HistoryID string

	ConfirmToken string
}

var (
	actPageGet       = action{name: "pages.get", readOnly: true}
	actPageRecent    = action{name: "pages.recent", readOnly: true}
	actPageState     = action{name: "pages.publication_state", readOnly: true}
	actPageBreadcrmb = action{name: "pages.breadcrumbs", readOnly: true}
	actPageHistory   = action{name: "pages.history", readOnly: true}
	actTrashList     = action{name: "pages.trash_list", readOnly: true}

	actPageCreate    = action{name: "pages.create"}
	actPageUpdate    = action{name: "pages.update"}
	actPagePublish   = action{name: "pages.publish"}
	actPageDuplicate = action{name: "pages.duplicate"}
	actTrashRestore  = action{name: "pages.trash_restore"}

	actPageDelete   = action{name: "pages.delete", destructive: true}
	actPageMove     = action{name: "pages.move", destructive: true}
	actPageDiscard  = action{name: "pages.discard_draft", destructive: true}
	actHistoryRest  = action{name: "pages.history_restore", destructive: true}
	actTrashPurge   = action{name: "pages.trash_permanently_delete", destructive: true}
	actTrashClear   = action{name: "pages.trash_clear", destructive: true}
	trashPathMarker = "/.trash/"
)

// Pages performs a page operation against the live instance.
func (s *Service) Pages(ctx context.Context, in PagesInput) (any, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	switch in.Action {
	case "get":
		if err := requireURL(in.URL, "get"); err != nil {
			return nil, err
		}
		if _, err := s.gate(actPageGet, in.URL, in.ConfirmToken); err != nil {
			return nil, err
		}
		return s.client.post(ctx, APIBase+"/page/data", map[string]any{"url": normalizeURL(in.URL)})

	case "recent", "list":
		if _, err := s.gate(actPageRecent, "/", in.ConfirmToken); err != nil {
			return nil, err
		}
		return s.client.post(ctx, APIBase+"/page-collection/get-recently-edited", map[string]any{})

	case "publication_state":
		if err := requireURL(in.URL, "publication_state"); err != nil {
			return nil, err
		}
		if _, err := s.gate(actPageState, in.URL, in.ConfirmToken); err != nil {
			return nil, err
		}
		return s.client.post(ctx, APIBase+"/page/get-publication-state", map[string]any{"url": normalizeURL(in.URL)})

	case "breadcrumbs":
		if err := requireURL(in.URL, "breadcrumbs"); err != nil {
			return nil, err
		}
		if _, err := s.gate(actPageBreadcrmb, in.URL, in.ConfirmToken); err != nil {
			return nil, err
		}
		return s.client.post(ctx, APIBase+"/page/breadcrumbs", map[string]any{"url": normalizeURL(in.URL)})

	case "history":
		if err := requireURL(in.URL, "history"); err != nil {
			return nil, err
		}
		if _, err := s.gate(actPageHistory, in.URL, in.ConfirmToken); err != nil {
			return nil, err
		}
		return s.client.post(ctx, APIBase+"/history/log", map[string]any{"url": normalizeURL(in.URL)})

	case "trash_list":
		if _, err := s.gate(actTrashList, "/", in.ConfirmToken); err != nil {
			return nil, err
		}
		return s.client.post(ctx, APIBase+"/page-trash/list", map[string]any{})

	case "create":
		return s.createPage(ctx, in)

	case "update":
		return s.updatePage(ctx, in)

	case "publish":
		if err := requireURL(in.URL, "publish"); err != nil {
			return nil, err
		}
		if pending, err := s.gate(actPagePublish, in.URL, in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		return s.client.post(ctx, APIBase+"/page/publish", map[string]any{"url": normalizeURL(in.URL)})

	case "duplicate":
		if err := requireURL(in.URL, "duplicate"); err != nil {
			return nil, err
		}
		if pending, err := s.gate(actPageDuplicate, in.URL, in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		return s.client.post(ctx, APIBase+"/page/duplicate", map[string]any{"url": normalizeURL(in.URL)})

	case "discard_draft":
		if err := requireURL(in.URL, "discard_draft"); err != nil {
			return nil, err
		}
		if pending, err := s.gate(actPageDiscard, in.URL, in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		return s.client.post(ctx, APIBase+"/page/discard-draft", map[string]any{"url": normalizeURL(in.URL)})

	case "delete":
		return s.deletePage(ctx, in)

	case "move":
		return s.movePage(ctx, in)

	case "history_restore":
		if err := requireURL(in.URL, "history_restore"); err != nil {
			return nil, err
		}
		if strings.TrimSpace(in.HistoryID) == "" {
			return nil, validationError("history_id is required for history_restore")
		}
		if pending, err := s.gate(actHistoryRest, in.URL, in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		return s.client.post(ctx, APIBase+"/history/restore", map[string]any{"url": normalizeURL(in.URL), "logId": in.HistoryID})

	case "trash_restore":
		path, err := trashPath(in.URL, "trash_restore")
		if err != nil {
			return nil, err
		}
		if pending, err := s.gate(actTrashRestore, path, in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		return s.client.post(ctx, APIBase+"/page-trash/restore", map[string]any{"path": path})

	case "trash_permanently_delete":
		path, err := trashPath(in.URL, "trash_permanently_delete")
		if err != nil {
			return nil, err
		}
		if pending, err := s.gate(actTrashPurge, path, in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		return s.client.post(ctx, APIBase+"/page-trash/permanently-delete", map[string]any{"path": path})

	case "trash_clear":
		if pending, err := s.gate(actTrashClear, "/", in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		return s.client.post(ctx, APIBase+"/page-trash/clear", map[string]any{})

	default:
		return nil, validationError("unknown pages action %q", in.Action)
	}
}

func (s *Service) createPage(ctx context.Context, in PagesInput) (any, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, validationError("title is required for create")
	}
	parent := in.TargetURL
	if parent == "" {
		parent = in.URL
	}
	if strings.TrimSpace(parent) == "" {
		return nil, validationError("target_url (parent page) is required for create")
	}
	if pending, err := s.gate(actPageCreate, normalizeURL(parent), in.ConfirmToken); err != nil || pending != nil {
		return pending, err
	}

	payload := map[string]any{"targetPage": normalizeURL(parent), "title": in.Title}
	if in.Template != "" {
		payload["theme_template"] = in.Template
	}
	if in.Private != nil {
		payload["private"] = *in.Private
	}
	created, err := s.client.post(ctx, APIBase+"/page/add", payload)
	if err != nil {
		return nil, err
	}
	slug := slugFromResult(created)
	result := map[string]any{"ok": true, "created": created}
	if slug != "" {
		result["url"] = slug
		if in.Publish == nil || *in.Publish {
			published, perr := s.client.post(ctx, APIBase+"/page/publish", map[string]any{"url": slug})
			if perr != nil {
				result["published"] = false
				result["warning"] = "page created but publish failed: " + perr.Error()
			} else {
				result["published"] = true
				result["publish_result"] = published
			}
		}
	}
	return result, nil
}

// updatePage performs a full-replace save: v2 rejects a save without a title and
// resets any field not present in the payload, so the current record is read and
// the caller's changes merged on top, carrying the template forward.
func (s *Service) updatePage(ctx context.Context, in PagesInput) (any, error) {
	if err := requireURL(in.URL, "update"); err != nil {
		return nil, err
	}
	if pending, err := s.gate(actPageUpdate, in.URL, in.ConfirmToken); err != nil || pending != nil {
		return pending, err
	}
	page := normalizeURL(in.URL)

	storedFields, storedTemplate, err := s.readStoredPage(ctx, page)
	if err != nil {
		return nil, err
	}
	data := map[string]any{}
	for k, v := range storedFields {
		data[k] = v
	}
	if in.Title != "" {
		data["title"] = in.Title
	}
	if in.Private != nil {
		data["private"] = *in.Private
	}
	if in.Tags != nil {
		var tags []string
		for _, t := range in.Tags {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}
		data["tags"] = strings.Join(tags, ",")
	}
	for k, v := range in.Fields {
		data[k] = v
	}
	if title, _ := data["title"].(string); strings.TrimSpace(title) == "" {
		return nil, validationError("cannot update %s: the page has no title and none was supplied (v2 requires one on every save)", in.URL)
	}

	payload := map[string]any{"url": page, "data": data}
	template := in.Template
	if template == "" {
		template = storedTemplate
	}
	if template != "" {
		payload["theme_template"] = template
	}
	saved, err := s.client.post(ctx, APIBase+"/page/data", payload)
	if err != nil {
		return nil, err
	}
	resultingURL := page
	if slug := slugFromResult(saved); slug != "" {
		resultingURL = slug
	}
	result := map[string]any{"ok": true, "url": resultingURL}
	if in.Publish == nil || *in.Publish {
		published, perr := s.client.post(ctx, APIBase+"/page/publish", map[string]any{"url": resultingURL})
		if perr != nil {
			result["published"] = false
			result["warning"] = "page saved but publish failed: " + perr.Error()
		} else {
			result["published"] = true
			result["publish_result"] = published
		}
	}
	return result, nil
}

func (s *Service) deletePage(ctx context.Context, in PagesInput) (any, error) {
	if err := requireURL(in.URL, "delete"); err != nil {
		return nil, err
	}
	if pending, err := s.gate(actPageDelete, in.URL, in.ConfirmToken); err != nil || pending != nil {
		return pending, err
	}
	deleted, err := s.client.post(ctx, APIBase+"/page/delete", map[string]any{"url": normalizeURL(in.URL)})
	if err != nil {
		return nil, err
	}
	// The deleted page keeps being served from cache otherwise.
	if _, cerr := s.client.post(ctx, APIBase+"/cache/clear", map[string]any{}); cerr != nil {
		return map[string]any{"deleted": deleted, "warning": "deleted but cache clear failed: " + cerr.Error()}, nil
	}
	return map[string]any{"deleted": deleted}, nil
}

func (s *Service) movePage(ctx context.Context, in PagesInput) (any, error) {
	if err := requireURL(in.URL, "move"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.TargetURL) == "" {
		return nil, validationError("target_url (destination parent page) is required for move")
	}
	if pending, err := s.gate(actPageMove, in.URL, in.ConfirmToken); err != nil || pending != nil {
		return pending, err
	}
	payload := map[string]any{"url": normalizeURL(in.URL), "targetPage": normalizeURL(in.TargetURL)}
	if in.Layout != "" {
		payload["layout"] = in.Layout
	}
	return s.client.post(ctx, APIBase+"/page/move", payload)
}

// readStoredPage returns a page's current fields (declared + unused) and its
// template id, so an update can merge onto the full record.
func (s *Service) readStoredPage(ctx context.Context, page string) (map[string]any, string, error) {
	raw, err := s.client.post(ctx, APIBase+"/page/data", map[string]any{"url": page})
	if err != nil {
		return nil, "", err
	}
	rec, _ := raw.(map[string]any)
	fields := map[string]any{}
	if f, ok := rec["fields"].(map[string]any); ok {
		for k, v := range f {
			fields[k] = v
		}
	}
	if u, ok := rec["unused"].(map[string]any); ok {
		for k, v := range u {
			fields[k] = v
		}
	}
	template := ""
	if t, ok := rec["template"].(string); ok {
		template = templateIDFromPath(t)
	}
	return fields, template, nil
}

func requireURL(u, action string) error {
	if strings.TrimSpace(u) == "" {
		return validationError("url is required for %s", action)
	}
	return nil
}

// normalizeURL ensures a page slug is rooted with a leading slash.
func normalizeURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return u
	}
	if !strings.HasPrefix(u, "/") {
		return "/" + u
	}
	return u
}

// trashPath validates that a value addresses a page inside v2's trash.
func trashPath(value, action string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", validationError("url (trash path) is required for %s", action)
	}
	if !strings.Contains(value, trashPathMarker) {
		return "", validationError("%s expects a trash path (containing %q) as returned by trash_list, got %q", action, trashPathMarker, value)
	}
	return value, nil
}

var slugRE = regexp.MustCompile(`[?&]url=([^&]+)`)

// slugFromResult extracts a page slug from a v2 result's redirect/slug field.
func slugFromResult(result any) string {
	m, ok := result.(map[string]any)
	if !ok {
		return ""
	}
	if slug, ok := m["slug"].(string); ok && slug != "" {
		return normalizeURL(slug)
	}
	redirect, ok := m["redirect"].(string)
	if !ok {
		return ""
	}
	match := slugRE.FindStringSubmatch(redirect)
	if match == nil {
		return ""
	}
	decoded, err := url.QueryUnescape(match[1])
	if err != nil {
		return ""
	}
	return normalizeURL(decoded)
}

// templateIDFromPath converts v2's absolute template path
// (/app/packages/automad/standard-lite/home.php) to its id form
// (automad/standard-lite/home). Returns "" when no template is selected
// (v2 stores that as a path with an empty basename, e.g. ".../standard-lite/.php").
func templateIDFromPath(p string) string {
	const marker = "/packages/"
	idx := strings.Index(p, marker)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSuffix(p[idx+len(marker):], ".php")
	if rest == "" || strings.HasSuffix(rest, "/") {
		return ""
	}
	return rest
}
