package media

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/anupam-chopra/prism/internal/model"
)

type renditionProfile struct {
	Name      string
	Width     int
	Height    int
	VideoRate string
	MaxRate   string
	BufSize   string
	Bandwidth int
}

var abrProfiles = []renditionProfile{
	{Name: "360p", Width: 640, Height: 360, VideoRate: "800k", MaxRate: "856k", BufSize: "1200k", Bandwidth: 928000},
	{Name: "540p", Width: 960, Height: 540, VideoRate: "1400k", MaxRate: "1498k", BufSize: "2100k", Bandwidth: 1528000},
	{Name: "720p", Width: 1280, Height: 720, VideoRate: "2800k", MaxRate: "2996k", BufSize: "4200k", Bandwidth: 2928000},
	{Name: "1080p", Width: 1920, Height: 1080, VideoRate: "5000k", MaxRate: "5350k", BufSize: "7500k", Bandwidth: 5128000},
}

type Processor struct {
	root          string
	ffmpegPath    string
	shakaPackager string
	logger        *slog.Logger
}

type ProcessOptions struct {
	DRM string
}

func NewProcessor(root, ffmpegPath, shakaPackager string, logger *slog.Logger) *Processor {
	if root == "" {
		root = "data/assets"
	}
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	if shakaPackager == "" {
		shakaPackager = "packager"
	}
	return &Processor{
		root:          root,
		ffmpegPath:    ffmpegPath,
		shakaPackager: shakaPackager,
		logger:        logger,
	}
}

func (p *Processor) Process(ctx context.Context, assetID string, src io.Reader) error {
	return p.ProcessWithOptions(ctx, assetID, src, ProcessOptions{})
}

func (p *Processor) ProcessWithOptions(ctx context.Context, assetID string, src io.Reader, opts ProcessOptions) error {
	if !model.IsValidAssetID(assetID) {
		return fmt.Errorf("invalid asset id %q", assetID)
	}
	if opts.DRM != "" && opts.DRM != DRMModeClearKey && opts.DRM != DRMModeWidevine && opts.DRM != DRMModeFairPlay {
		return fmt.Errorf("unsupported drm mode %q", opts.DRM)
	}

	if err := os.MkdirAll(p.root, 0o755); err != nil {
		return fmt.Errorf("create media root: %w", err)
	}

	tmpDir, err := os.MkdirTemp(p.root, ".processing-"+assetID+"-")
	if err != nil {
		return fmt.Errorf("create processing directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "source")
	inputFile, err := os.Create(inputPath)
	if err != nil {
		return fmt.Errorf("create upload file: %w", err)
	}
	if _, err := io.Copy(inputFile, src); err != nil {
		_ = inputFile.Close()
		return fmt.Errorf("save upload: %w", err)
	}
	if err := inputFile.Close(); err != nil {
		return fmt.Errorf("close upload file: %w", err)
	}

	start := time.Now()

	hlsDir := filepath.Join(tmpDir, "hls")
	if err := os.MkdirAll(hlsDir, 0o755); err != nil {
		return fmt.Errorf("create HLS directory: %w", err)
	}

	if err := p.processHLS(ctx, assetID, inputPath, hlsDir); err != nil {
		return err
	}

	dashDir := filepath.Join(tmpDir, "dash")
	if err := os.MkdirAll(dashDir, 0o755); err != nil {
		return fmt.Errorf("create DASH directory: %w", err)
	}

	if err := p.processDASH(ctx, assetID, inputPath, dashDir); err != nil {
		return err
	}

	var clearKeyMetadata *ClearKeyMetadata
	if opts.DRM == DRMModeClearKey {
		var err error
		clearKeyMetadata, err = NewClearKeyMetadata(assetID)
		if err != nil {
			return err
		}

		clearKeyDir := filepath.Join(tmpDir, "dash_clearkey")
		if err := os.MkdirAll(clearKeyDir, 0o755); err != nil {
			return fmt.Errorf("create ClearKey DASH directory: %w", err)
		}
		if err := p.processClearKeyDASH(ctx, assetID, inputPath, clearKeyDir, clearKeyMetadata); err != nil {
			return err
		}
	}

	var drmMetadata *DRMMetadata
	if opts.DRM == DRMModeWidevine {
		var err error
		drmMetadata, err = NewDRMMetadata(assetID)
		if err != nil {
			return err
		}

		drmDir := filepath.Join(tmpDir, "dash_drm")
		if err := os.MkdirAll(drmDir, 0o755); err != nil {
			return fmt.Errorf("create DRM DASH directory: %w", err)
		}
		if err := p.processWidevinePackagedDASH(ctx, assetID, inputPath, drmDir, drmMetadata); err != nil {
			return err
		}
	}

	var fairPlayMetadata *FairPlayMetadata
	if opts.DRM == DRMModeFairPlay {
		var err error
		fairPlayMetadata, err = NewFairPlayMetadata(assetID)
		if err != nil {
			return err
		}

		fpDir := filepath.Join(tmpDir, "hls_fairplay")
		if err := os.MkdirAll(fpDir, 0o755); err != nil {
			return fmt.Errorf("create FairPlay HLS directory: %w", err)
		}
		if err := p.processFairPlayHLS(ctx, assetID, inputPath, fpDir, fairPlayMetadata); err != nil {
			return err
		}
	}

	if p.logger != nil {
		p.logger.InfoContext(ctx, "video processed",
			"asset_id", assetID,
			"drm", opts.DRM,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}

	_ = os.Remove(inputPath)

	finalDir := filepath.Join(p.root, assetID)
	if err := os.RemoveAll(finalDir); err != nil {
		return fmt.Errorf("remove previous asset output: %w", err)
	}
	if err := os.Rename(tmpDir, finalDir); err != nil {
		return fmt.Errorf("publish processed asset: %w", err)
	}

	if clearKeyMetadata != nil {
		if err := SaveClearKeyMetadata(p.root, clearKeyMetadata); err != nil {
			return err
		}
	}

	if drmMetadata != nil {
		if err := SaveDRMMetadata(p.root, drmMetadata); err != nil {
			return err
		}
	}

	if fairPlayMetadata != nil {
		if err := SaveFairPlayMetadata(p.root, fairPlayMetadata); err != nil {
			return err
		}
	}

	return nil
}

func (p *Processor) processHLS(ctx context.Context, assetID, inputPath, hlsDir string) error {
	for _, profile := range abrProfiles {
		variantDir := filepath.Join(hlsDir, profile.Name)
		if err := os.MkdirAll(variantDir, 0o755); err != nil {
			return fmt.Errorf("create HLS %s directory: %w", profile.Name, err)
		}

		args := []string{
			"-hide_banner",
			"-y",
			"-i", inputPath,
			"-map", "0:v:0",
			"-map", "0:a?",
			"-vf", profile.scaleFilter(),
			"-c:v", "libx264",
			"-preset", "veryfast",
			"-profile:v", "main",
			"-pix_fmt", "yuv420p",
			"-b:v", profile.VideoRate,
			"-maxrate", profile.MaxRate,
			"-bufsize", profile.BufSize,
			"-c:a", "aac",
			"-b:a", "128k",
			"-ar", "48000",
			"-ac", "2",
			"-f", "hls",
			"-hls_time", "4",
			"-hls_playlist_type", "vod",
			"-hls_segment_filename", filepath.Join(variantDir, "segment_%03d.ts"),
			"-hls_base_url", "/assets/" + assetID + "/hls/" + profile.Name + "/",
			filepath.Join(variantDir, "index.m3u8"),
		}

		if err := p.runFFmpeg(ctx, args, "", "hls "+profile.Name); err != nil {
			return err
		}
	}

	return writeHLSMaster(assetID, hlsDir)
}

func writeHLSMaster(assetID, hlsDir string) error {
	var master strings.Builder
	master.WriteString("#EXTM3U\n")
	master.WriteString("#EXT-X-VERSION:3\n")
	master.WriteString("#EXT-X-INDEPENDENT-SEGMENTS\n")

	for _, profile := range abrProfiles {
		fmt.Fprintf(&master,
			"#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d\n",
			profile.Bandwidth,
			profile.Width,
			profile.Height,
		)
		fmt.Fprintf(&master, "/assets/%s/hls/%s/index.m3u8\n", assetID, profile.Name)
	}

	if err := os.WriteFile(filepath.Join(hlsDir, "index.m3u8"), []byte(master.String()), 0o644); err != nil {
		return fmt.Errorf("write HLS master playlist: %w", err)
	}
	return nil
}

func (p *Processor) processDASH(ctx context.Context, assetID, inputPath, dashDir string) error {
	hasAudio := p.hasAudio(ctx, inputPath)

	// Encode each rendition separately to keep peak memory low in constrained containers.
	workDir := filepath.Join(filepath.Dir(dashDir), "dash_work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fmt.Errorf("create DASH work dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	type renditionMP4 struct {
		videoPath string
		audioPath string
		profile   renditionProfile
	}
	renditions := make([]renditionMP4, 0, len(abrProfiles))

	for _, profile := range abrProfiles {
		outPath := filepath.Join(workDir, profile.Name+".mp4")
		args := []string{
			"-hide_banner", "-y",
			"-i", inputPath,
			"-map", "0:v:0",
			"-vf", profile.scaleFilter(),
			"-c:v", "libx264",
			"-preset", "veryfast",
			"-profile:v", "main",
			"-pix_fmt", "yuv420p",
			"-b:v", profile.VideoRate,
			"-maxrate", profile.MaxRate,
			"-bufsize", profile.BufSize,
			"-an",
			outPath,
		}
		if err := p.runFFmpeg(ctx, args, "", "dash video "+profile.Name); err != nil {
			return err
		}

		r := renditionMP4{videoPath: outPath, profile: profile}

		if hasAudio {
			audioPath := filepath.Join(workDir, profile.Name+"_audio.mp4")
			aargs := []string{
				"-hide_banner", "-y",
				"-i", inputPath,
				"-map", "0:a:0",
				"-vn",
				"-c:a", "aac",
				"-b:a", "128k",
				"-ar", "48000",
				"-ac", "2",
				audioPath,
			}
			if err := p.runFFmpeg(ctx, aargs, "", "dash audio "+profile.Name); err != nil {
				return err
			}
			r.audioPath = audioPath
		}
		renditions = append(renditions, r)
	}

	// Build a single DASH mux from the pre-encoded per-rendition MP4s.
	args := []string{"-hide_banner", "-y"}
	for _, r := range renditions {
		args = append(args, "-i", r.videoPath)
	}
	audioIdx := -1
	if hasAudio && len(renditions) > 0 && renditions[0].audioPath != "" {
		args = append(args, "-i", renditions[0].audioPath)
		audioIdx = len(renditions)
	}
	for i := range renditions {
		args = append(args, "-map", fmt.Sprintf("%d:v:0", i))
	}
	if audioIdx >= 0 {
		args = append(args, "-map", fmt.Sprintf("%d:a:0", audioIdx))
	}
	args = append(args, "-c:v", "copy")
	if audioIdx >= 0 {
		args = append(args, "-c:a", "copy",
			"-adaptation_sets", fmt.Sprintf("id=0,streams=%s id=1,streams=%d", videoStreamIndexes(), len(renditions)),
		)
	} else {
		args = append(args, "-adaptation_sets", "id=0,streams="+videoStreamIndexes())
	}
	args = append(args,
		"-f", "dash",
		"-seg_duration", "4",
		"-use_template", "1",
		"-use_timeline", "1",
		"-init_seg_name", "init-$RepresentationID$.m4s",
		"-media_seg_name", "chunk-$RepresentationID$-$Number%05d$.m4s",
		"index.mpd",
	)

	if err := p.runFFmpeg(ctx, args, dashDir, "dash mux"); err != nil {
		return err
	}

	manifestPath := filepath.Join(dashDir, "index.mpd")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read DASH manifest: %w", err)
	}
	manifest := addDASHBaseURL(string(manifestBytes), "/assets/"+assetID+"/dash/")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		return fmt.Errorf("write DASH manifest: %w", err)
	}

	return nil
}

func (p *Processor) processClearKeyDASH(ctx context.Context, assetID, inputPath, outDir string, metadata *ClearKeyMetadata) error {
	workDir := filepath.Join(filepath.Dir(outDir), "clearkey_work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fmt.Errorf("create ClearKey work directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	for _, profile := range abrProfiles {
		args := []string{
			"-hide_banner",
			"-y",
			"-i", inputPath,
			"-map", "0:v:0",
			"-vf", profile.scaleFilter(),
			"-an",
			"-c:v", "libx264",
			"-preset", "veryfast",
			"-profile:v", "main",
			"-pix_fmt", "yuv420p",
			"-b:v", profile.VideoRate,
			"-maxrate", profile.MaxRate,
			"-bufsize", profile.BufSize,
			"-movflags", "+faststart",
			filepath.Join(workDir, profile.Name+".mp4"),
		}
		if err := p.runFFmpeg(ctx, args, "", "clearkey source "+profile.Name); err != nil {
			return err
		}
	}

	hasAudio := p.hasAudio(ctx, inputPath)
	if hasAudio {
		args := []string{
			"-hide_banner",
			"-y",
			"-i", inputPath,
			"-vn",
			"-map", "0:a:0",
			"-c:a", "aac",
			"-b:a", "128k",
			"-ar", "48000",
			"-ac", "2",
			"-movflags", "+faststart",
			filepath.Join(workDir, "audio.mp4"),
		}
		if err := p.runFFmpeg(ctx, args, "", "clearkey source audio"); err != nil {
			return err
		}
	}

	args := make([]string, 0, len(abrProfiles)+8)
	for _, profile := range abrProfiles {
		args = append(args, fmt.Sprintf("input=%s,stream=video,output=%s",
			filepath.Join(workDir, profile.Name+".mp4"),
			filepath.Join(outDir, profile.Name+".mp4"),
		))
	}
	if hasAudio {
		args = append(args, fmt.Sprintf("input=%s,stream=audio,output=%s",
			filepath.Join(workDir, "audio.mp4"),
			filepath.Join(outDir, "audio.mp4"),
		))
	}
	args = append(args,
		"--enable_raw_key_encryption",
		"--keys", "key_id="+metadata.KIDHex+":key="+metadata.KeyHex,
		"--mpd_output", filepath.Join(outDir, "index.mpd"),
	)

	if err := p.runPackager(ctx, args, "clearkey dash"); err != nil {
		return err
	}

	manifestPath := filepath.Join(outDir, "index.mpd")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read ClearKey DASH manifest: %w", err)
	}
	manifest := addDASHBaseURL(string(manifestBytes), "/assets/"+assetID+"/dash_clearkey/")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		return fmt.Errorf("write ClearKey DASH manifest: %w", err)
	}

	return nil
}

func (p *Processor) processWidevinePackagedDASH(ctx context.Context, assetID, inputPath, outDir string, metadata *DRMMetadata) error {
	workDir := filepath.Join(filepath.Dir(outDir), "drm_work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fmt.Errorf("create DRM work directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	for _, profile := range abrProfiles {
		args := []string{
			"-hide_banner",
			"-y",
			"-i", inputPath,
			"-map", "0:v:0",
			"-vf", profile.scaleFilter(),
			"-an",
			"-c:v", "libx264",
			"-preset", "veryfast",
			"-profile:v", "main",
			"-pix_fmt", "yuv420p",
			"-b:v", profile.VideoRate,
			"-maxrate", profile.MaxRate,
			"-bufsize", profile.BufSize,
			"-movflags", "+faststart",
			filepath.Join(workDir, profile.Name+".mp4"),
		}
		if err := p.runFFmpeg(ctx, args, "", "drm source "+profile.Name); err != nil {
			return err
		}
	}

	hasAudio := p.hasAudio(ctx, inputPath)
	if hasAudio {
		args := []string{
			"-hide_banner",
			"-y",
			"-i", inputPath,
			"-vn",
			"-map", "0:a:0",
			"-c:a", "aac",
			"-b:a", "128k",
			"-ar", "48000",
			"-ac", "2",
			"-movflags", "+faststart",
			filepath.Join(workDir, "audio.mp4"),
		}
		if err := p.runFFmpeg(ctx, args, "", "drm source audio"); err != nil {
			return err
		}
	}

	args := make([]string, 0, len(abrProfiles)+8)
	for _, profile := range abrProfiles {
		args = append(args, fmt.Sprintf("input=%s,stream=video,output=%s",
			filepath.Join(workDir, profile.Name+".mp4"),
			filepath.Join(outDir, profile.Name+".mp4"),
		))
	}
	if hasAudio {
		args = append(args, fmt.Sprintf("input=%s,stream=audio,output=%s",
			filepath.Join(workDir, "audio.mp4"),
			filepath.Join(outDir, "audio.mp4"),
		))
	}
	args = append(args,
		"--enable_raw_key_encryption",
		"--keys", "key_id="+metadata.KIDHex+":key="+metadata.KeyHex,
		"--protection_systems", "Widevine,PlayReady,CommonSystem",
		"--mpd_output", filepath.Join(outDir, "index.mpd"),
	)

	if err := p.runPackager(ctx, args, "widevine dash"); err != nil {
		return err
	}

	manifestPath := filepath.Join(outDir, "index.mpd")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read DRM DASH manifest: %w", err)
	}
	manifest := addDASHBaseURL(string(manifestBytes), "/assets/"+assetID+"/dash_drm/")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		return fmt.Errorf("write DRM DASH manifest: %w", err)
	}

	return nil
}

func (p *Processor) processFairPlayHLS(ctx context.Context, assetID, inputPath, outDir string, metadata *FairPlayMetadata) error {
	workDir := filepath.Join(filepath.Dir(outDir), "fairplay_work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fmt.Errorf("create FairPlay work directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	for _, profile := range abrProfiles {
		args := []string{
			"-hide_banner",
			"-y",
			"-i", inputPath,
			"-map", "0:v:0",
			"-vf", profile.scaleFilter(),
			"-an",
			"-c:v", "libx264",
			"-preset", "veryfast",
			"-profile:v", "main",
			"-pix_fmt", "yuv420p",
			"-b:v", profile.VideoRate,
			"-maxrate", profile.MaxRate,
			"-bufsize", profile.BufSize,
			"-movflags", "+faststart",
			filepath.Join(workDir, profile.Name+".mp4"),
		}
		if err := p.runFFmpeg(ctx, args, "", "fairplay source "+profile.Name); err != nil {
			return err
		}
	}

	hasAudio := p.hasAudio(ctx, inputPath)
	if hasAudio {
		args := []string{
			"-hide_banner",
			"-y",
			"-i", inputPath,
			"-vn",
			"-map", "0:a:0",
			"-c:a", "aac",
			"-b:a", "128k",
			"-ar", "48000",
			"-ac", "2",
			"-movflags", "+faststart",
			filepath.Join(workDir, "audio.mp4"),
		}
		if err := p.runFFmpeg(ctx, args, "", "fairplay source audio"); err != nil {
			return err
		}
	}

	for _, profile := range abrProfiles {
		variantDir := filepath.Join(outDir, profile.Name)
		if err := os.MkdirAll(variantDir, 0o755); err != nil {
			return fmt.Errorf("create FairPlay %s directory: %w", profile.Name, err)
		}
	}
	if hasAudio {
		if err := os.MkdirAll(filepath.Join(outDir, "audio"), 0o755); err != nil {
			return fmt.Errorf("create FairPlay audio directory: %w", err)
		}
	}

	args := make([]string, 0, len(abrProfiles)+8)
	for _, profile := range abrProfiles {
		args = append(args, fmt.Sprintf("input=%s,stream=video,playlist_name=%s/index.m3u8,segment_template=%s/segment_$Number%%05d$.ts",
			filepath.Join(workDir, profile.Name+".mp4"),
			profile.Name,
			profile.Name,
		))
	}
	if hasAudio {
		args = append(args, fmt.Sprintf("input=%s,stream=audio,playlist_name=audio/index.m3u8,segment_template=audio/segment_$Number%%05d$.ts",
			filepath.Join(workDir, "audio.mp4"),
		))
	}
	args = append(args,
		"--enable_raw_key_encryption",
		"--keys", "key_id="+metadata.KIDHex+":key="+metadata.KeyHex,
		"--protection_systems", "FairPlay",
		"--hls_key_uri", "skd://fairplay/"+metadata.KIDHex,
		"--hls_master_playlist_output", filepath.Join(outDir, "index.m3u8"),
	)

	if err := p.runPackager(ctx, args, "fairplay hls"); err != nil {
		return err
	}

	return rewriteFairPlayMasterPlaylist(outDir, assetID)
}

func rewriteFairPlayMasterPlaylist(outDir, assetID string) error {
	manifestPath := filepath.Join(outDir, "index.m3u8")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read FairPlay master playlist: %w", err)
	}

	lines := strings.Split(string(manifestBytes), "\n")
	prefix := "/assets/" + assetID + "/hls_fairplay/"
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
			continue
		}
		lines[i] = prefix + trimmed
	}

	if err := os.WriteFile(manifestPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return fmt.Errorf("write FairPlay master playlist: %w", err)
	}
	return nil
}

func videoStreamIndexes() string {
	indexes := make([]string, 0, len(abrProfiles))
	for i := range abrProfiles {
		indexes = append(indexes, fmt.Sprintf("%d", i))
	}
	return strings.Join(indexes, ",")
}

func (p renditionProfile) scaleFilter() string {
	return fmt.Sprintf("scale=w=%d:h=%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1,format=yuv420p",
		p.Width,
		p.Height,
		p.Width,
		p.Height,
	)
}

func (p *Processor) hasAudio(ctx context.Context, inputPath string) bool {
	cmd := exec.CommandContext(ctx, p.ffprobePath(), "-v", "error", "-select_streams", "a:0", "-show_entries", "stream=index", "-of", "csv=p=0", inputPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.TrimSpace(out.String()) != ""
}

func (p *Processor) ffprobePath() string {
	if filepath.Base(p.ffmpegPath) == "ffmpeg" {
		dir := filepath.Dir(p.ffmpegPath)
		if dir == "." {
			return "ffprobe"
		}
		return filepath.Join(dir, "ffprobe")
	}
	return "ffprobe"
}

func (p *Processor) runFFmpeg(ctx context.Context, args []string, dir, label string) error {
	cmd := exec.CommandContext(ctx, p.ffmpegPath, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var ffmpegOutput bytes.Buffer
	cmd.Stdout = &ffmpegOutput
	cmd.Stderr = &ffmpegOutput

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg %s failed: %w: %s", label, err, tail(ffmpegOutput.String(), 2400))
	}
	return nil
}

func (p *Processor) runPackager(ctx context.Context, args []string, label string) error {
	cmd := exec.CommandContext(ctx, p.shakaPackager, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("packager %s failed: %w: %s", label, err, tail(output.String(), 2400))
	}
	return nil
}

func addDASHBaseURL(manifest, baseURL string) string {
	if strings.Contains(manifest, "<BaseURL>"+baseURL+"</BaseURL>") {
		return manifest
	}
	mpdStart := strings.Index(manifest, "<MPD")
	if mpdStart == -1 {
		return manifest
	}
	idx := strings.Index(manifest[mpdStart:], ">")
	if idx == -1 {
		return manifest
	}
	idx += mpdStart
	return manifest[:idx+1] + "\n  <BaseURL>" + baseURL + "</BaseURL>" + manifest[idx+1:]
}

func tail(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[len(s)-max:]
}
