package automad

import (
	"context"
	"encoding/base64"
	"net/url"
	"strings"
)

// MediaInput selects a media action and carries its parameters.
type MediaInput struct {
	Action string
	// URL is the page directory the file belongs to; "" or "/" is the shared
	// (site-wide) media collection.
	URL string
	// Filename is the plain file name for upload/delete (no path separators).
	Filename string
	// MimeType is the content type for upload.
	MimeType string
	// DataBase64 is the base64-encoded file content for upload.
	DataBase64 string
	// ImportURL is an http(s) URL for the server-side import action.
	ImportURL string

	ConfirmToken string
}

var (
	actMediaList   = action{name: "media.list", readOnly: true}
	actMediaUpload = action{name: "media.upload"}
	actMediaImport = action{name: "media.import"}
	actMediaDelete = action{name: "media.delete", destructive: true}
)

// Media lists, uploads, imports, or deletes files on the live instance.
func (s *Service) Media(ctx context.Context, in MediaInput) (any, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	switch in.Action {
	case "list":
		if _, err := s.gate(actMediaList, in.URL, in.ConfirmToken); err != nil {
			return nil, err
		}
		return s.client.post(ctx, APIBase+"/file-collection/list", map[string]any{"url": in.URL})

	case "upload":
		if err := validateFilename(in.Filename); err != nil {
			return nil, err
		}
		if strings.TrimSpace(in.MimeType) == "" {
			return nil, validationError("mime_type is required for upload")
		}
		if strings.TrimSpace(in.DataBase64) == "" {
			return nil, validationError("data_base64 is required for upload")
		}
		data, err := base64.StdEncoding.DecodeString(in.DataBase64)
		if err != nil {
			return nil, validationError("data_base64 is not valid base64: %v", err)
		}
		if pending, gerr := s.gate(actMediaUpload, in.URL, in.ConfirmToken); gerr != nil || pending != nil {
			return pending, gerr
		}
		return s.client.upload(ctx, APIBase+"/file-collection/upload", in.URL, in.Filename, in.MimeType, data)

	case "import":
		if err := validateImportURL(in.ImportURL); err != nil {
			return nil, err
		}
		if pending, gerr := s.gate(actMediaImport, in.URL, in.ConfirmToken); gerr != nil || pending != nil {
			return pending, gerr
		}
		if _, err := s.client.post(ctx, APIBase+"/file/import", map[string]any{
			"importUrl": in.ImportURL,
			"url":       in.URL,
		}); err != nil {
			return nil, err
		}
		// v2 answers a successful import with an empty envelope; report the
		// request instead. The stored filename is sanitized server-side, so
		// callers read it back with list.
		return map[string]any{"ok": true, "import_url": in.ImportURL, "url": in.URL}, nil

	case "delete":
		if err := validateFilename(in.Filename); err != nil {
			return nil, err
		}
		if strings.TrimSpace(in.URL) == "" {
			return nil, validationError("url (the file's parent directory) is required for delete")
		}
		if pending, gerr := s.gate(actMediaDelete, in.URL+"/"+in.Filename, in.ConfirmToken); gerr != nil || pending != nil {
			return pending, gerr
		}
		return s.client.post(ctx, APIBase+"/file-collection/list", map[string]any{
			"url":      in.URL,
			"action":   "delete",
			"selected": map[string]any{in.Filename: true},
		})

	default:
		return nil, validationError("unknown media action %q (want: list, upload, import, delete)", in.Action)
	}
}

func validateFilename(name string) error {
	if strings.TrimSpace(name) == "" {
		return validationError("filename must not be empty")
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return validationError("filename must be a plain file name without path separators")
	}
	return nil
}

func validateImportURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return validationError("import_url is required for import")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return validationError("import_url is not a valid URL: %v", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return validationError("import_url must use http or https, got %q", parsed.Scheme)
	}
	return nil
}
