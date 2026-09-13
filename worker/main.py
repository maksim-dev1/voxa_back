"""voxa GPU worker (runs on nexus).

Pulls job ids off the Redis queue that voxa-storage-relay pushes to,
transcribes the audio with faster-whisper (GPU), runs pyannote diarization
afterwards (CPU, GPU already released by then), merges speaker labels onto
the transcript, and writes the result back into Postgres.

Single process, one job at a time — the GPU (4GB VRAM, shared with other
nexus workloads) doesn't have room for concurrent transcriptions anyway.
"""

import logging
import signal
import sys

import config
import db
from audio_convert import as_wav
from diarize import Diarizer, merge_speakers
from queue_consumer import QueueConsumer
from transcribe import Transcriber

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(name)s %(levelname)s %(message)s")
log = logging.getLogger("voxa.worker")

_shutdown = False


def _handle_signal(signum, _frame):
    global _shutdown
    log.info("received signal %s, shutting down after current job", signum)
    _shutdown = True


def process_job(job_id: str, database: db.Db, transcriber: Transcriber, diarizer: Diarizer) -> None:
    audio_path = database.job_audio_path(job_id)
    if audio_path is None:
        log.error("job %s not found in database, skipping", job_id)
        return

    log.info("processing job %s (%s)", job_id, audio_path)
    database.mark_processing(job_id)

    try:
        # Both models get the same WAV copy — faster-whisper can decode
        # the original m4a fine on its own, but pyannote's soundfile
        # backend can't read AAC at all, so convert once up front rather
        # than have each step handle formats differently.
        with as_wav(audio_path) as wav_path:
            segments = transcriber.transcribe(wav_path)
            turns = diarizer.diarize(wav_path)
        segments = merge_speakers(segments, turns)
        database.mark_done(job_id, segments)
        log.info("job %s done: %d segments", job_id, len(segments))
    except Exception as exc:  # noqa: BLE001 - a failed job must not crash the worker loop
        log.exception("job %s failed", job_id)
        database.mark_failed(job_id, str(exc))


def main() -> None:
    cfg = config.load()
    signal.signal(signal.SIGTERM, _handle_signal)
    signal.signal(signal.SIGINT, _handle_signal)

    log.info("voxa-worker starting")
    transcriber = Transcriber(cfg.whisper_model, cfg.whisper_device, cfg.whisper_compute_type)
    diarizer = Diarizer(cfg.hf_token)
    queue = QueueConsumer(cfg.redis_url)

    with db.connect(cfg.database_url) as database:
        log.info("ready, waiting for jobs")
        while not _shutdown:
            job_id = queue.next_job_id(timeout_seconds=5)
            if job_id is None:
                continue
            process_job(job_id, database, transcriber, diarizer)

    log.info("voxa-worker stopped")


if __name__ == "__main__":
    sys.exit(main() or 0)
