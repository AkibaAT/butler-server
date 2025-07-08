package handlers

import (
	"butler-server/auth"
	"butler-server/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type CoreHandlers struct {
	db models.Database
}

func NewCoreHandlers(db models.Database) *CoreHandlers {
	return &CoreHandlers{db: db}
}

// GET /profile - Get current user profile
func (h *CoreHandlers) GetProfile(w http.ResponseWriter, r *http.Request) {
	user := auth.MustGetUser(r.Context())

	response := map[string]interface{}{
		"user": map[string]interface{}{
			"id":           user.ID,
			"username":     user.Username,
			"display_name": user.DisplayName,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /profile/games - List games for current user
func (h *CoreHandlers) GetProfileGames(w http.ResponseWriter, r *http.Request) {
	user := auth.MustGetUser(r.Context())

	games, err := h.db.GetGamesByUserID(user.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"errors":["%s"]}`, err.Error()), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"games": games,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /games/{id} - Get game by ID
func (h *CoreHandlers) GetGame(w http.ResponseWriter, r *http.Request) {
	gameIDStr := mux.Vars(r)["id"]
	gameID, err := strconv.ParseInt(gameIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"errors":["invalid game id"]}`, http.StatusBadRequest)
		return
	}

	user, game, err := h.db.GetGameByID(gameID)
	if err != nil {
		http.Error(w, `{"errors":["game not found"]}`, http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"game": map[string]interface{}{
			"id":             game.ID,
			"title":          game.Title,
			"short_text":     game.ShortText,
			"type":           game.Type,
			"classification": game.Classification,
			"url":            game.URL,
			"user": map[string]interface{}{
				"id":           user.ID,
				"username":     user.Username,
				"display_name": user.DisplayName,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /games/{id}/uploads - List uploads for a game
func (h *CoreHandlers) GetGameUploads(w http.ResponseWriter, r *http.Request) {
	gameIDStr := mux.Vars(r)["id"]
	gameID, err := strconv.ParseInt(gameIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"errors":["invalid game id"]}`, http.StatusBadRequest)
		return
	}

	// Check if game exists
	_, _, err = h.db.GetGameByID(gameID)
	if err != nil {
		http.Error(w, `{"errors":["game not found"]}`, http.StatusNotFound)
		return
	}

	uploads, err := h.db.GetUploadsByGameID(gameID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"errors":["%s"]}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Convert uploads to response format
	var uploadsResponse []map[string]interface{}
	for _, upload := range uploads {
		uploadsResponse = append(uploadsResponse, map[string]interface{}{
			"id":           upload.ID,
			"filename":     upload.Filename,
			"display_name": upload.DisplayName,
			"size":         upload.Size,
			"storage":      upload.Storage,
			"type":         upload.Type,
			"platforms":    upload.Platforms,
		})
	}

	response := map[string]interface{}{
		"uploads": uploadsResponse,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /uploads/{id} - Get upload by ID
func (h *CoreHandlers) GetUpload(w http.ResponseWriter, r *http.Request) {
	uploadIDStr := mux.Vars(r)["id"]
	uploadID, err := strconv.ParseInt(uploadIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"errors":["invalid upload id"]}`, http.StatusBadRequest)
		return
	}

	upload, err := h.db.GetUploadByID(uploadID)
	if err != nil {
		http.Error(w, `{"errors":["upload not found"]}`, http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"upload": map[string]interface{}{
			"id":           upload.ID,
			"filename":     upload.Filename,
			"display_name": upload.DisplayName,
			"size":         upload.Size,
			"storage":      upload.Storage,
			"type":         upload.Type,
			"platforms":    upload.Platforms,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /uploads/{id}/builds - List builds for an upload
func (h *CoreHandlers) GetUploadBuilds(w http.ResponseWriter, r *http.Request) {
	uploadIDStr := mux.Vars(r)["id"]
	uploadID, err := strconv.ParseInt(uploadIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"errors":["invalid upload id"]}`, http.StatusBadRequest)
		return
	}

	// Check if upload exists
	_, err = h.db.GetUploadByID(uploadID)
	if err != nil {
		http.Error(w, `{"errors":["upload not found"]}`, http.StatusNotFound)
		return
	}

	builds, err := h.db.GetBuildsByUploadID(uploadID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"errors":["%s"]}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Convert builds to response format
	var buildsResponse []map[string]interface{}
	for _, build := range builds {
		buildData := map[string]interface{}{
			"id":           build.ID,
			"user_version": build.UserVersion,
			"state":        build.State,
			"created_at":   build.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}

		if build.ParentBuildID != nil {
			buildData["parent_build_id"] = *build.ParentBuildID
		}

		buildsResponse = append(buildsResponse, buildData)
	}

	response := map[string]interface{}{
		"builds": buildsResponse,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /builds/{id} - Get build by ID
func (h *CoreHandlers) GetBuild(w http.ResponseWriter, r *http.Request) {
	buildIDStr := mux.Vars(r)["id"]
	buildID, err := strconv.ParseInt(buildIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"errors":["invalid build id"]}`, http.StatusBadRequest)
		return
	}

	build, err := h.db.GetBuildByID(buildID)
	if err != nil {
		http.Error(w, `{"errors":["build not found"]}`, http.StatusNotFound)
		return
	}

	buildData := map[string]interface{}{
		"id":           build.ID,
		"upload_id":    build.UploadID,
		"user_version": build.UserVersion,
		"state":        build.State,
		"created_at":   build.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if build.ParentBuildID != nil {
		buildData["parent_build_id"] = *build.ParentBuildID
	}

	response := map[string]interface{}{
		"build": buildData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /uploads/{id}/download - Generate download URL for upload
func (h *CoreHandlers) GetUploadDownload(w http.ResponseWriter, r *http.Request) {
	uploadIDStr := mux.Vars(r)["id"]
	uploadID, err := strconv.ParseInt(uploadIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"errors":["invalid upload id"]}`, http.StatusBadRequest)
		return
	}

	upload, err := h.db.GetUploadByID(uploadID)
	if err != nil {
		http.Error(w, `{"errors":["upload not found"]}`, http.StatusNotFound)
		return
	}

	// For now, just generate a simple download URL
	// In a real implementation, this would be a signed URL with expiration
	downloadURL := fmt.Sprintf("http://localhost:8080/downloads/uploads/%d/%s", upload.ID, upload.Filename)

	response := map[string]interface{}{
		"url": downloadURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
