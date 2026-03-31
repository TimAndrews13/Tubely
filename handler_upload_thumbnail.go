package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	// "thumbnail" should match the HTML form input name
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	//Get Media Type from Content-Type Header
	mediaType := header.Header.Get("Content-Type")

	//Read image data into a byte slice
	data, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to read from file", err)
		return
	}

	//Pull Video Metadata from Database
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Video metadata does not exit", err)
		return
	}

	//Check Authenitcated User against Video Ownere
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "current user not owner of video", err)
		return
	}

	//Create new thumbnail struct with image data and media type
	thumbnail := thumbnail{
		data:      data,
		mediaType: mediaType,
	}

	//Add Thumbnail to the global map with VideoID as Key
	videoThumbnails[video.ID] = thumbnail

	//Update video metadata to the new thumbnail url
	thumbnailURL := "http://localhost:" + cfg.port + "/api/thumbnails/" + video.ID.String()

	video.ThumbnailURL = &thumbnailURL

	//Update Database record for Video by updating the Video
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error updating updated video record", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
