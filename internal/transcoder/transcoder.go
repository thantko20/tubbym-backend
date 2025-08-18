package transcoder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Variant struct {
	Name    string
	Width   int
	Height  int
	Bitrate string
}

var variants = []Variant{
	// {Name: "1080p", Width: 1920, Height: 1080, Bitrate: "5000k"},
	{Name: "720p", Width: 1280, Height: 720, Bitrate: "2800k"},
	{Name: "480p", Width: 854, Height: 480, Bitrate: "1400k"},
}

type Transcoder struct {
	ffmpegPath string
	tempDir    string
}

func New(ffmpegPath, tempDir string) *Transcoder {
	return &Transcoder{
		ffmpegPath: ffmpegPath,
		tempDir:    tempDir,
	}
}

func (t *Transcoder) TranscodeToHLS(inputPath string) (string, error) {
	outputDir := filepath.Join(t.tempDir, filepath.Base(inputPath)+"-hls")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	variantPlaylists := []string{}
	for _, v := range variants {
		playlist := fmt.Sprintf("%s.m3u8", v.Name)
		playlistPath := filepath.Join(outputDir, playlist)
		args := []string{
			"-i", inputPath,
			"-threads", "1",
			"-vf", fmt.Sprintf("scale=w=%d:h=%d:force_original_aspect_ratio=decrease:force_divisible_by=2", v.Width, v.Height),
			"-preset", "veryfast",
			"-c:a", "aac", "-ar", "48000", "-c:v", "h264", "-profile:v", "main",
			"-crf", "20", "-sc_threshold", "0",
			"-g", "48", "-keyint_min", "48",
			"-b:v", v.Bitrate,
			"-maxrate", v.Bitrate,
			"-bufsize", "1200k",
			"-hls_time", "6",
			"-hls_playlist_type", "vod",
			"-f", "hls",
			"-hls_segment_filename", filepath.Join(outputDir, fmt.Sprintf("%s_%%03d.ts", v.Name)),
			playlistPath,
		}
		// cmd := exec.Command(t.ffmpegPath, args...)
		cmd := exec.Command("nice", append([]string{"-n", "10", "--", t.ffmpegPath}, args...)...)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("transcoding %s failed: %w", v.Name, err)
		}
		variantPlaylists = append(variantPlaylists, playlist)
	}

	// Create master playlist
	masterPath := filepath.Join(outputDir, "playlist.m3u8")
	master, err := os.Create(masterPath)
	if err != nil {
		return "", err
	}
	defer master.Close()

	for i, v := range variants {
		fmt.Fprintf(master, "#EXT-X-STREAM-INF:BANDWIDTH=%s,RESOLUTION=%dx%d\n%s\n",
			strings.TrimSuffix(v.Bitrate, "k")+"000", v.Width, v.Height, variantPlaylists[i])
	}

	return outputDir, nil
}

func (t *Transcoder) Cleanup(outputDir string) error {
	return os.RemoveAll(outputDir)
}
