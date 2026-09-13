"""faster-whisper transcription step (GPU)."""

import logging

from faster_whisper import WhisperModel

log = logging.getLogger("voxa.worker.transcribe")


class Transcriber:
    """Loads the whisper model once at process start — reloading per job
    would dominate runtime on this GPU."""

    def __init__(self, model_size: str, device: str, compute_type: str) -> None:
        log.info(
            "loading faster-whisper model=%s device=%s compute_type=%s",
            model_size,
            device,
            compute_type,
        )
        self._model = WhisperModel(model_size, device=device, compute_type=compute_type)

    def transcribe(self, audio_path: str) -> list[dict]:
        """Returns raw whisper segments (no speaker labels yet — diarize.py
        adds those in a second pass)."""
        segments, info = self._model.transcribe(audio_path, vad_filter=True)
        log.info(
            "transcribed %s: language=%s duration=%.1fs",
            audio_path,
            info.language,
            info.duration,
        )

        result = []
        for seg in segments:
            result.append(
                {
                    "start": seg.start,
                    "end": seg.end,
                    "text": seg.text.strip(),
                }
            )
        return result
