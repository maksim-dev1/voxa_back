# syntax=docker/dockerfile:1
# voxa-worker — GPU transcription + diarization, runs on nexus.
# cudnn-runtime base matches what ctranslate2 (faster-whisper's backend)
# needs at runtime; driver on nexus (580.173.02) is new enough to run
# this CUDA 12.4 userspace via backward compatibility.
FROM nvidia/cuda:12.4.1-cudnn-runtime-ubuntu22.04

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 python3-pip ffmpeg && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
# A cache mount (not --no-cache-dir) so pip's download cache survives
# across builds even when this layer gets invalidated by a
# requirements.txt change — without it, every edit re-downloads the
# whole list from scratch (torch alone is 2GB+), turning a one-line pin
# bump into an 11+ minute rebuild instead of a few seconds for the
# actually-changed packages.
RUN --mount=type=cache,target=/root/.cache/pip \
    pip3 install -r requirements.txt

COPY . .

RUN useradd -u 10001 -m voxa && chown -R voxa:voxa /app
USER voxa

# Model weights (whisper + pyannote) are cached under /data/voxa/models,
# mounted from the host — see docker-compose.yml. Without the mount this
# still works, just re-downloads on every container recreate.
ENV HF_HOME=/data/voxa/models \
    XDG_CACHE_HOME=/data/voxa/models

ENTRYPOINT ["python3", "main.py"]
