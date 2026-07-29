// automad_tools.go registers MCP tools that bridge to a live Automad v2
// instance through its dashboard JSON API (/_api). These tools manage real site
// content — pages, media, shared data, config/cache, and Composer packages —
// so an agent can operate a running site end to end rather than only reading
// about it. The bridge is active only when AUTOMAD_URL/USER/PASS are set; see
// internal/automad for the auth, envelope, and write-guard model.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cabroe/automad-mcp-server-golang/internal/automad"
)

// pagesToolInput is the input schema for the automad_pages tool.
type pagesToolInput struct {
	Action       string         `json:"action" jsonschema:"The page operation: get, list, create, update, delete, move, duplicate, publish, discard_draft, publication_state, breadcrumbs, history, history_restore, trash_list, trash_restore, trash_permanently_delete, trash_clear"`
	URL          string         `json:"url,omitempty" jsonschema:"The page slug the action targets (e.g. '/blog/post'). For trash actions this is the trash path returned by trash_list."`
	TargetURL    string         `json:"target_url,omitempty" jsonschema:"Destination parent page for create and move (e.g. '/')."`
	Title        string         `json:"title,omitempty" jsonschema:"Page title for create and update."`
	Template     string         `json:"template,omitempty" jsonschema:"Template id in 'package/name' form (e.g. 'automad/standard-lite/home'), as returned by get. Carried forward on update when omitted. Setting it also pins the page to that package's theme, so the page stops following later site-wide theme changes."`
	Private      *bool          `json:"private,omitempty" jsonschema:"Whether the page is private (draft-like visibility)."`
	Tags         []string       `json:"tags,omitempty" jsonschema:"Tags to set on update."`
	Fields       map[string]any `json:"fields,omitempty" jsonschema:"Arbitrary additional page data fields to set on create or update."`
	Publish      *bool          `json:"publish,omitempty" jsonschema:"Whether to publish after create/update. Defaults to true."`
	Layout       string         `json:"layout,omitempty" jsonschema:"JSON array of sibling URL strings controlling order for move."`
	HistoryID    string         `json:"history_id,omitempty" jsonschema:"Revision id for history_restore (from the history action)."`
	ConfirmToken string         `json:"confirm_token,omitempty" jsonschema:"Confirmation token returned by a prior destructive call; required to execute destructive actions in confirm-destructive mode."`
}

// mediaToolInput is the input schema for the automad_media tool.
type mediaToolInput struct {
	Action       string `json:"action" jsonschema:"The media operation: list, upload, import, delete"`
	URL          string `json:"url,omitempty" jsonschema:"The page directory the file belongs to; empty or '/' targets the shared media collection."`
	Filename     string `json:"filename,omitempty" jsonschema:"Plain file name (no path separators) for upload and delete."`
	MimeType     string `json:"mime_type,omitempty" jsonschema:"Content type for upload (e.g. 'image/png')."`
	DataBase64   string `json:"data_base64,omitempty" jsonschema:"Base64-encoded file content for upload."`
	ImportURL    string `json:"import_url,omitempty" jsonschema:"http(s) URL for the server-side import action."`
	ConfirmToken string `json:"confirm_token,omitempty" jsonschema:"Confirmation token for destructive actions (delete)."`
}

// sharedToolInput is the input schema for the automad_shared tool.
type sharedToolInput struct {
	Action       string         `json:"action" jsonschema:"The shared-data operation: get, set, publish, discard_draft, publication_state"`
	Fields       map[string]any `json:"fields,omitempty" jsonschema:"Field name/value pairs to set (required for set). Only these change; the rest of the shared data is preserved. An empty string clears a field."`
	Publish      *bool          `json:"publish,omitempty" jsonschema:"Whether to publish after set. Defaults to true; with false the change stays a draft that visitors do not see."`
	ConfirmToken string         `json:"confirm_token,omitempty" jsonschema:"Confirmation token for destructive actions (discard_draft)."`
}

// configToolInput is the input schema for the automad_config tool.
type configToolInput struct {
	Action       string         `json:"action" jsonschema:"The config/cache operation: get, update, cache_clear, cache_purge"`
	Type         string         `json:"type,omitempty" jsonschema:"Config section/type for update."`
	Payload      map[string]any `json:"payload,omitempty" jsonschema:"Config values to write for update."`
	ConfirmToken string         `json:"confirm_token,omitempty" jsonschema:"Confirmation token for destructive actions (cache_purge)."`
}

// packagesToolInput is the input schema for the automad_packages tool.
type packagesToolInput struct {
	Action       string `json:"action" jsonschema:"The package-manager operation: list_installed, outdated, update, update_all, uninstall"`
	Package      string `json:"package,omitempty" jsonschema:"Composer package name (e.g. 'automad/standard-lite') for update and uninstall."`
	ConfirmToken string `json:"confirm_token,omitempty" jsonschema:"Confirmation token for destructive actions (uninstall)."`
}

// RegisterAutomadTools adds the live Automad v2 API bridge tools to the server.
func RegisterAutomadTools(s *mcp.Server, svc *automad.Service) {
	registerPagesTool(s, svc)
	registerMediaTool(s, svc)
	registerSharedTool(s, svc)
	registerConfigTool(s, svc)
	registerPackagesTool(s, svc)
}

func registerPagesTool(s *mcp.Server, svc *automad.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "automad_pages",
		Description: `Create, read, update, and manage pages on a live Automad v2 site via its dashboard API.
Actions: get, list, create, update, delete, move, duplicate, publish, discard_draft, publication_state,
breadcrumbs, history, history_restore, trash_list, trash_restore, trash_permanently_delete, trash_clear.
update is a full-replace save that reads the current page and merges your changes, so partial updates are safe.
get reports template as its 'package/name' id, so a get result can be fed straight back into update.
Destructive actions (delete, move, discard_draft, history_restore, trash_permanently_delete, trash_clear)
return a confirm_token in confirm-destructive mode; re-run the same call with that token to execute.
Requires AUTOMAD_URL/USER/PASS.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in pagesToolInput) (*mcp.CallToolResult, any, error) {
		result, err := svc.Pages(ctx, automad.PagesInput{
			Action: in.Action, URL: in.URL, TargetURL: in.TargetURL, Title: in.Title,
			Template: in.Template, Private: in.Private, Tags: in.Tags, Fields: in.Fields,
			Publish: in.Publish, Layout: in.Layout, HistoryID: in.HistoryID, ConfirmToken: in.ConfirmToken,
		})
		return jsonResult(result, err)
	})
}

func registerMediaTool(s *mcp.Server, svc *automad.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "automad_media",
		Description: `Manage media files on a live Automad v2 site: list, upload (base64), import (from an http(s) URL), delete.
Files attach to a page directory (url) or the shared collection when url is empty/'/'.
delete returns a confirm_token in confirm-destructive mode; re-run with it to execute. Requires AUTOMAD_URL/USER/PASS.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mediaToolInput) (*mcp.CallToolResult, any, error) {
		result, err := svc.Media(ctx, automad.MediaInput{
			Action: in.Action, URL: in.URL, Filename: in.Filename, MimeType: in.MimeType,
			DataBase64: in.DataBase64, ImportURL: in.ImportURL, ConfirmToken: in.ConfirmToken,
		})
		return jsonResult(result, err)
	})
}

func registerSharedTool(s *mcp.Server, svc *automad.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "automad_shared",
		Description: `Read or write Automad's shared (site-wide) data fields, e.g. sitename or theme.
Actions: get, set, publish, discard_draft, publication_state.
set is a targeted write: it reads the stored data and merges your fields on top, so fields you do not pass are kept.
Shared writes land in a draft, so set publishes by default (pass publish=false to keep it a draft) and reports the
verified publication state. discard_draft is destructive and returns a confirm_token in confirm-destructive mode.
Requires AUTOMAD_URL/USER/PASS.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in sharedToolInput) (*mcp.CallToolResult, any, error) {
		result, err := svc.Shared(ctx, automad.SharedInput{
			Action: in.Action, Fields: in.Fields, Publish: in.Publish, ConfirmToken: in.ConfirmToken,
		})
		return jsonResult(result, err)
	})
}

func registerConfigTool(s *mcp.Server, svc *automad.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "automad_config",
		Description: `Inspect and control a live Automad v2 site's configuration and cache:
get (bootstrap/system info), update (write a config section), cache_clear, cache_purge.
get omits v2's dashboard-only keys ("text", the UI translations, and "languages"); they are named under "omitted".
cache_purge is destructive and returns a confirm_token in confirm-destructive mode. Requires AUTOMAD_URL/USER/PASS.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in configToolInput) (*mcp.CallToolResult, any, error) {
		result, err := svc.Config(ctx, automad.ConfigInput{
			Action: in.Action, Type: in.Type, Payload: in.Payload, ConfirmToken: in.ConfirmToken,
		})
		return jsonResult(result, err)
	})
}

func registerPackagesTool(s *mcp.Server, svc *automad.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "automad_packages",
		Description: `Manage installed Composer packages (themes and extensions) on a live Automad v2 site:
list_installed, outdated, update (one package), update_all, uninstall.
uninstall is destructive and returns a confirm_token in confirm-destructive mode. Requires AUTOMAD_URL/USER/PASS.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in packagesToolInput) (*mcp.CallToolResult, any, error) {
		result, err := svc.Packages(ctx, automad.PackagesInput{
			Action: in.Action, Package: in.Package, ConfirmToken: in.ConfirmToken,
		})
		return jsonResult(result, err)
	})
}

// jsonResult renders a bridge result as pretty JSON, or maps an *automad.APIError
// to a tool error the model can act on.
func jsonResult(result any, err error) (*mcp.CallToolResult, any, error) {
	if err != nil {
		var apiErr *automad.APIError
		if errors.As(err, &apiErr) {
			return toolError(fmt.Sprintf("[%s] %s", apiErr.Code, apiErr.Message)), nil, nil
		}
		return toolError(err.Error()), nil, nil
	}
	encoded, merr := json.MarshalIndent(result, "", "  ")
	if merr != nil {
		return toolError(fmt.Sprintf("failed to encode result: %v", merr)), nil, nil
	}
	return toolText(string(encoded)), nil, nil
}
