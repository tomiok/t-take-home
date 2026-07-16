---
name: implement-step
description: Implement one step of the course-request unification pipeline (timely take-home, EXERCISE.md) in Go. Use when picking up the next feature branch in the plan — domain model, one SIS adapter, the unification pipeline, validation, or the CLI.
---

# Implement one pipeline step

You are implementing exactly one step of the plan tracked in this session's tasks. Read `EXERCISE.md` first if you haven't already this conversation — it is the spec of record. Then read the relevant `data/<source>/` files directly (don't assume the Appendix data dictionary is complete; read a full sample of the actual file).

## Ground rules

- **Scope discipline.** Implement only the current step. Don't build adapters ahead of when they're needed, don't add an interface until at least two concrete implementations need it, don't add config/flags/abstractions the spec didn't ask for.
- **No comments unless the WHY is non-obvious** (a data quirk, a workaround, an invariant a reader would miss). Never comment what the code already says.
- **Handle bad data, don't crash on it.** A malformed row, a missing catalog code, a dangling reference — these are expected inputs from messy SIS exports, not programmer errors. Collect them as structured issues/warnings rather than panicking or silently dropping them. Following steps (validation, CLI) need to surface these, so don't swallow information a later step will need.
- **Keep the shared domain shape stable.** Once step 1 (domain-and-catalog) is merged, every adapter must produce the same `CourseRequest`-shaped output — don't invent a per-source result type.
- **Idiomatic Go**: table-driven where it helps, explicit error returns (wrapped with context via `fmt.Errorf("...: %w", err)`), small packages under a sensible layout (e.g. `internal/domain`, `internal/source/<name>`, `internal/pipeline`), no global state.

## Known data quirks to keep in mind across steps

- Catalog corequisites are declared **one-directional** (e.g. `SCI301` lists `MTH202` but not vice versa) — treat coreq pairs as symmetric wherever they're consumed.
- Apex: `rank` of `500` is a sentinel for "backup/alternate," not a real priority position.
- Cornerstone: student identity is email, and raw values have case/whitespace noise — normalize before using as a join key. The primary/alternate file split is itself the required/elective signal.
- Helix: `chainId`/`chainSeq` orders multi-term sequences; at least one `helixCourseUuid` in the sample roster is deliberately absent from `course_map.json` (dangling reference) — this must resolve to a "course not found" outcome, not a crash.

## When done

- `go build ./...` and `go vet ./...` must pass.
- Leave tests to the `write-tests` skill — don't write exhaustive tests yourself, a couple of sanity checks while developing is fine, but the dedicated test pass comes next.
- Summarize what you implemented and any assumption you made (these need to land in the README's Design section eventually — note them so they aren't lost).
