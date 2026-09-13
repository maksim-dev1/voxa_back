"""Converts recordings to WAV before they hit the ML pipeline.

faster-whisper decodes compressed audio fine (PyAV/ffmpeg-backed), but
pyannote.audio loads through torchaudio's soundfile (libsndfile) backend,
which cannot read AAC/M4A at all — it fails with "Format not recognised."
Converting once up front keeps both steps on the same known-good WAV
input instead of relying on each library's own format support.
"""

import contextlib
import logging
import os
import subprocess
import tempfile
from collections.abc import Iterator

log = logging.getLogger("voxa.worker.audio_convert")


@contextlib.contextmanager
def as_wav(audio_path: str) -> Iterator[str]:
    """Yields a path to a 16kHz mono WAV copy of audio_path, deleted on
    exit. 16kHz mono matches what both whisper and pyannote resample to
    internally anyway, so this doesn't lose anything either model uses."""
    with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
        wav_path = tmp.name

    try:
        subprocess.run(
            [
                "ffmpeg",
                "-y",
                "-i",
                audio_path,
                "-ac",
                "1",
                "-ar",
                "16000",
                "-vn",
                wav_path,
            ],
            check=True,
            capture_output=True,
            text=True,
        )
        log.info("converted %s -> %s", audio_path, wav_path)
        yield wav_path
    except subprocess.CalledProcessError as exc:
        log.error("ffmpeg conversion failed for %s: %s", audio_path, exc.stderr)
        raise
    finally:
        if os.path.exists(wav_path):
            os.remove(wav_path)
