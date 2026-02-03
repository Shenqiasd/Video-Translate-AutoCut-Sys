package dto

// AnalyzeRequest 请求分析视频
type SmartClipperAnalyzeReq struct {
	Url      string `json:"url"`
	Language string `json:"language"` // 目标语言，默认中文
}

// TopicClip 单个主题片段
type TopicClip struct {
	Id       int    `json:"id"`
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Start    string `json:"start"` // HH:MM:SS
	End      string `json:"end"`   // HH:MM:SS
	Reason   string `json:"reason"`
	Duration int    `json:"duration"` // Seconds
}

// AnalyzeResponse 分析结果
type SmartClipperAnalyzeResData struct {
	VideoTitle string      `json:"video_title"`
	Duration   string      `json:"duration"`
	Clips      []TopicClip `json:"clips"`
	Token      string      `json:"token"` // 用于后续提交任务的凭证（缓存Key）
}

type SmartClipperAnalyzeRes struct {
	Error int32                       `json:"error"`
	Msg   string                      `json:"msg"`
	Data  *SmartClipperAnalyzeResData `json:"data"`
}

// SubmitClipsRequest 提交剪辑任务
type SmartClipperSubmitReq struct {
	Token           string                    `json:"token"`
	SelectedClipIds []int                     `json:"selected_clip_ids"`
	TaskParams      StartVideoSubtitleTaskReq `json:"task_params"` // 复用现有任务参数
}

type SmartClipperSubmitResData struct {
	TaskIds []string `json:"task_ids"`
}

type SmartClipperSubmitRes struct {
	Error int32                      `json:"error"`
	Msg   string                     `json:"msg"`
	Data  *SmartClipperSubmitResData `json:"data"`
}
