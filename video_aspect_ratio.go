package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	//run cli command
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	//set bytes.buffer
	var stdoutBuffer bytes.Buffer

	cmd.Stdout = &stdoutBuffer
	//run command
	err := cmd.Run()
	if err != nil {
		fmt.Printf("cmd.Run() failed with %s\n", err)
		return "", err
	}
	//set json struct
	type videoDimensionsStruct struct {
		Streams []struct {
			Width  float64 `json:"width,omitempty"`
			Height float64 `json:"height,omitempty"`
		} `json:"streams"`
	}
	//unmarshal stdout into json struct
	var videoDimensions videoDimensionsStruct
	if err := json.Unmarshal(stdoutBuffer.Bytes(), &videoDimensions); err != nil {
		fmt.Printf("error unmarshalling: %v", err)
		return "", err
	}
	//determine aspect ratio
	aspectRatio := videoDimensions.Streams[0].Width / videoDimensions.Streams[0].Height

	if aspectRatio >= 0.5 && aspectRatio <= 0.6 {
		return "9:16", nil
	} else if aspectRatio >= 1.7 && aspectRatio <= 1.8 {
		return "16:9", nil
	} else {
		return "other", nil
	}
}
