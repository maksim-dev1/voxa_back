"""pyannote.audio diarization step (CPU) — runs after whisper has already
released the GPU, then speaker labels are merged onto whisper's segments
by time overlap."""

import logging

from pyannote.audio import Pipeline

log = logging.getLogger("voxa.worker.diarize")


class Diarizer:
    def __init__(self, hf_token: str | None) -> None:
        log.info("loading pyannote speaker-diarization pipeline (CPU)")
        pipeline = Pipeline.from_pretrained(
            "pyannote/speaker-diarization-3.1", use_auth_token=hf_token
        )
        # from_pretrained() doesn't raise on failure (bad/missing HF_TOKEN,
        # gated model license not accepted) — it logs a warning and
        # returns None, which would otherwise blow up per-job later with a
        # confusing "'NoneType' object is not callable". Fail loud here,
        # at worker startup, instead.
        if pipeline is None:
            raise RuntimeError(
                "pyannote Pipeline.from_pretrained() returned None — check HF_TOKEN is set "
                "and the pyannote/speaker-diarization-3.1 license was accepted on huggingface.co"
            )
        self._pipeline = pipeline

    def diarize(self, audio_path: str) -> list[tuple[float, float, str]]:
        """Returns (start, end, speaker_label) turns. Recordings with no
        detected speech (silence, pure tone, music-only) make pyannote's
        internal clustering blow up with `max() arg is an empty sequence`
        instead of returning zero turns — treat that as "no speakers
        found" rather than failing the whole job."""
        try:
            diarization = self._pipeline(audio_path)
        except ValueError as exc:
            log.warning("diarization found no speech in %s: %s", audio_path, exc)
            return []

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
