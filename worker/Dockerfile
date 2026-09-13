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
RUN pip3 install --no-cache-dir -r requirements.txt

COPY . .

RUN useradd -u 10001 -m voxa && chown -R voxa:voxa /app
USER voxa

# Model weights (whisper + pyannote) are cached under /data/voxa/models,
# mounted from the host — see docker-compose.yml. Without the mount this
# still works, just re-downloads on every container recreate.
ENV HF_HOME=/data/voxa/models \
    XDG_CACHE_HOME=/data/voxa/models

ENTRYPOINT ["python3", "main.py"]
