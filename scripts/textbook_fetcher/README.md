# LearnQuest Textbook Fetcher

A Python script that writes original, condensed study notes for every
subject/topic in your curriculum and adds them straight to the Library
section — students pick a subject, pick a topic, and read it right there
in the app. Nothing links out to an external site.

## Why generated notes instead of external links

This mirrors how Awajis (and study-guide products generally) handle
prescribed texts: they write their **own** summaries rather than
reproducing the original book. That's both the safer approach legally —
copying substantial portions of an existing copyrighted textbook, "trimmed
down" or not, is still reproducing it — and the more reliable one
practically, since it doesn't depend on some external site happening to
already have a free page that matches your exact curriculum topic.

The prompt explicitly tells Gemini to write in its own words, not
paraphrase a specific existing book. Google Search grounding is **off by
default** — as of Google's December 2025 free-tier cuts, grounded requests
trigger `429 RESOURCE_EXHAUSTED` almost immediately on free-tier keys,
regardless of actual usage, while plain generation works fine. Set
`USE_GROUNDING=true` in `.env` once you have billing enabled if you want
it back for extra factual accuracy.

## Setup

```bash
cd scripts/textbook_fetcher
pip install -r requirements.txt
cp .env.example .env
```

Get a free Gemini API key at **https://aistudio.google.com/apikey**
(no credit card needed) and put it in `.env` as `GEMINI_API_KEY`.

Make sure `INGEST_KEY` in this `.env` matches `INGEST_SECRET` in
`backend/.env` — if you haven't set one on the backend, its dev default is
`dev-ingest-secret-change-me`, which is already the default here too.

## Run it

Your backend needs to already be running (`go run main.go` in `backend/`),
since this script reads the curriculum from it and writes notes back to it.

**One pass, then exit** — good for testing, or for scheduling with cron:
```bash
python fetch_textbooks.py --once
```

**Continuously** — rechecks for newly-added curriculum topics every
`POLL_INTERVAL_HOURS` (default 6), skipping topics already covered:
```bash
python fetch_textbooks.py
```

**Force a full re-check** (e.g. after you tweak the prompt in the script) —
existing notes get overwritten with the new version, not duplicated:
```bash
python fetch_textbooks.py --once --reset
```

Progress is saved in `processed_topics.json` in this folder as it goes, so
Ctrl+C-ing it mid-run and restarting later doesn't lose work or re-spend
API calls on topics already covered.

## A couple of things worth knowing

- **There's a daily request cap on the free tier** — 500 requests/day/model/project.
  A full curriculum pass is ~680 topics, so expect to need a second day to finish.
  When the cap is hit the script stops immediately (retrying a spent daily quota
  can't help until it resets) and tells you how many topics are left. Unfinished
  topics are never marked processed, so just re-run — it picks up exactly where
  it stopped. Quota resets on a rolling 24h basis, so it may free up sooner.
- **You can get a second daily bucket by switching models.** The cap is
  per-model, so if `GEMINI_MODEL` points at a different model that isn't
  overloaded, you get another 500 requests that day. Check which models are
  responding before assuming you're blocked.
- **Every topic gets covered.** Since this generates content instead of
  searching for a pre-existing resource, there's no "nothing found for
  this topic" case the way there would be with external links — that's
  the main reason this approach replaced the earlier link-finding version.
- **429 vs 503 are handled differently.** A per-minute burst (429) or a
  momentarily overloaded model (503) is retried with backoff. The daily cap
  is detected and stops the pass instead, since backoff can't clear it.
- **429 errors on the very first topic usually mean grounding was on.**
  Make sure `USE_GROUNDING` is `false` (or unset) in `.env` unless you have
  billing enabled — see above. If you're still hitting 429s with grounding
  off, that's the free-tier cap; the script retries briefly, then stops the
  pass rather than looping.
- **Spot-check the output the first time.** The prompt asks for accuracy
  and originality, but it's still a model — skim a few generated notes
  against what you already know before trusting the full set blindly.
