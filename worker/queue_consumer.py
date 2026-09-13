"""Redis queue consumer — blocking pop matching storage-relay's LPush onto
the same list key, so jobs are processed strictly FIFO, one at a time."""

import logging

import redis

log = logging.getLogger("voxa.worker.queue")

QUEUE_KEY = "voxa:jobs:queue"


class QueueConsumer:
    def __init__(self, redis_url: str) -> None:
        self._redis = redis.Redis.from_url(redis_url)

    def next_job_id(self, timeout_seconds: int = 5) -> str | None:
        """Blocks up to timeout_seconds waiting for a job; returns None on
        timeout so the caller's loop can check for shutdown signals."""
        result = self._redis.brpop([QUEUE_KEY], timeout=timeout_seconds)
        if result is None:
            return None
        _, job_id = result
        return job_id.decode("utf-8")
