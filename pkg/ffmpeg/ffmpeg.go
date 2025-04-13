package ffmpeg

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type FFmpeg struct {
	ffmpegPath string
	cmd        *exec.Cmd
	startArgs  []string
	endArgs    []string
	errs       []string
}

func NewFfmpeg(ffmpegPath string) (*FFmpeg, error) {
	f := &FFmpeg{
		ffmpegPath: ffmpegPath,
	}

	if _, err := os.Stat("tmp"); os.IsNotExist(err) {
		err := os.Mkdir("tmp", 0755)
		if err != nil {
			return nil, err
		}
	}

	if _, err := os.Stat(f.GetFileWithExt()); err == nil || ffmpegPath != "" {
		return f, nil
	}

	tempFile, err := os.Create(f.GetFileWithExt())
	if err != nil {
		return nil, fmt.Errorf("failed to create ffmpeg file: %w", err)
	}

	fileData, err := fs.ReadFile(ffmpeg, fmt.Sprintf("bin/ffmpeg-%s-%s", runtime.GOOS, runtime.GOARCH))
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded binary: %w", err)
	}

	if _, err := tempFile.Write(fileData); err != nil {
		return nil, fmt.Errorf("failed to write binary data: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temporary file: %w", err)
	}

	if err := os.Chmod(tempFile.Name(), 0755); err != nil {
		return nil, fmt.Errorf("failed to set executable permissions: %w", err)
	}

	return f, nil
}

func (f *FFmpeg) GetFileWithExt() string {
	if f.ffmpegPath != "" {
		return f.ffmpegPath
	}

	if runtime.GOOS == "windows" {
		return "tmp/ffmpeg.exe"
	}

	return "tmp/ffmpeg"
}

func (f *FFmpeg) Execute(inputPath []string, outputPath string) error {
	if len(f.errs) > 0 {
		return errors.New(strings.Join(f.errs, "\n"))
	}

	args := f.startArgs
	for _, path := range inputPath {
		args = append(args, "-i", path)
	}
	args = append(args, f.endArgs...)
	args = append(args, outputPath)

	fmt.Println(args)
	f.cmd = exec.Command(f.GetFileWithExt(), args...)
	f.cmd.SysProcAttr = GetSysProcAttr()

	f.cmd.Stdout = os.Stdout
	f.cmd.Stderr = os.Stderr

	if err := f.cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (f *FFmpeg) Clear() *FFmpeg {
	f.startArgs = f.startArgs[:0]
	f.endArgs = f.endArgs[:0]
	return f
}

// LogLevel is an analog of the -loglevel parameter in ffmpeg
func (f *FFmpeg) LogLevel(logLevel string) *FFmpeg {
	f.endArgs = append(f.endArgs, "-loglevel", logLevel)
	return f
}

// Yes is an analog of the -y parameter in ffmpeg (overwrite output files without asking)
func (f *FFmpeg) Yes() *FFmpeg {
	f.endArgs = append(f.endArgs, "-y")
	return f
}

// ErrDetect is an analog of the -err_detect parameter in ffmpeg
func (f *FFmpeg) ErrDetect(errDetect string) *FFmpeg {
	f.endArgs = append(f.endArgs, "-err_detect", errDetect)
	return f
}

// VideoCodec is an analog of the -c:v parameter in ffmpeg (or -vn for no video)
func (f *FFmpeg) VideoCodec(videoCodec string) *FFmpeg {
	if videoCodec == "" || videoCodec == "none" {
		f.endArgs = append(f.endArgs, "-vn")
	} else {
		f.endArgs = append(f.endArgs, "-c:v", videoCodec)
	}

	return f
}

// AudioCodec is an analog of the -c:a parameter in ffmpeg (or -an for no audio)
func (f *FFmpeg) AudioCodec(audioCodec string) *FFmpeg {
	if audioCodec == "" || audioCodec == "none" {
		f.endArgs = append(f.endArgs, "-an")
	} else {
		f.endArgs = append(f.endArgs, "-c:a", audioCodec)
	}

	return f
}

// AudioChannels is an analog of the -ac parameter in ffmpeg
func (f *FFmpeg) AudioChannels(audioChannels int) *FFmpeg {
	if audioChannels < 1 {
		f.errs = append(f.errs, "ac must be ≥ 1")
	}
	f.endArgs = append(f.endArgs, "-ac", fmt.Sprint(audioChannels))

	return f
}

// AudioRate is an analog of the -ar parameter in ffmpeg
func (f *FFmpeg) AudioRate(audioRate int) *FFmpeg {
	if audioRate < 1 {
		f.errs = append(f.errs, "ar must be ≥ 1")
	}
	f.endArgs = append(f.endArgs, "-ar", fmt.Sprint(audioRate))

	return f
}

// Async is an analog of the -async parameter in ffmpeg
func (f *FFmpeg) Async(async int) *FFmpeg {
	f.endArgs = append(f.endArgs, "-async", fmt.Sprint(async))
	return f
}

// Vsync is an analog of the -async parameter in ffmpeg
func (f *FFmpeg) Vsync(vsync int) *FFmpeg {
	f.endArgs = append(f.endArgs, "-vsync", fmt.Sprint(vsync))
	return f
}

// FpsMode is an analog of the -fps_mode parameter in ffmpeg
func (f *FFmpeg) FpsMode(fpsMode string) *FFmpeg {
	f.endArgs = append(f.endArgs, "-fps_mode", fpsMode)
	return f
}

// Start is an analog of the -ss parameter in ffmpeg (set start time offset)
func (f *FFmpeg) Start(start string) *FFmpeg {
	f.endArgs = append(f.endArgs, "-ss", start)
	return f
}

// End is an analog of the -to parameter in ffmpeg (record or transcode stop time)
func (f *FFmpeg) End(end string) *FFmpeg {
	f.endArgs = append(f.endArgs, "-to", end)
	return f
}

// Duration is an analog of the -t parameter in ffmpeg (record or transcode duration)
func (f *FFmpeg) Duration(dur string) *FFmpeg {
	f.endArgs = append(f.endArgs, "-t", dur)
	return f
}

// Format is an analog of the -f parameter in ffmpeg
func (f *FFmpeg) Format(format string) *FFmpeg {
	f.startArgs = append(f.startArgs, "-f", format)
	return f
}

// Safe is an analog of the -safe parameter in ffmpeg
func (f *FFmpeg) Safe(safe int) *FFmpeg {
	f.endArgs = append(f.endArgs, "-safe", fmt.Sprint(safe))
	return f
}

// ExtraArgs appends additional command-line arguments to the ffmpeg command.
// Arguments should be provided as a []string slice, where flags and their values
// are passed sequentially, e.g., []string{"-c:v", "copy", "-preset", "fast"}.
// For boolean flags (no value), pass only the flag: []string{"-nostdin"}.
func (f *FFmpeg) ExtraArgs(extraArgs []string) *FFmpeg {
	f.endArgs = append(f.endArgs, extraArgs...)
	return f
}
