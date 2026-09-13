"""pyannote.audio diarization step (CPU) — runs after whisper has already
released the GPU, then speaker labels are merged onto whisper's segments
by time overlap."""

import logging

from pyannote.audio import Pipeline

log = logging.getLogger("voxa.worker.diarize")


class Diarizer:
    def __init__(self, hf_token: str | None) -> None:
        log.info("loading pyannote speaker-diarization pipeline (CPU)")
        self._pipeline = Pipeline.from_pretrained(
            "pyannote/speaker-diarization-3.1", use_auth_token=hf_token
        )

    def diarize(self, audio_path: str) -> list[tuple[float, float, str]]:
        """Returns (start, end, speaker_label) turns."""
        diarization = self._pipeline(audio_path)
        turns = []
        for turn, _, speaker in diarization.itertracks(yield_label=True):
            turns.append((turn.start, turn.end, speaker))
        return turns


def merge_speakers(segments: list[dict], turns: list[tuple[float, float, str]]) -> list[dict]:
    """Assigns each whisper segment the speaker whose turn overlaps it the
    most. If diarization found nothing (e.g. mono recording, no second
    speaker detected), segments are returned unchanged with speaker=None."""
    if not turns:
        for seg in segments:
            seg["speaker"] = None
        return segments

    for seg in segments:
        best_speaker = None
        best_overlap = 0.0
        for start, end, speaker in turns:
            overlap = min(seg["end"], end) - max(seg["start"], start)
            if overlap > best_overlap:
                best_overlap = overlap
                best_speaker = speaker
        seg["speaker"] = best_speaker
    return segments
