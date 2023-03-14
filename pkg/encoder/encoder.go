package encoder

import (
	"bytes"
	"os/exec"
)

// Encode to mp3 128kbps
func EncodeToMP3(input []byte) ([]byte, error) {
	out := &bytes.Buffer{}

	cmd := exec.Command("ffmpeg",
		"-i", "pipe:0",
		"-map_metadata", "-1",
		"-preset", "veryfast",
		"-b:a", "128k",
		"-f", "mp3",
		"-vn",
		"-id3v2_version", "0",
		"-write_xing", "0",
		"-c:a", "libmp3lame",
		"-vsync", "2",
		"pipe:1",
	)
	cmd.Stdout = out

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	_, err = stdin.Write(input)
	if err != nil {
		return nil, err
	}

	err = stdin.Close()
	if err != nil {
		return nil, err
	}

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}
