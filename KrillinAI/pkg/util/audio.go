package util

import (
	"fmt"
	"go.uber.org/zap"
	"krillin-ai/internal/storage"
	"krillin-ai/log"
	"os/exec"
	"path/filepath"
	"strings"
)

// 把音频处理成单声道、16k采样率
func ProcessAudio(filePath string) (string, error) {
	dest := strings.ReplaceAll(filePath, filepath.Ext(filePath), "_mono_16K.mp3")
	cmdArgs := []string{"-i", filePath, "-ac", "1", "-ar", "16000", "-b:a", "192k", dest}
	cmd := exec.Command(storage.FfmpegPath, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.GetLogger().Error("处理音频失败", zap.Error(err), zap.String("audio file", filePath), zap.String("output", string(output)))
		return "", err
	}
	return dest, nil
}

// MixAudioTracks blends TTS audio with instrumental/background music
// ttsAudioPath: the generated TTS dubbing audio (will be louder)
// instrumentalPath: the background music/instrumental track
// outputPath: where to save the mixed result
// ttsVolume: volume multiplier for TTS (e.g., 1.0 = 100%)
// bgmVolume: volume multiplier for background music (e.g., 0.3 = 30%)
func MixAudioTracks(ttsAudioPath, instrumentalPath, outputPath string, ttsVolume, bgmVolume float64) error {
	// Advanced Mixing Logic: Sidechain Ducking + Loudness Normalization
	// 1. [tts]: TTS Audio (Control Signal)
	// 2. [bgm]: Background Music (Target Signal)
	// Effect: When TTS speaks, BGM volume is reduced (Ducked).
	
	// Complex Filter breakdown:
	// 1. Adjust volumes: TTS standard, BGM slightly boosted (since it will be ducked)
	// 2. sidechaincompress: Ducks [bgm] based on [tts] input.
	//    threshold=0.1: Trigger ducking when TTS signal > 10%
	//    ratio=5: Compression ratio (reduce BGM by 5:1)
	//    attack=100: 100ms fade out (smooth transition)
	//    release=1000: 1000ms fade in (slow return after speech)
	// 3. amix: Combine the Ducked BGM and the Original TTS
	// 4. loudnorm: Normalize final output to -14 LUFS (YouTube standard) to prevent clipping/distortion
	
	filterComplex := fmt.Sprintf(
		"[0:a]volume=%.2f[tts];"+
		"[1:a]volume=%.2f[bgm];"+
		"[bgm][tts]sidechaincompress=threshold=0.08:ratio=6:attack=100:release=800:link=average[ducked_bgm][control_tts];"+
		"[ducked_bgm][control_tts]amix=inputs=2:duration=first[mixed];"+
		"[mixed]loudnorm=I=-14:TP=-1.5:LRA=11[out]",
		ttsVolume, bgmVolume*1.5, // Boost BGM base volume slightly so it's audible during silence
	)

	cmdArgs := []string{
		"-y",
		"-i", ttsAudioPath,       // Input 0
		"-i", instrumentalPath,   // Input 1
		"-filter_complex", filterComplex,
		"-map", "[out]",
		"-ac", "2",      // Stereo output
		"-ar", "44100",  // Standard sample rate
		"-c:a", "aac",   // AAC codec
		"-b:a", "192k",  // High bitrate
		outputPath,
	}

	cmd := exec.Command(storage.FfmpegPath, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.GetLogger().Error("MixAudioTracks failed",
			zap.Error(err),
			zap.String("ttsAudioPath", ttsAudioPath),
			zap.String("instrumentalPath", instrumentalPath),
			zap.String("output", string(output)))
		return err
	}

	log.GetLogger().Info("MixAudioTracks success",
		zap.String("outputPath", outputPath))
	return nil
}

func formatVolume(v float64) string {
	return strings.TrimRight(strings.TrimRight(
		fmt.Sprintf("%.2f", v), "0"), ".")
}

