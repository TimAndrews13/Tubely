package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func processVideoForFastStart(filePath string) (string, error) {
	//set new path
	newFilePath := filePath + ".processing"
	//set cli command
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", newFilePath)
	//set bytes.Buffer
	var stdoutBuffer bytes.Buffer
	cmd.Stderr = &stdoutBuffer
	//run command
	err := cmd.Run()
	if err != nil {
		fmt.Printf("cmd.Run() failed with %s\n", err)
		return "", err
	}
	//validate newFilePath
	info, err := os.Stat(newFilePath)
	if err != nil {
		fmt.Printf("could not stat file: %v\n", err)
		return "", err
	}
	if info.Size() == 0 {
		return "", fmt.Errorf("fiel is empty")
	}
	//return newFilePath
	return newFilePath, nil
}
