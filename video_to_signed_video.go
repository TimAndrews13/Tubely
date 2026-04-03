package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}
	parts := strings.Split(*video.VideoURL, ",")
	if len(parts) < 2 {
		return video, nil
	}
	bucket := parts[0]
	key := parts[1]

	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Hour)
	if err != nil {
		fmt.Printf("error generating presigned url: %v\n", err)
		return database.Video{}, err
	}

	video.VideoURL = &presignedURL
	return video, nil
}
