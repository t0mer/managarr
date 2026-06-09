// internal/api/backup.go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/t0mer/galactica/internal/providers"
	"github.com/t0mer/galactica/internal/storage"
)

var unsafeNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// safeName strips characters that could be used for path traversal from a name
// intended for use as a filesystem path component.
func safeName(s string) string {
	return strings.Trim(unsafeNameChars.ReplaceAllString(s, "_"), "._")
}

// containedIn returns true if target, after cleaning, is strictly under root.
func containedIn(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	return strings.HasPrefix(target, root+string(os.PathSeparator))
}

// BackupHandler handles /api/v1/backup routes.
type BackupHandler struct{ *Deps }

type backupTargetResp struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	RetentionDays int    `json:"retention_days"`
	Enabled       bool   `json:"enabled"`
	CreatedAt     string `json:"created_at"`
}

type backupResp struct {
	ID         string `json:"id"`
	TargetID   string `json:"target_id"`
	InstanceID string `json:"instance_id"`
	TS         string `json:"ts"`
	SizeBytes  int64  `json:"size_bytes"`
	Status     string `json:"status"`
	Location   string `json:"location,omitempty"`
	Error      string `json:"error,omitempty"`
}

func toBackupTargetResp(r storage.BackupTargetRow) backupTargetResp {
	return backupTargetResp{
		ID:            r.ID,
		Name:          r.Name,
		Type:          r.Type,
		RetentionDays: r.RetentionDays,
		Enabled:       r.Enabled,
		CreatedAt:     r.CreatedAt.Format(time.RFC3339),
	}
}

func toBackupResp(r storage.BackupRow) backupResp {
	return backupResp{
		ID:         r.ID,
		TargetID:   r.TargetID,
		InstanceID: r.InstanceID,
		TS:         r.TS.Format(time.RFC3339),
		SizeBytes:  r.SizeBytes,
		Status:     r.Status,
		Location:   r.Location,
		Error:      r.Error,
	}
}

// ListTargets handles GET /api/v1/backup/targets.
func (h *BackupHandler) ListTargets(w http.ResponseWriter, r *http.Request) {
	targets, err := storage.ListBackupTargets(h.DB)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]backupTargetResp, len(targets))
	for i, t := range targets {
		out[i] = toBackupTargetResp(t)
	}
	jsonResponse(w, http.StatusOK, out)
}

type createTargetReq struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Path          string `json:"path"`
	RetentionDays int    `json:"retention_days"`
	Enabled       bool   `json:"enabled"`
}

// CreateTarget handles POST /api/v1/backup/targets.
func (h *BackupHandler) CreateTarget(w http.ResponseWriter, r *http.Request) {
	var req createTargetReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" || req.Path == "" {
		jsonError(w, http.StatusBadRequest, "name and path are required")
		return
	}
	if req.Type == "" {
		req.Type = "local"
	}
	if req.RetentionDays <= 0 {
		req.RetentionDays = 30
	}

	// Store path in config_encrypted field (unencrypted for local paths in v1).
	configJSON, _ := json.Marshal(map[string]string{"path": req.Path})

	id := uuid.New().String()
	row := storage.BackupTargetRow{
		ID:              id,
		Name:            req.Name,
		Type:            req.Type,
		ConfigEncrypted: configJSON,
		RetentionDays:   req.RetentionDays,
		Enabled:         req.Enabled,
	}
	if err := storage.InsertBackupTarget(h.DB, row); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	saved, err := storage.GetBackupTarget(h.DB, id)
	if err != nil || saved == nil {
		jsonError(w, http.StatusInternalServerError, "target created but could not be retrieved")
		return
	}
	jsonResponse(w, http.StatusCreated, toBackupTargetResp(*saved))
}

// DeleteTarget handles DELETE /api/v1/backup/targets/{id}.
func (h *BackupHandler) DeleteTarget(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := storage.GetBackupTarget(h.DB, id)
	if err != nil || existing == nil {
		jsonError(w, http.StatusNotFound, "target not found")
		return
	}
	if err := storage.DeleteBackupTarget(h.DB, id); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type runBackupReq struct {
	TargetID   string `json:"target_id"`
	InstanceID string `json:"instance_id"`
}

// RunBackup handles POST /api/v1/backup/run.
func (h *BackupHandler) RunBackup(w http.ResponseWriter, r *http.Request) {
	var req runBackupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.TargetID == "" || req.InstanceID == "" {
		jsonError(w, http.StatusBadRequest, "target_id and instance_id are required")
		return
	}
	target, err := storage.GetBackupTarget(h.DB, req.TargetID)
	if err != nil || target == nil {
		jsonError(w, http.StatusNotFound, "target not found")
		return
	}
	inst, err := h.resolve(r.Context(), req.InstanceID)
	if err != nil {
		jsonError(w, http.StatusNotFound, "instance not found")
		return
	}
	p, err := providers.Get(inst.Kind)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "provider not registered")
		return
	}
	cb, ok := p.(providers.ConfigBackup)
	if !ok {
		jsonError(w, http.StatusBadRequest, "provider does not support config backup")
		return
	}

	// Extract path from target config.
	var cfg map[string]string
	_ = json.Unmarshal(target.ConfigEncrypted, &cfg)
	backupRoot := filepath.Clean(cfg["path"])
	if backupRoot == "" || backupRoot == "." {
		jsonError(w, http.StatusInternalServerError, "backup target has no path configured")
		return
	}

	backupID := uuid.New().String()
	now := time.Now()
	_ = storage.InsertBackup(h.DB, storage.BackupRow{
		ID:         backupID,
		TargetID:   req.TargetID,
		InstanceID: req.InstanceID,
		TS:         now,
		Status:     "pending",
	})

	// Run asynchronously.
	go func() {
		blob, err := cb.ExportConfig(r.Context(), inst)
		if err != nil {
			_ = storage.UpdateBackupStatus(h.DB, backupID, "error", "", err.Error(), 0)
			return
		}
		// Sanitize path components to prevent traversal.
		safeID := safeName(inst.ID)
		safeName_ := safeName(inst.Name)
		if safeID == "" {
			safeID = "unknown"
		}
		if safeName_ == "" {
			safeName_ = "instance"
		}
		dir := filepath.Join(backupRoot, safeID)
		if !containedIn(backupRoot, dir) {
			_ = storage.UpdateBackupStatus(h.DB, backupID, "error", "", "computed dir escapes backup root", 0)
			return
		}
		if err := os.MkdirAll(dir, 0o750); err != nil {
			_ = storage.UpdateBackupStatus(h.DB, backupID, "error", "", fmt.Sprintf("mkdir: %v", err), 0)
			return
		}
		fileName := fmt.Sprintf("%s-%s.json", safeName_, now.Format("20060102T150405Z"))
		fullPath := filepath.Join(dir, fileName)
		if !containedIn(backupRoot, fullPath) {
			_ = storage.UpdateBackupStatus(h.DB, backupID, "error", "", "computed path escapes backup root", 0)
			return
		}
		if err := os.WriteFile(fullPath, blob.Data, 0o640); err != nil {
			_ = storage.UpdateBackupStatus(h.DB, backupID, "error", "", fmt.Sprintf("write: %v", err), 0)
			return
		}
		// Store only the relative filename, not the full server path, to avoid disclosing server layout.
		relPath, relErr := filepath.Rel(backupRoot, fullPath)
		if relErr != nil {
			relPath = fileName
		}
		_ = storage.UpdateBackupStatus(h.DB, backupID, "success", relPath, "", int64(len(blob.Data)))
	}()

	jsonResponse(w, http.StatusAccepted, map[string]string{"backup_id": backupID, "status": "pending"})
}

// ListBackups handles GET /api/v1/backup/targets/{id}/backups.
func (h *BackupHandler) ListBackups(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")
	rows, err := storage.ListBackups(h.DB, targetID, 50)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]backupResp, len(rows))
	for i, row := range rows {
		out[i] = toBackupResp(row)
	}
	jsonResponse(w, http.StatusOK, out)
}
