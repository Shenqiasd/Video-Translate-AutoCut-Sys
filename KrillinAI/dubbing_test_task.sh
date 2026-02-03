#!/bin/bash

curl -X POST http://127.0.0.1:8888/api/capability/subtitleTask \
-H "Content-Type: application/json" \
-d '{
    "language": "zh_cn",
    "origin_lang": "en",
    "target_lang": "zh_cn",
    "bilingual": 1,
    "translation_subtitle_pos": 1,
    "tts": 1,
    "tts_provider": "edge-tts",
    "tts_voice_code": "zh-CN-XiaoxiaoNeural",
    "embed_subtitle_video_type": "vertical",
    "url": "https://www.youtube.com/shorts/ga5A8CRQnSA"
}'
