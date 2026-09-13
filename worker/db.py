"""Postgres access for voxa-worker — sync psycopg, one job at a time, no
need for async here since the GPU work itself is blocking."""

import json
from contextlib import contextmanager
from typing import Iterator

import psycopg


class Db:
    def __init__(self, database_url: str) -> None:
        self._conn = psycopg.connect(database_url, autocommit=True)

    def close(self) -> None:
        self._conn.close()

    def job_audio_path(self, job_id: str) -> str | None:
        with self._conn.cursor() as cur:
            cur.execute("select audio_path from jobs where id = %s", (job_id,))
            row = cur.fetchone()
            return row[0] if row else None

    def mark_processing(self, job_id: str) -> None:
        with self._conn.cursor() as cur:
            cur.execute(
                "update jobs set status = 'processing', updated_at = now() where id = %s",
                (job_id,),
            )

    def mark_done(self, job_id: str, segments: list[dict]) -> None:
        with self._conn.cursor() as cur:
            cur.execute(
                """
                insert into transcripts (job_id, segments) values (%s, %s)
                on conflict (job_id) do update set segments = excluded.segments
                """,
                (job_id, json.dumps(segments)),
            )
            cur.execute(
                "update jobs set status = 'done', updated_at = now() where id = %s",
                (job_id,),
            )

    def mark_failed(self, job_id: str, error: str) -> None:
        with self._conn.cursor() as cur:
            cur.execute(
                "update jobs set status = 'failed', error = %s, updated_at = now() where id = %s",
                (error, job_id),
            )


@contextmanager
def connect(database_url: str) -> Iterator[Db]:
    db = Db(database_url)
    try:
        yield db
    finally:
        db.close()
