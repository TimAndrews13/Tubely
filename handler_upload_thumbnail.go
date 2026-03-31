package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	mediaInfo := header.Header.Get("Content-Type")
	if mediaInfo == "" {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type Header", nil)
		return
	}
	//Parse Media Type from header response
	mediaType, _, _ := mime.ParseMediaType(mediaInfo)
	if mediaType != "image/png" && mediaType != "image/jpeg" {
		respondWithError(w, http.StatusBadRequest, "file type must be an image for thumbnail", nil)
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

	//Create Random 32 byte slice and convert to random base64 string
	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating random file sting", err)
		return
	}
	encodedString := base64.RawURLEncoding.EncodeToString(b)

	//Create Unique File Path for File Image to be written to /assets folder
	fileExtension := strings.Split(mediaInfo, "/")[1]

	discPath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s.%s", encodedString, fileExtension))

	//Get URL and Store in the Video at ThumbnailURL
	fileURL := fmt.Sprintf("http://localhost:%s/assets/%s.%s", os.Getenv("PORT"), encodedString, fileExtension)

	video.ThumbnailURL = &fileURL

	//Create file at Filepath
	dst, err := os.Create(discPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating file", err)
		return
	}
	defer dst.Close()

	//Write to filepath now
	if _, err := io.Copy(dst, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "error writing to new file", err)
		return
	}

	//Update Database record for Video by updating the Video
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error updating updated video record", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
