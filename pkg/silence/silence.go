// Package to generate silence bytes
package silence

import (
	"bytes"
	"os/exec"
	"strconv"
)

// Generate silence
func Generate(n int) ([]byte, error) {

	// ffmpeg command to generate silence
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi",
		"-i", "anullsrc",
		"-t", strconv.Itoa(n),
		"-map_metadata", "-1",
		"-c:a", "libmp3lame",
		"-id3v2_version", "0",
		"-b:a", "128k",
		"-f", "mp3",
		"-",
	)
	out := &bytes.Buffer{}
	cmd.Stdout = out

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}
