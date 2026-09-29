"""
LearnQuest Textbook Fetcher
----------------------------
Continuously scans the curriculum for subject/topic pairs that don't have
a study note yet, asks Gemini to write one - an original, condensed note
covering that topic, in the same spirit as how Awajis writes its own
summaries of prescribed texts rather than reproducing them - and saves it
straight into the LearnQuest library. Students read it inside the app;
nothing here links out to an external site.

Because this generates original content instead of searching for an
existing resource, every topic can be covered - there's no dependency on
whether some outside site happens to already have a matching page.

Setup:
    cd scripts/textbook_fetcher
    pip install -r requirements.txt
    cp .env.example .env
    # edit .env: add your GEMINI_API_KEY and set INGEST_KEY to match
    # INGEST_SECRET in backend/.env (or the backend's default dev value)

Run once - good for testing, or for scheduling with cron instead of --loop:
    python fetch_textbooks.py --once

Run continuously - rechecks for newly-added curriculum topics every
POLL_INTERVAL_HOURS (default 6h), skipping topics already covered:
    python fetch_textbooks.py

Force a full re-check (e.g. after improving the prompt) - existing notes
get overwritten with the new version, not duplicated:
    python fetch_textbooks.py --once --reset
"""

import argparse
import json
import os
import re
import sys
import time
from pathlib import Path

import requests
from dotenv import load_dotenv
from google import genai
from google.genai.types import GenerateContentConfig, GoogleSearch, Tool

load_dotenv()

GEMINI_API_KEY = os.getenv("GEMINI_API_KEY", "")
GEMINI_MODEL = os.getenv("GEMINI_MODEL", "gemini-3.5-flash")
BACKEND_URL = os.getenv("BACKEND_URL", "http://localhost:8080").rstrip("/")
INGEST_KEY = os.getenv("INGEST_KEY", "")
POLL_INTERVAL_HOURS = float(os.getenv("POLL_INTERVAL_HOURS", "6"))
REQUEST_DELAY_SECONDS = float(os.getenv("REQUEST_DELAY_SECONDS", "3"))
# Google Search grounding is OFF by default. As of the free tier cuts in
# December 2025, grounded requests trigger 429 RESOURCE_EXHAUSTED almost
# immediately on free-tier keys, independent of actual usage — plain
# generation (no tools) has a much more workable free quota. Flip this on
# in .env once you have billing enabled, if you want it back.
USE_GROUNDING = os.getenv("USE_GROUNDING", "false").lower() == "true"
MAX_RETRIES = int(os.getenv("MAX_RETRIES", "3"))
RETRY_BACKOFF_SECONDS = float(os.getenv("RETRY_BACKOFF_SECONDS", "20"))

STATE_FILE = Path(__file__).parent / "processed_topics.json"


def fail(message: str) -> None:
    print(f"\n{message}\n", file=sys.stderr)
    sys.exit(1)


if not GEMINI_API_KEY:
    fail(
        "GEMINI_API_KEY is not set. Get a free key at https://aistudio.google.com/apikey "
        "and put it in scripts/textbook_fetcher/.env"
    )
if not INGEST_KEY:
    fail(
        "INGEST_KEY is not set in .env. It must match INGEST_SECRET in backend/.env "
        "(or the backend's insecure dev default, if you haven't set one)."
    )

client = genai.Client(api_key=GEMINI_API_KEY)


# ---------- local "already covered" tracking ----------

def load_processed() -> set:
    if STATE_FILE.exists():
        return set(json.loads(STATE_FILE.read_text()))
    return set()


def save_processed(processed: set) -> None:
    STATE_FILE.write_text(json.dumps(sorted(processed), indent=2))


# ---------- talking to the backend ----------

def fetch_curriculum() -> dict:
    res = requests.get(f"{BACKEND_URL}/api/curriculum", timeout=15)
    res.raise_for_status()
    body = res.json()
    # LearnQuest wraps responses in {success, data}; unwrap it.
    return body["data"] if isinstance(body, dict) and "data" in body else body


def ingest_note(subject: str, topic: str, title: str, content: str) -> bool:
    if not title.strip() or not content.strip():
        return False
    res = requests.post(
        f"{BACKEND_URL}/api/library/notes",
        json={"subject": subject, "topic": topic, "title": title.strip(), "content": content.strip()},
        headers={"X-Ingest-Key": INGEST_KEY},
        timeout=30,
    )
    return res.status_code in (200, 201)


# ---------- talking to Gemini ----------

def build_prompt(subject: str, topic: str) -> str:
    return f"""Write a condensed, original study note covering the topic "{topic}"
within the subject "{subject}", for a Nigerian secondary school student
(JSS/SS level) preparing for WAEC/NECO.

Write this entirely in your own words - do not copy or closely paraphrase
text from any specific existing textbook. Explain the core concept(s)
clearly, include the key formula or definition where relevant, walk
through one worked example if the topic involves calculation, and end
with a short tip on how WAEC/NECO tends to test this topic.

Length: roughly 4-6 short paragraphs - enough to actually teach the
topic, not just a one-line summary. Plain text paragraphs, no markdown
headers or bullet points.

Return ONLY a JSON object (no markdown formatting, no commentary before
or after it), shaped like:
{{
  "title": "a clear, specific title for this note",
  "content": "the full note text, as plain paragraphs separated by newlines"
}}"""


def extract_json_object(text: str) -> dict:
    text = (text or "").strip()
    text = re.sub(r"^```(?:json)?\s*", "", text)
    text = re.sub(r"\s*```$", "", text)
    match = re.search(r"\{.*\}", text, re.DOTALL)
    if not match:
        return {}
    try:
        parsed = json.loads(match.group(0))
        return parsed if isinstance(parsed, dict) else {}
    except json.JSONDecodeError:
        return {}


def ask_gemini(subject: str, topic: str) -> dict:
    config_kwargs = {}
    if USE_GROUNDING:
        config_kwargs["tools"] = [Tool(google_search=GoogleSearch())]

    last_error = None
    for attempt in range(1, MAX_RETRIES + 1):
        try:
            response = client.models.generate_content(
                model=GEMINI_MODEL,
                contents=build_prompt(subject, topic),
                config=GenerateContentConfig(**config_kwargs),
            )
            return extract_json_object(response.text)
        except Exception as e:
            last_error = e
            is_quota_error = "RESOURCE_EXHAUSTED" in str(e) or "429" in str(e)
            if is_quota_error and attempt < MAX_RETRIES:
                wait = RETRY_BACKOFF_SECONDS * attempt
                print(f"    Rate limited — waiting {wait}s before retry {attempt}/{MAX_RETRIES - 1}...")
                time.sleep(wait)
                continue
            raise last_error


# ---------- main loop ----------

def process_topic(subject: str, topic: str) -> bool:
    print(f"  Writing notes: {subject} - {topic}")
    try:
        result = ask_gemini(subject, topic)
    except Exception as e:  # Gemini errors shouldn't kill the whole run
        print(f"    Gemini call failed: {e}")
        return False

    title = (result.get("title") or "").strip()
    content = (result.get("content") or "").strip()

    if not title or not content:
        print("    Gemini didn't return usable content - skipping.")
        return False

    if ingest_note(subject, topic, title, content):
        print(f"    Added: {title}")
        return True

    print(f"    Backend rejected: {title}")
    return False


def run_once() -> None:
    processed = load_processed()
    curriculum = fetch_curriculum()

    total_added = 0
    total_failed = 0
    for subject in curriculum.get("subjects", []):
        subject_name = subject["name"]
        for topic in subject.get("topics", []):
            key = f"{subject_name}::{topic}"
            if key in processed:
                continue
            # Only mark as processed on real success — a topic that failed
            # (quota error, bad response, etc.) needs to be retried on the
            # next run, not silently skipped forever.
            if process_topic(subject_name, topic):
                total_added += 1
                processed.add(key)
                save_processed(processed)  # persist after every success, not just at the end
            else:
                total_failed += 1
            time.sleep(REQUEST_DELAY_SECONDS)

    print(f"\nPass complete - {total_added} new notes added, {total_failed} failed (will retry next run), {len(processed)} topics covered in total.")


def main() -> None:
    parser = argparse.ArgumentParser(description="Generate original study notes for LearnQuest's library")
    parser.add_argument("--once", action="store_true", help="Run a single pass and exit (good for cron)")
    parser.add_argument("--reset", action="store_true", help="Forget which topics were already checked and redo all of them")
    args = parser.parse_args()

    if args.reset and STATE_FILE.exists():
        STATE_FILE.unlink()
        print("Cleared saved progress - every topic will be re-checked.\n")

    if args.once:
        run_once()
        return

    print(f"Running continuously - rechecking for new topics every {POLL_INTERVAL_HOURS}h. Ctrl+C to stop.\n")
    while True:
        run_once()
        print(f"\nSleeping {POLL_INTERVAL_HOURS}h before the next pass...\n")
        time.sleep(POLL_INTERVAL_HOURS * 3600)


if __name__ == "__main__":
    main()
