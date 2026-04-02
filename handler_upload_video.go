package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	//Set upload limit
	const maxUploadSize = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	//extract videoID from URL
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
	}

	//Authenticate user
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Coudln't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
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

	//Parse the uploaded video file form the form data
	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	//Get Media Type from Content-Type Header
	media := header.Header.Get("Content-Type")
	if media == "" {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type header", nil)
		return
	}
	//Validate Media Type is a Video
	mediaType, _, _ := mime.ParseMediaType(media)
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "file tybe must be a video/mp4 for video upload", err)
		return
	}

	//Save uploaded file to temporary file on disk
	f, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error saving video to temp file", err)
		return
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err := io.Copy(f, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "error writing to file", err)
		return
	}

	//get Aspect Ratio of the Video file from temp file
	aspectRatio, err := getVideoAspectRatio(f.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error getting aspect ratio", err)
		return
	}

	//create processed version of the video
	processedFilePath, err := processVideoForFastStart(f.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error processing video for fast start", err)
		return
	}

	//open processed file
	processedFile, err := os.Open(processedFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error reading processed file", err)
		return
	}
	defer os.Remove(processedFile.Name())
	defer processedFile.Close()

	//Create Random 32 byte slice and convert to hex encoded string
	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error Creating random file string", err)
		return
	}
	encodedString := hex.EncodeToString(b)
	//Get file Extentsio
	fileExtension := strings.Split(media, "/")[1]
	//Set File Key
	var fileKey string
	if aspectRatio == "16:9" {
		fileKey = fmt.Sprintf("landscape/%s.%s", encodedString, fileExtension)
	} else if aspectRatio == "9:16" {
		fileKey = fmt.Sprintf("portrait/%s.%s", encodedString, fileExtension)
	} else if aspectRatio == "other" {
		fileKey = fmt.Sprintf("other/%s.%s", encodedString, fileExtension)
	}

	//Put video onto s3
	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fileKey,
		Body:        processedFile,
		ContentType: &mediaType,
	})
	if err != nil {
		log.Printf("S3 upload error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "error uploading file to s3", err)
		return
	}

	//Update Database record for Video by updating Video URL
	videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, fileKey)

	video.VideoURL = &videoURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error updating video record url", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
