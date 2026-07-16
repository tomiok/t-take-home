# Take-Home Exercise: Converging Course Requests from Many Sources

**Time:** Up to 2 hours — please don't go over. We're after something real you can walk us through and reason about, not exhaustive coverage.

**AI tools:** Use whatever you'd use on the job — AI included. We're interested in *how* you work with it, so capture your prompts in the **AI Log** section of your README (the full session, or just the ones that mattered).

---

## Background

Each year, students are assigned course requests for the upcoming year — some **required** (graduation requirements, retakes), some **elective** (interest, counselor recommendation). At its core, a course request maps a student to a course in the school's catalog.

But that data doesn't arrive in one clean shape. Every district runs a different Student Information System (SIS), and each exports course requests in its own format — its own identifiers, its own way of expressing "this is a backup choice," its own quirks. Your application needs to work with them **one consistent way**, regardless of which SIS a school uses.

**This exercise is about that convergence problem.** You're given course-request data from four SIS feeds and a shared catalog. Turn them into one consistent internal representation the application can use — and make the seam between "messy external data" and "what the app works with" maintainable and extensible.

---

## What you're given (`data/`)

| Path | What it is |
|------|------------|
| `data/catalog/courses.json` | The district course catalog — the source of truth for valid courses. Includes prerequisite and corequisite relationships. |
| `data/meridian/` | **Meridian High** — a flat CSV of course requests. |
| `data/apex/` | **Apex Charter** — a nested JSON export, plus a course-number crosswalk file. |
| `data/cornerstone/` | **Cornerstone Prep** — two CSVs (a primary file and a separate alternates file). |
| `data/helix/` | **Helix Academy** — a denormalized JSON roster export, plus a course-UUID map file. |

Each school uses exactly one SIS. The four feeds describe **different** students; together they make up the district. There's a per-source data dictionary in the Appendix, but don't assume it tells you everything — read the data closely.

---

## What to build

A small, runnable program (a CLI or a tiny local service — **no UI required**) that:

1. **Ingests all four SIS feeds** and produces **one unified set of course requests** that the application can work with, independent of which SIS each came from.
2. **Runs via a single documented command and writes its unified output to a file** (JSON or similar). The *shape* of that output is your design decision.
3. **Demonstrates that the unified output actually serves the application's needs.** Concretely, show that the app can:
    - list a given student's course requests, clearly separating their required choices from their backup/elective choices, in a sensible order;
    - confirm every request points at a real catalog course, and do something sensible when one doesn't;
    - flag a student who has requested one course of a corequisite pair but not the other (the catalog defines these pairs).

You do **not** need to build a database, authentication, or a UI. Seed/read the data straight from the provided files.

---

## Language & running it

- **We prefer Python** — it's what we work in, and the standard library (`csv`, `json`, `dataclasses`, etc.) is more than enough for this. **If you're stronger in another language, you're more than welcome to use it instead.**
- **Keep dependencies minimal.** If you add any, make them trivial to bootstrap: include the manifest (`pyproject.toml`, `requirements.txt`, `package.json`, etc.) and give **one documented command that runs the project from a clean checkout**.

---

## Assumptions & key decisions

Build a first implementation from what's described here, and as you go **keep a running list** of:

- the assumptions you made where the spec or the data was ambiguous, and
- the questions you'd want to settle with a product stakeholder before this went to production, and why they matter.

Capture that list in the **Design** section of your README (below).

---

## Deliverables

We've included a starter `README.md` with the three sections below already stubbed out — just fill them in.

- A runnable program (GitHub repo or ZIP): the four feeds in, unified output to a file, one documented command to run it.
- A **Quick Start** section in your `README.md`: how to set up and run it — the one documented command, from a clean checkout.
- A **Design** section in your `README.md`. Walk us through the design of your course-request pipeline and its key parts, the way you would in a quick design review with a colleague. Make it clear:
    1. **How your code represents a course request internally, and why that shape.**
    2. **How you decide that two records are the same student, or the same course.**
    3. **What you did when the data was wrong, missing, or didn't line up — and why.**

  Fold in the running list of assumptions and stakeholder questions here too.
- An **AI Log** section in your `README.md` with the prompts you used — the full session, or just the ones that mattered. If you didn't use AI, say so.

---

## Appendix: per-source data dictionary

### Meridian High — `data/meridian/course_requests.csv`
Flat CSV, one row per request. Columns: `student_id` (local numeric), `student_name`, `grade`, `course_code` (matches the catalog directly), `request_type` (`REQUIRED` or `ELECTIVE`).

### Apex Charter — `data/apex/requests.json` (+ `course_crosswalk.csv`)
Nested JSON: a list of students, each with a list of `requests`. Each request has a `courseNumber` (an Apex-local number, **not** a catalog code) and a `rank`. Students are identified by `stateStudentId`. Use `course_crosswalk.csv` to translate `courseNumber` → catalog code. The ranking convention — including how a backup/alternate is flagged — is documented in the file's notes.

### Cornerstone Prep — `data/cornerstone/primary_requests.csv` + `alternate_requests.csv`
Two CSVs with the same columns: `student_email`, `course_code` (matches the catalog), `terms`. Students are identified by email. The `terms` field indicates which part(s) of the year the request applies to (e.g. `FULL`, `S1`, or a quarter list like `Q1;Q2`). The split across two files is itself meaningful.

### Helix Academy — `data/helix/roster_export.json` (+ `course_map.json`)
Denormalized JSON: a `roster` of students, each carrying their own `courseRequests` and `alternates`. Courses are identified by `helixCourseUuid`; resolve them with `course_map.json`. Some requests are part of a sequence and carry a `chainId` / `chainSeq`. Students are identified by `studentId`.
