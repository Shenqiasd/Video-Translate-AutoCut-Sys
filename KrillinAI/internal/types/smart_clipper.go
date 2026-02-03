package types

import "time"

// SmartClipperCacheData 存储在缓存中的分析结果
type SmartClipperCacheData struct {
	VideoId      string
	VideoTitle   string
	VideoPath    string // 下载后的原始视频路径
	SubtitlePath string // 原始字幕路径
	Clips        []ClipInfo
	CreatedAt    time.Time
}

type ClipInfo struct {
	Id       int    `json:"id"`
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Start    string `json:"start"`
	End      string `json:"end"`
	Duration int    `json:"duration"`
}

// Kimi (Moonshot) Prompt
var SmartClipperPrompt = `你是一个专业的视频剪辑师和内容分析专家。
我将提供一份长视频的字幕文件。请根据内容的语义、话题转换点，将其拆分为多个独立主题的精彩短视频片段。
目标是让用户可以只看自己感兴趣的片段。

要求：
1. **完整性**：提取的片段必须包含完整的话题论述，不要在句子中间截断。
2. **独立性**：每个片段应能独立成片，有清晰的开始和结束。
3. **时长控制**：每个片段时长建议在 %d 到 %d 秒之间。如果话题很长，请拆分成 Part 1, Part 2。
4. **标题与摘要**：为每个片段起一个吸引人的标题（中文），并写一段简短摘要（50字以内）。
5. **JSON输出**：结果必须是严格的 JSON 数组格式。

输出 JSON 结构：
[
  {
    "id": 1,
    "start": "HH:MM:SS",
    "end": "HH:MM:SS",
    "title": "片段标题",
    "summary": "片段摘要",
    "reason": "切分理由"
  }
]

请确保时间戳准确对应字幕。

以下是字幕内容：
%s
`
