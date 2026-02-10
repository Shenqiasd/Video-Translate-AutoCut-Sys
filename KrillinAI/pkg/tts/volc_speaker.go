package tts

import "strings"

func IsVolcSpeakerID(voice string) bool {
	return strings.HasPrefix(voice, "S_")
}
