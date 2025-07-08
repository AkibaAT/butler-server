package handlers

import (
	"archive/zip"
	"butler-server/auth"
	"butler-server/models"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
)

// validateNamespaceAccess checks if the user can access the given namespace
func (h *WharfHandlers) validateNamespaceAccess(user *models.User, namespace string) error {
	if !user.CanAccessNamespace(namespace) {
		return fmt.Errorf("access denied: user '%s' cannot access namespace '%s'", user.Username, namespace)
	}
	return nil
}

type WharfHandlers struct {
	db          models.Database
	minioClient *minio.Client
	bucketName  string
}

func NewWharfHandlers(db models.Database, minioClient *minio.Client, bucketName string) *WharfHandlers {
	return &WharfHandlers{db: db, minioClient: minioClient, bucketName: bucketName}
}

// MinIO helper methods
func (h *WharfHandlers) GetPresignedUploadURL(objectName string, expiry time.Duration) (string, error) {
	ctx := context.Background()
	presignedURL, err := h.minioClient.PresignedPutObject(ctx, h.bucketName, objectName, expiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL: %v", err)
	}
	return presignedURL.String(), nil
}

func (h *WharfHandlers) FileExists(objectName string) bool {
	ctx := context.Background()
	_, err := h.minioClient.StatObject(ctx, h.bucketName, objectName, minio.StatObjectOptions{})
	return err == nil
}

func (h *WharfHandlers) GetFileSize(objectName string) (int64, error) {
	ctx := context.Background()
	stat, err := h.minioClient.StatObject(ctx, h.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to get object stat: %v", err)
	}
	return stat.Size, nil
}

func (h *WharfHandlers) GetSignedURL(objectName string, expiry time.Duration) (string, error) {
	ctx := context.Background()
	presignedURL, err := h.minioClient.PresignedGetObject(ctx, h.bucketName, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %v", err)
	}
	return presignedURL.String(), nil
}

// GET /wharf/status - Check wharf infrastructure status
func (h *WharfHandlers) GetWharfStatus(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"ok": true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /wharf/channels - List all channels for a target
func (h *WharfHandlers) ListChannels(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")

	if target == "" {
		http.Error(w, `{"errors":["missing build target (need game_id or target)"]}`, http.StatusBadRequest)
		return
	}

	// Parse target format: "username/gamename"
	parts := strings.Split(target, "/")
	if len(parts) != 2 {
		http.Error(w, `{"errors":["invalid target format, expected username/gamename"]}`, http.StatusBadRequest)
		return
	}

	username := parts[0]
	gamename := parts[1]

	// Get user from context (set by auth middleware)
	user := auth.MustGetUser(r.Context())

	// Validate namespace access
	err := h.validateNamespaceAccess(user, username)
	if err != nil {
		fmt.Printf("Namespace access denied: %v\n", err)
		http.Error(w, `{"errors":["access denied"]}`, http.StatusForbidden)
		return
	}

	// Note: User and namespace validation already done above

	// Find the game
	game, err := h.db.GetGameByUserAndTitle(user.ID, gamename)
	if err != nil {
		http.Error(w, `{"errors":["game not found"]}`, http.StatusNotFound)
		return
	}

	// Get all uploads for this game
	uploads, err := h.db.GetUploadsByGameID(game.ID)
	if err != nil {
		http.Error(w, `{"errors":["failed to get uploads"]}`, http.StatusInternalServerError)
		return
	}

	// Build channels response
	channels := make(map[string]interface{})

	for _, upload := range uploads {
		// Get actual channels for this upload from the channels table
		uploadChannels, err := h.db.GetChannelsByUploadID(upload.ID)
		if err != nil {
			continue // Skip this upload if we can't get channels
		}

		for _, channel := range uploadChannels {
			var currentBuild *models.Build
			if channel.CurrentBuildID != nil {
				// Get the current build from the channel
				currentBuild, err = h.db.GetBuildByID(*channel.CurrentBuildID)
				if err != nil {
					continue // Skip this channel if we can't get the build
				}
			}

			channelData := map[string]interface{}{
				"name": channel.Name,
				"upload": map[string]interface{}{
					"id": upload.ID,
				},
			}

			if currentBuild != nil {
				buildData := map[string]interface{}{
					"id":    currentBuild.ID,
					"state": currentBuild.State,
				}

				if currentBuild.UserVersion != "" {
					buildData["user_version"] = currentBuild.UserVersion
				}

				if currentBuild.ParentBuildID != nil {
					buildData["parent_build_id"] = *currentBuild.ParentBuildID
				}

				channelData["head"] = buildData
			}

			channels[channel.Name] = channelData
		}
	}

	response := map[string]interface{}{
		"channels": channels,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /wharf/channels/{channel} - Get channel information
func (h *WharfHandlers) GetChannel(w http.ResponseWriter, r *http.Request) {
	channelName := mux.Vars(r)["channel"]
	target := r.URL.Query().Get("target")

	if target == "" {
		http.Error(w, `{"errors":["missing build target (need game_id or target)"]}`, http.StatusBadRequest)
		return
	}

	// Parse target format: "username/gamename"
	parts := strings.Split(target, "/")
	if len(parts) != 2 {
		http.Error(w, `{"errors":["invalid target format, expected username/gamename"]}`, http.StatusBadRequest)
		return
	}

	username := parts[0]
	gamename := parts[1]

	// Get user from context (set by auth middleware)
	user, ok := auth.GetUser(r.Context())
	if !ok || user == nil {
		http.Error(w, `{"errors":["user not found in context"]}`, http.StatusInternalServerError)
		return
	}

	// Validate namespace access
	err := h.validateNamespaceAccess(user, username)
	if err != nil {
		fmt.Printf("Namespace access denied: %v\n", err)
		http.Error(w, `{"errors":["access denied"]}`, http.StatusForbidden)
		return
	}

	// Find the target user (for namespace access)
	var targetUserID int64
	if user.Username == username {
		// User accessing their own namespace
		targetUserID = user.ID
	} else {
		// Admin user accessing another user's namespace - look up the target user
		targetUser, err := h.db.GetUserByUsername(username)
		if err != nil {
			http.Error(w, `{"errors":["target user not found"]}`, http.StatusNotFound)
			return
		}
		targetUserID = targetUser.ID
	}

	// Find the game owned by the target user
	game, err := h.db.GetGameByUserAndTitle(targetUserID, gamename)
	if err != nil {
		http.Error(w, `{"errors":["game not found"]}`, http.StatusNotFound)
		return
	}

	// Get all uploads for this game
	uploads, err := h.db.GetUploadsByGameID(game.ID)
	if err != nil {
		http.Error(w, `{"errors":["failed to get uploads"]}`, http.StatusInternalServerError)
		return
	}

	// Find the channel across all uploads
	var foundChannel *models.Channel
	var foundUpload *models.Upload

	for _, upload := range uploads {
		channels, err := h.db.GetChannelsByUploadID(upload.ID)
		if err != nil {
			continue
		}

		for _, channel := range channels {
			if channel.Name == channelName {
				foundChannel = channel
				foundUpload = upload
				break
			}
		}

		if foundChannel != nil {
			break
		}
	}

	if foundChannel == nil {
		http.Error(w, `{"errors":["channel not found"]}`, http.StatusNotFound)
		return
	}

	channelData := map[string]interface{}{
		"name": foundChannel.Name,
		"upload": map[string]interface{}{
			"id": foundUpload.ID,
		},
	}

	// Get the current build if it exists
	if foundChannel.CurrentBuildID != nil {
		currentBuild, err := h.db.GetBuildByID(*foundChannel.CurrentBuildID)
		if err == nil {
			buildData := map[string]interface{}{
				"id":    currentBuild.ID,
				"state": currentBuild.State,
			}

			if currentBuild.UserVersion != "" {
				buildData["user_version"] = currentBuild.UserVersion
			}

			if currentBuild.ParentBuildID != nil {
				buildData["parent_build_id"] = *currentBuild.ParentBuildID
			}

			channelData["head"] = buildData
		}
	}

	response := map[string]interface{}{
		"channel": channelData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /wharf/builds - Create a new build
func (h *WharfHandlers) CreateBuild(w http.ResponseWriter, r *http.Request) {
	user := auth.MustGetUser(r.Context())

	// Debug: Read and log the raw request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("Error reading request body: %v\n", err)
		http.Error(w, `{"errors":["could not read request body"]}`, http.StatusBadRequest)
		return
	}
	fmt.Printf("CreateBuild request body: %s\n", string(body))
	fmt.Printf("Content-Type: %s\n", r.Header.Get("Content-Type"))

	// Parse request body - try JSON first, then form data
	var req struct {
		Target      string `json:"target"`
		Channel     string `json:"channel"`
		UserVersion string `json:"user_version"`
	}

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		// Parse as JSON
		if err = json.Unmarshal(body, &req); err != nil {
			fmt.Printf("JSON parsing error: %v\n", err)
			http.Error(w, fmt.Sprintf(`{"errors":["invalid request body: %s"]}`, err.Error()), http.StatusBadRequest)
			return
		}
	} else {
		// Try parsing as form data by recreating the body
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		if err = r.ParseForm(); err != nil {
			fmt.Printf("Form parsing error: %v\n", err)
			http.Error(w, fmt.Sprintf(`{"errors":["invalid form data: %s"]}`, err.Error()), http.StatusBadRequest)
			return
		}
		req.Target = r.Form.Get("target")
		req.Channel = r.Form.Get("channel")
		req.UserVersion = r.Form.Get("user_version")
	}

	fmt.Printf("Parsed request: target=%s, channel=%s, user_version=%s\n", req.Target, req.Channel, req.UserVersion)

	if req.Target == "" {
		http.Error(w, `{"errors":["missing target"]}`, http.StatusBadRequest)
		return
	}

	// Parse target to find game/upload
	parts := strings.Split(req.Target, "/")
	if len(parts) != 2 {
		http.Error(w, `{"errors":["invalid target format"]}`, http.StatusBadRequest)
		return
	}

	username, gameName := parts[0], parts[1]

	// Validate namespace access
	err = h.validateNamespaceAccess(user, username)
	if err != nil {
		fmt.Printf("Namespace access denied: %v\n", err)
		http.Error(w, `{"errors":["access denied"]}`, http.StatusForbidden)
		return
	}

	// For this simple implementation, we'll create a game and upload if they don't exist
	// In practice, you'd want better lookup logic

	// Find the namespace owner (the user who owns this namespace)
	namespaceOwner, err := h.db.GetUserByUsername(username)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"errors":["namespace owner not found: %s"]}`, username), http.StatusNotFound)
		return
	}

	// Create or find game
	fmt.Printf("Looking for existing game: namespace_owner_id=%d, title='%s'\n", namespaceOwner.ID, gameName)

	// First try to find existing game owned by the namespace owner
	var games []*models.Game
	games, err = h.db.GetGamesByUserID(namespaceOwner.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"errors":["%s"]}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Find game by title
	var game *models.Game
	for _, g := range games {
		if g.Title == gameName {
			game = g
			fmt.Printf("Found existing game: ID=%d, Title='%s'\n", game.ID, game.Title)
			break
		}
	}

	if game == nil {
		// Create new game owned by the namespace owner
		game = &models.Game{
			UserID:         namespaceOwner.ID,
			Title:          gameName,
			Type:           "default",
			Classification: "game",
		}

		err = h.db.CreateGame(game)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"errors":["%s"]}`, err.Error()), http.StatusInternalServerError)
			return
		}
		fmt.Printf("Created new game: ID=%d, Title='%s', Owner='%s'\n", game.ID, game.Title, namespaceOwner.Username)
	}

	// Create or find upload - look for existing upload that matches the channel
	var uploads []*models.Upload
	uploads, err = h.db.GetUploadsByGameID(game.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"errors":["%s"]}`, err.Error()), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Found %d existing uploads for game %d\n", len(uploads), game.ID)

	var upload *models.Upload

	// Try to find existing upload that has a channel matching our request
	for _, existingUpload := range uploads {
		fmt.Printf("Checking upload %d for channel %s\n", existingUpload.ID, req.Channel)
		// Check if this upload has a channel with the requested name
		_, channelErr := h.db.GetChannelByName(req.Channel, existingUpload.ID)
		if channelErr == nil {
			// Found an upload that already has this channel
			upload = existingUpload
			fmt.Printf("Found existing upload %d with channel %s\n", upload.ID, req.Channel)
			break
		} else {
			fmt.Printf("Upload %d does not have channel %s: %v\n", existingUpload.ID, req.Channel, channelErr)
		}
	}

	if upload == nil {
		// No existing upload with this channel, create a new one
		upload = &models.Upload{
			GameID:      game.ID,
			Filename:    fmt.Sprintf("%s.zip", gameName),
			DisplayName: gameName,
			Storage:     "hosted",
			Type:        "default",
			Platforms:   `["windows","linux","osx"]`,
		}

		err = h.db.CreateUpload(upload)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"errors":["%s"]}`, err.Error()), http.StatusInternalServerError)
			return
		}
		fmt.Printf("Created new upload %d for channel %s\n", upload.ID, req.Channel)
	}

	// Find parent build (current build for this channel)
	var parentBuildID *int64
	var existingChannel *models.Channel

	// First check if channel already exists to get current build
	fmt.Printf("Looking for existing channel: name='%s', upload_id=%d\n", req.Channel, upload.ID)
	existingChannel, err = h.db.GetChannelByName(req.Channel, upload.ID)
	if err != nil {
		fmt.Printf("Channel lookup error: %v\n", err)
		fmt.Printf("No existing channel found, this is the first build\n")
	} else {
		fmt.Printf("Found existing channel: ID=%d, CurrentBuildID=%v\n", existingChannel.ID, existingChannel.CurrentBuildID)
		if existingChannel.CurrentBuildID != nil {
			// Channel exists and has a current build - use it as parent
			parentBuildID = existingChannel.CurrentBuildID
			fmt.Printf("Using existing build ID %d as parent\n", *parentBuildID)
		} else {
			fmt.Printf("Existing channel has no current build\n")
		}
	}

	// Create new build
	build := &models.Build{
		UploadID:      upload.ID,
		UserVersion:   req.UserVersion,
		ParentBuildID: parentBuildID,
		State:         "started",
	}

	fmt.Printf("Creating build: UploadID=%d, ParentBuildID=%v, UserVersion='%s'\n",
		build.UploadID, build.ParentBuildID, build.UserVersion)

	err = h.db.CreateBuild(build)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"errors":["%s"]}`, err.Error()), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Created build with ID: %d\n", build.ID)

	// Create or update channel to point to new build
	if existingChannel != nil {
		// Channel exists, update it to point to new build
		existingChannel.CurrentBuildID = &build.ID
		err = h.db.UpdateChannel(existingChannel)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"errors":["%s"]}`, err.Error()), http.StatusInternalServerError)
			return
		}
		fmt.Printf("Updated existing channel to point to build %d\n", build.ID)
	} else {
		// Channel doesn't exist, create it
		channel := &models.Channel{
			Name:           req.Channel,
			UploadID:       upload.ID,
			CurrentBuildID: &build.ID,
		}
		err = h.db.CreateChannel(channel)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"errors":["%s"]}`, err.Error()), http.StatusInternalServerError)
			return
		}
		fmt.Printf("Created new channel pointing to build %d\n", build.ID)
	}

	buildResponse := map[string]interface{}{
		"id":          build.ID,
		"uploadId":    build.UploadID,
		"userVersion": build.UserVersion,
		"state":       build.State,
	}

	if build.ParentBuildID != nil {
		buildResponse["parentBuild"] = map[string]interface{}{
			"id": *build.ParentBuildID,
		}
		fmt.Printf("Build response includes parentBuild.id: %d\n", *build.ParentBuildID)
	} else {
		fmt.Printf("Build response has no parentBuild (first build)\n")
	}

	response := map[string]interface{}{
		"build": buildResponse,
	}

	// Debug: Log the complete response JSON that will be sent to butler
	responseJSON, _ := json.MarshalIndent(response, "", "  ")
	fmt.Printf("CreateBuild response JSON being sent to butler:\n%s\n", string(responseJSON))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /wharf/builds/{id}/files - List files for a build
func (h *WharfHandlers) GetBuildFiles(w http.ResponseWriter, r *http.Request) {
	buildIDStr := mux.Vars(r)["id"]
	fmt.Printf("GetBuildFiles request for build: %s\n", buildIDStr)

	buildID, err := strconv.ParseInt(buildIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"errors":["invalid build id"]}`, http.StatusBadRequest)
		return
	}

	// Check if build exists
	_, err = h.db.GetBuildByID(buildID)
	if err != nil {
		http.Error(w, `{"errors":["build not found"]}`, http.StatusNotFound)
		return
	}

	var buildFiles []*models.BuildFile
	buildFiles, err = h.db.GetBuildFilesByBuildID(buildID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"errors":["%s"]}`, err.Error()), http.StatusInternalServerError)
		return
	}

	var filesResponse []map[string]interface{}
	for _, file := range buildFiles {
		fileResponse := map[string]interface{}{
			"id":      file.ID,
			"type":    file.Type,
			"subType": file.SubType,
			"size":    file.Size,
			"state":   file.State,
		}
		filesResponse = append(filesResponse, fileResponse)
		fmt.Printf("Returning build file: id=%d, type=%s, subType=%s, size=%d, state=%s\n",
			file.ID, file.Type, file.SubType, file.Size, file.State)
	}

	response := map[string]interface{}{
		"Files": filesResponse,
	}

	fmt.Printf("GetBuildFiles response for build %d: %d files\n", buildID, len(buildFiles))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /wharf/builds/{id}/files - Create a build file
func (h *WharfHandlers) CreateBuildFile(w http.ResponseWriter, r *http.Request) {
	buildIDStr := mux.Vars(r)["id"]
	buildID, err := strconv.ParseInt(buildIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"errors":["invalid build id"]}`, http.StatusBadRequest)
		return
	}

	// Debug: Read and log the raw request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("Error reading request body: %v\n", err)
		http.Error(w, `{"errors":["could not read request body"]}`, http.StatusBadRequest)
		return
	}
	fmt.Printf("CreateBuildFile request body: %s\n", string(body))
	fmt.Printf("CreateBuildFile Content-Type: %s\n", r.Header.Get("Content-Type"))

	// Check if build exists
	_, err = h.db.GetBuildByID(buildID)
	if err != nil {
		http.Error(w, `{"errors":["build not found"]}`, http.StatusNotFound)
		return
	}

	// Parse request body - handle form data like the build creation
	var req struct {
		Type       string `json:"type"`
		SubType    string `json:"sub_type"`
		UploadType string `json:"upload_type"`
	}

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		// Parse as JSON
		if err = json.Unmarshal(body, &req); err != nil {
			fmt.Printf("JSON parsing error: %v\n", err)
			http.Error(w, fmt.Sprintf(`{"errors":["invalid request body: %s"]}`, err.Error()), http.StatusBadRequest)
			return
		}
	} else {
		// Try parsing as form data
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		if err = r.ParseForm(); err != nil {
			fmt.Printf("Form parsing error: %v\n", err)
			http.Error(w, fmt.Sprintf(`{"errors":["invalid form data: %s"]}`, err.Error()), http.StatusBadRequest)
			return
		}
		req.Type = r.Form.Get("type")
		req.SubType = r.Form.Get("sub_type")
		req.UploadType = r.Form.Get("upload_type")
	}

	fmt.Printf("CreateBuildFile parsed: type=%s, sub_type=%s, upload_type=%s\n", req.Type, req.SubType, req.UploadType)

	if req.Type == "" {
		http.Error(w, `{"errors":["missing type"]}`, http.StatusBadRequest)
		return
	}

	if req.SubType == "" {
		req.SubType = "default"
	}

	// Generate unique file ID for storage
	fileID := uuid.New().String()

	// Create storage path in MinIO
	storagePath := fmt.Sprintf("builds/%d/%s_%s_%s", buildID, req.Type, req.SubType, fileID)

	// Generate presigned upload URL for MinIO (expires in 1 hour)
	uploadURL, err := h.GetPresignedUploadURL(storagePath, time.Hour)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"errors":["failed to generate upload URL: %s"]}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Create build file
	buildFile := &models.BuildFile{
		BuildID:     buildID,
		Type:        req.Type,
		SubType:     req.SubType,
		State:       "uploading",
		StoragePath: storagePath,
		UploadURL:   uploadURL,
	}

	err = h.db.CreateBuildFile(buildFile)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"errors":["%s"]}`, err.Error()), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"file": map[string]interface{}{
			"id":         buildFile.ID,
			"type":       buildFile.Type,
			"sub_type":   buildFile.SubType,
			"state":      buildFile.State,
			"upload_url": buildFile.UploadURL,
			"upload_headers": map[string]interface{}{
				"Content-Type": "application/octet-stream",
			},
		},
	}

	fmt.Printf("CreateBuildFile response: %+v\n", response)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /wharf/builds/{buildId}/files/{fileId} - Finalize build file upload
func (h *WharfHandlers) FinalizeBuildFile(w http.ResponseWriter, r *http.Request) {
	buildIDStr := mux.Vars(r)["buildId"]
	fileIDStr := mux.Vars(r)["fileId"]

	buildID, err := strconv.ParseInt(buildIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"errors":["invalid build id"]}`, http.StatusBadRequest)
		return
	}

	fileID, err := strconv.ParseInt(fileIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"errors":["invalid file id"]}`, http.StatusBadRequest)
		return
	}

	// Debug: Read and log the raw request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("Error reading finalize request body: %v\n", err)
		http.Error(w, `{"errors":["could not read request body"]}`, http.StatusBadRequest)
		return
	}
	fmt.Printf("FinalizeBuildFile request body: %s\n", string(body))
	fmt.Printf("FinalizeBuildFile Content-Type: %s\n", r.Header.Get("Content-Type"))

	// Parse request body - handle form data like other endpoints
	var req struct {
		Size int64 `json:"size"`
	}

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		// Parse as JSON
		if err = json.Unmarshal(body, &req); err != nil {
			fmt.Printf("JSON parsing error: %v\n", err)
			http.Error(w, fmt.Sprintf(`{"errors":["invalid request body: %s"]}`, err.Error()), http.StatusBadRequest)
			return
		}
	} else {
		// Try parsing as form data
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		if err = r.ParseForm(); err != nil {
			fmt.Printf("Form parsing error: %v\n", err)
			http.Error(w, fmt.Sprintf(`{"errors":["invalid form data: %s"]}`, err.Error()), http.StatusBadRequest)
			return
		}
		sizeStr := r.Form.Get("size")
		if sizeStr != "" {
			req.Size, err = strconv.ParseInt(sizeStr, 10, 64)
			if err != nil {
				fmt.Printf("Size parsing error: %v\n", err)
				http.Error(w, fmt.Sprintf(`{"errors":["invalid size: %s"]}`, err.Error()), http.StatusBadRequest)
				return
			}
		}
	}

	fmt.Printf("FinalizeBuildFile parsed: size=%d\n", req.Size)

	// Get build file
	var buildFile *models.BuildFile
	buildFile, err = h.db.GetBuildFileByID(fileID)
	if err != nil {
		http.Error(w, `{"errors":["build file not found"]}`, http.StatusNotFound)
		return
	}

	if buildFile.BuildID != buildID {
		http.Error(w, `{"errors":["build file does not belong to build"]}`, http.StatusBadRequest)
		return
	}

	// Verify that the file was actually uploaded to MinIO
	if !h.FileExists(buildFile.StoragePath) {
		http.Error(w, `{"errors":["file not found in storage - upload may have failed"]}`, http.StatusBadRequest)
		return
	}

	// Get actual file size from MinIO to verify
	actualSize, err := h.GetFileSize(buildFile.StoragePath)
	if err != nil {
		http.Error(w, `{"errors":["could not verify file size in storage"]}`, http.StatusInternalServerError)
		return
	}

	// Update build file with actual size from storage and mark as uploaded
	buildFile.Size = actualSize
	buildFile.State = "uploaded"

	err = h.db.UpdateBuildFile(buildFile)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"errors":["%s"]}`, err.Error()), http.StatusInternalServerError)
		return
	}

	fmt.Printf("File upload verified: %s (size: %d bytes)\n", buildFile.StoragePath, actualSize)

	// Check if all files for this build are now uploaded and update build state
	err = h.checkAndUpdateBuildState(buildID)
	if err != nil {
		fmt.Printf("Warning: Failed to update build state: %v\n", err)
		// Don't fail the request, just log the warning
	}

	response := map[string]interface{}{
		"file": map[string]interface{}{
			"id":    buildFile.ID,
			"size":  buildFile.Size,
			"state": buildFile.State,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// checkAndUpdateBuildState checks if all files for a build are uploaded and updates the build state accordingly
func (h *WharfHandlers) checkAndUpdateBuildState(buildID int64) error {
	// Get the current build
	build, err := h.db.GetBuildByID(buildID)
	if err != nil {
		return fmt.Errorf("failed to get build: %w", err)
	}

	// Only process builds that are in "started" state
	if build.State != "started" {
		return nil
	}

	// Get all files for this build
	buildFiles, err := h.db.GetBuildFilesByBuildID(buildID)
	if err != nil {
		return fmt.Errorf("failed to get build files: %w", err)
	}

	// Check if we have any files and if all are uploaded
	if len(buildFiles) == 0 {
		// No files yet, keep in "started" state
		return nil
	}

	allUploaded := true
	for _, file := range buildFiles {
		if file.State != "uploaded" {
			allUploaded = false
			break
		}
	}

	if allUploaded {
		// All files are uploaded, transition to "processing" then immediately to "completed"
		fmt.Printf("All files uploaded for build %d, transitioning to processing\n", buildID)

		build.State = "processing"
		err = h.db.UpdateBuild(build)
		if err != nil {
			return fmt.Errorf("failed to update build state to processing: %w", err)
		}

		// Generate archive file for fetch operations
		err = h.generateArchiveFile(build)
		if err != nil {
			fmt.Printf("Warning: Failed to generate archive file for build %d: %v\n", buildID, err)
			// Don't fail the build, just log the warning
		}

		build.State = "completed"
		err = h.db.UpdateBuild(build)
		if err != nil {
			return fmt.Errorf("failed to update build state to completed: %w", err)
		}

		fmt.Printf("Build %d state updated to: %s\n", buildID, build.State)
	}

	return nil
}

// generateArchiveFile creates a ZIP archive containing the full game content for fetch operations
func (h *WharfHandlers) generateArchiveFile(build *models.Build) error {
	fmt.Printf("Generating archive file for build %d\n", build.ID)

	// Generate archive from all build files in MinIO
	// This creates a ZIP archive containing all files from this build

	archivePath, archiveSize, err := h.createArchiveFromBuildFiles(build.ID)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}

	// Create a build file entry for the archive
	archiveFile := &models.BuildFile{
		BuildID:     build.ID,
		Type:        "archive",
		SubType:     "default",
		State:       "uploaded",
		Size:        archiveSize,
		StoragePath: fmt.Sprintf("builds/%d/files", build.ID), // Will be updated after we get the file ID
	}

	err = h.db.CreateBuildFile(archiveFile)
	if err != nil {
		return fmt.Errorf("failed to create archive build file: %w", err)
	}

	// Update the StoragePath with the correct file ID
	archiveFile.StoragePath = fmt.Sprintf("builds/%d/files/%d", build.ID, archiveFile.ID)
	err = h.db.UpdateBuildFile(archiveFile)
	if err != nil {
		return fmt.Errorf("failed to update archive build file storage path: %w", err)
	}

	// Move the archive to the proper storage location
	finalPath := fmt.Sprintf("storage/builds/%d/files/%d", build.ID, archiveFile.ID)
	err = os.MkdirAll(filepath.Dir(finalPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	err = os.Rename(archivePath, finalPath)
	if err != nil {
		return fmt.Errorf("failed to move archive to final location: %w", err)
	}

	fmt.Printf("Generated archive file %d for build %d (size: %d bytes)\n", archiveFile.ID, build.ID, archiveSize)
	return nil
}

// createArchiveFromBuildFiles creates a ZIP archive from all build files in MinIO
func (h *WharfHandlers) createArchiveFromBuildFiles(buildID int64) (string, int64, error) {
	// Get all build files for this build
	buildFiles, err := h.db.GetBuildFilesByBuildID(buildID)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get build files: %w", err)
	}
	// Create a temporary file for the archive
	tempFile, err := os.CreateTemp("", "archive-*.zip")
	if err != nil {
		return "", 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Create ZIP writer
	zipWriter := zip.NewWriter(tempFile)
	defer zipWriter.Close()

	ctx := context.Background()

	// Add each build file to the archive
	for _, buildFile := range buildFiles {
		if buildFile.State != "uploaded" {
			continue // Skip files that aren't fully uploaded
		}

		// Get the file from MinIO
		object, err := h.minioClient.GetObject(ctx, h.bucketName, buildFile.StoragePath, minio.GetObjectOptions{})
		if err != nil {
			fmt.Printf("Warning: failed to get file %s from MinIO: %v\n", buildFile.StoragePath, err)
			continue
		}

		// Create entry in ZIP
		filename := fmt.Sprintf("%s_%s", buildFile.Type, buildFile.SubType)
		if buildFile.Type == "archive" {
			filename += ".zip"
		}

		writer, err := zipWriter.Create(filename)
		if err != nil {
			object.Close()
			return "", 0, fmt.Errorf("failed to create zip entry: %w", err)
		}

		// Copy file content to ZIP
		_, err = io.Copy(writer, object)
		object.Close()
		if err != nil {
			return "", 0, fmt.Errorf("failed to copy file to archive: %w", err)
		}

		fmt.Printf("Added file %s to archive\n", filename)
	}

	// If no files were added, create a placeholder
	if len(buildFiles) == 0 {
		writer, err := zipWriter.Create("README.txt")
		if err != nil {
			return "", 0, fmt.Errorf("failed to create placeholder: %w", err)
		}
		content := fmt.Sprintf("Build %d\nGenerated at: %s\nNo files uploaded yet.\n", buildID, time.Now().Format(time.RFC3339))
		_, err = writer.Write([]byte(content))
		if err != nil {
			return "", 0, fmt.Errorf("failed to write placeholder: %w", err)
		}
	}

	// Close ZIP writer to finalize
	err = zipWriter.Close()
	if err != nil {
		return "", 0, fmt.Errorf("failed to close zip writer: %w", err)
	}

	// Get file size
	stat, err := os.Stat(tempFile.Name())
	if err != nil {
		return "", 0, fmt.Errorf("failed to get file size: %w", err)
	}

	return tempFile.Name(), stat.Size(), nil
}

// GET /wharf/builds/{buildId}/files/{fileId}/download - Download build file
func (h *WharfHandlers) GetBuildFileDownload(w http.ResponseWriter, r *http.Request) {
	buildIDStr := mux.Vars(r)["buildId"]
	fileIDStr := mux.Vars(r)["fileId"]

	fmt.Printf("GetBuildFileDownload request: buildId=%s, fileId=%s\n", buildIDStr, fileIDStr)
	fmt.Printf("GetBuildFileDownload URL: %s\n", r.URL.String())

	buildID, err := strconv.ParseInt(buildIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"errors":["invalid build id"]}`, http.StatusBadRequest)
		return
	}

	fileID, err := strconv.ParseInt(fileIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"errors":["invalid file id"]}`, http.StatusBadRequest)
		return
	}

	// Get build file
	var buildFile *models.BuildFile
	buildFile, err = h.db.GetBuildFileByID(fileID)
	if err != nil {
		http.Error(w, `{"errors":["build file not found"]}`, http.StatusNotFound)
		return
	}

	if buildFile.BuildID != buildID {
		http.Error(w, `{"errors":["build file does not belong to build"]}`, http.StatusBadRequest)
		return
	}

	// Check if file exists in storage
	if !h.FileExists(buildFile.StoragePath) {
		http.Error(w, `{"errors":["file not found in storage"]}`, http.StatusNotFound)
		return
	}

	// Generate signed URL for secure download (expires in 1 hour)
	signedURL, err := h.GetSignedURL(buildFile.StoragePath, time.Hour)
	if err != nil {
		http.Error(w, `{"errors":["could not generate download URL"]}`, http.StatusInternalServerError)
		return
	}

	// Redirect to signed URL for direct download from MinIO
	http.Redirect(w, r, signedURL, http.StatusTemporaryRedirect)
}
