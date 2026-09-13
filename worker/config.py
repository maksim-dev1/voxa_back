"""Environment-driven settings for voxa-worker."""

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Config:
    database_url: str
    redis_url: str
    whisper_model: str
    whisper_device: str
    whisper_compute_type: str
    hf_token: str | None  # needed to pull the pyannote pretrained pipeline


def load() -> Config:
    database_url = os.environ["DATABASE_URL"]
    redis_url = os.environ["REDIS_URL"]

    return Config(
        database_url=database_url,
        redis_url=redis_url,
        whisper_model=os.environ.get("WHISPER_MODEL", "small"),
        whisper_device=os.environ.get("WHISPER_DEVICE", "cuda"),
        whisper_compute_type=os.environ.get("WHISPER_COMPUTE_TYPE", "int8"),
        hf_token=os.environ.get("HF_TOKEN"),
    )
