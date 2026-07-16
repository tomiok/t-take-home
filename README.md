# Course Request Project

## Quick Start

Requires Go 1.26+ (see `go.mod`). No external dependencies — standard library only.

From a clean checkout:

```
go run . run
```

This ingests all four SIS feeds under `data/` and writes the unified output to `output/unified_requests.json` (both paths configurable via `-data` and `-out`).

Two more subcommands demonstrate that the unified output actually serves the application's needs:

```
# A given student's requests, Required then Elective/backup, in sensible order.
# Student IDs are the source-qualified form written into the unified output,
# e.g. meridian:10042, apex:CA-2291043, cornerstone:marcus.thompson@cornerstone.edu, helix:HX-558.
go run . student meridian:10042

# Parsing issues, unknown-course requests, and missing-corequisite flags.
go run . validate
```

Run the test suite with `go test ./...`.

## Design

### How a course request is represented, and why

```go
type CourseRequest struct {
    StudentID  string
    CourseCode string
    Type       RequestType // Required | Elective
    Rank       int
    Source     string
    Terms      string
}
```

Every SIS adapter (`internal/source/{meridian,apex,cornerstone,helix}`) parses its own feed's format and produces this one shape — the seam between messy external data and what the rest of the app works with is the adapter boundary, not scattered throughout the codebase. A few choices worth calling out:

- **`CourseCode` always holds the catalog code when it can be resolved, and the raw source-local identifier when it can't.** An Apex course number with no `course_crosswalk.csv` entry, or a Helix UUID missing from `course_map.json`, still produces a `CourseRequest` — just with the broken reference visible on it — rather than silently dropping the request or crashing. Whether it's actually valid is a separate concern, handled by `internal/validation` against the catalog.
- **`Type` is just `Required`/`Elective`**, the one distinction every source expresses differently (Meridian's `REQUIRED`/`ELECTIVE` column, Apex's rank-500 sentinel, Cornerstone's file split, Helix's `isRequired` flag vs. its separate `alternates` list) but that the app needs uniformly.
- **`Rank` orders a student's requests within their `Type` group**, lower first. Its meaning is decided per-adapter (explicit priority for Apex, chain-aware ordering for Helix, plain file order for Meridian/Cornerstone) rather than baked into a single cross-source convention, since the sources don't agree on what "order" even means.
- **`Terms`** exists only because Cornerstone's `FULL`/`S1`/`Q1;Q2` scheduling data has nowhere else to live; it's stored verbatim rather than parsed into a structured type, since nothing downstream needs it split yet (see the open question below).

### How two records are decided to be the same student, or the same course

**Students:** `domain.StudentKey(source, rawID)` builds a source-qualified ID (e.g. `cornerstone:marcus.thompson@cornerstone.edu`). No cross-source student matching happens anywhere in the pipeline — the exercise states each SIS feed describes a disjoint set of students (one school per SIS), so there's no fuzzy-matching problem to solve, only a within-source identity problem: Cornerstone's raw emails have case/whitespace noise (`" Marcus.Thompson@cornerstone.edu "` vs. `marcus.thompson@cornerstone.edu`) that's normalized (trim + lowercase) before use as the join key, or the same student splits into two.

**Courses:** the catalog code is the single source of truth. Meridian and Cornerstone already use catalog codes directly; Apex and Helix use their own local identifiers (a course number, a UUID) and carry a crosswalk/map file that translates to a catalog code — each adapter does that translation itself, so everything downstream only ever deals in catalog codes (or, when translation fails, the raw identifier plus a flag that it didn't resolve).

### What happened when the data was wrong, missing, or didn't line up

The rule throughout: **surface it, don't hide it.** Every adapter returns a `source.Result{Students, Requests, Issues}` — row-level problems it recovered from become an `Issue` rather than a silent drop or a crash, and every one of these is exercised by a real quirk in the sample data:

| Source | Quirk | What happens |
|---|---|---|
| Meridian | Blank `request_type` | Defaults to `Elective` (promoting an ambiguous row to a graduation requirement is the riskier wrong guess) + an `Issue` |
| Meridian | Exact duplicate row | Deduped, first occurrence wins, + an `Issue` |
| Apex | `rank: 500` | Sentinel for backup/alternate, not a real priority — mapped to `Elective`, never carried through as a literal `Rank` of 500 |
| Apex | Course number absent from `course_crosswalk.csv` (at a *real priority* rank, not just a backup) | Raw identifier kept as `CourseCode` + an `Issue`; stays `Required`, not silently downgraded |
| Cornerstone | Case/whitespace-inconsistent email | Normalized before use as the join key |
| Cornerstone | No student name/grade in either file | Left zero-valued — see open question below |
| Helix | UUID absent from `course_map.json` | Same as Apex's unmapped course number: raw identifier kept, `Issue` logged, stays `Required` |
| Helix | `chainId`/`chainSeq` multi-term sequence | Reordered to stay contiguous and in sequence, anchored at the earliest position any chain member appears at — even if the file lists members out of order or scattered |

On top of adapter-level recovery, `internal/validation.Validate` checks the *unified* result against the catalog:
- a request whose `CourseCode` doesn't resolve to a real course becomes an `UnknownCourseFinding` (this is where Apex's `0999`, Cornerstone's `ART999`, and Helix's `zz9zz9` surface in the real data);
- a student who has one half of a corequisite pair but not the other becomes a `MissingCorequisiteFinding`. The catalog declares these pairs **one-directionally** (`SCI301` lists `MTH202` as a corequisite; `MTH202` doesn't list `SCI301` back) — the catalog loader was deliberately kept a faithful passthrough of that raw asymmetry, and `Validate` is the layer that treats a declared pair as symmetric in both directions.

### Assumptions made

- An ambiguous/blank `request_type` (Meridian) is safer defaulted to `Elective` than `Required`.
- An exact duplicate row (same student, course, and type) is a data-entry artifact, not a meaningful second request.
- A course reference that can't be resolved (crosswalk/map miss, or catalog miss entirely) should still produce a visible, flagged request rather than being dropped — losing a student's request silently seemed worse than surfacing a messy one.
- Cornerstone's missing name/grade data is acceptable to leave blank for this exercise rather than sourcing it from elsewhere.
- Each SIS feed represents a fully disjoint student population (no student appears in two feeds), per the exercise's own framing — so no cross-source student reconciliation was built.
- A catalog-declared corequisite is always meant to be bidirectional, and the one-directional declaration in the data is an omission rather than an intentional asymmetric relationship.
- `terms` doesn't need structured parsing yet — nothing in this exercise's scope (listing requests, validating, flagging corequisites) needs to reason about *which* term a request applies to.

### Questions for a product stakeholder before this went to production

- **Request-type ambiguity**: should a blank/unrecognized `request_type` actually block ingestion of that row and require an SIS-side correction, rather than silently defaulting to elective? Defaulting is convenient for a demo; it may be the wrong call operationally.
- **Unresolvable course references**: right now these're surfaced but still included as a request. Should they instead be excluded from what counselors/students see until resolved, or routed to a human review queue? What's the actual operational response when a crosswalk/map is out of date?
- **Duplicate emails / student identity changes**: Cornerstone identifies students by email; what happens if a student's email changes mid-year, or two different emails legitimately belong to the same person? Is there a more stable ID we should be using instead?
- **Cross-school transfers**: the assumption that each SIS feed's students are fully disjoint from every other feed's — is that guaranteed, or could a mid-year transfer make the same student appear (under different local IDs) in two feeds?
- **Corequisite symmetry**: can we confirm with whoever maintains the catalog that a one-directional corequisite declaration is always an omission, never an intentional one-way relationship (e.g. "B requires A alongside it, but A doesn't require B")?
- **`terms` semantics**: does the scheduling/conflict-detection side of the app need `terms` as structured data (e.g. to detect that two requests conflict only in `Q1`), or is the raw string enough indefinitely?
- **Output consumption shape**: is a flat `{students, requests, issues}` JSON file the right shape for whatever downstream system consumes this, or does it need to be nested per-student, paginated, or served some other way?

## AI Log

This project was built end-to-end with Claude Code (Anthropic), using a deliberately structured, review-gated workflow rather than one long freeform session:

1. **Planning.** Given the exercise brief and asked to produce a step-by-step plan before writing any code. Chose Go (the repo already had a `go.mod`/GoLand scaffold) over the exercise's suggested Python, after confirming the language choice explicitly. Settled on nine feature branches (domain model → one adapter per SIS → unification pipeline → validation → CLI → docs), each as its own branch, PR, and independent review before merging — the goal was a clean, inspectable history rather than one large commit.
2. **Skills.** Set up three project skills (`.claude/skills/{implement-step,write-tests,review-pr}`) to keep each step's development, testing, and review consistent: `implement-step` for scope-disciplined implementation of exactly one plan step; `write-tests` for table-driven tests targeting the specific data quirks in that step; `review-pr` as the checklist an independent reviewer follows before a PR can merge.
3. **Per-step workflow, repeated eight times** (once per adapter/pipeline/validation/CLI PR): implement the step directly against the real sample data (reading the actual CSV/JSON files rather than trusting the Appendix description alone), write tests pinning exact expected counts against the real fixtures, open a PR, then hand the PR to a fresh subagent with no prior context and instruct it to independently re-derive the expected output from the raw data by hand and review the diff against the exercise spec — not just check that tests pass. Every review either returned "no blocking findings" or caught something real:
   - Apex PR: a test asserted the wrong thing for an unresolved-course-at-a-real-priority-rank case (the exact "unresolved ≠ backup" trap), and rank derivation was inconsistent with the sibling adapter's convention.
   - Cornerstone PR: positional (not named) CSV column access, a minor consistency nit.
   - Helix PR: the chain-reordering algorithm's behavior under multiple interleaved chains was correct but undocumented and untested beyond the simple case.
   - Unification PR: a needlessly indirect closure-based loader pattern.
   - CLI PR: a real bug — `go run . -data foo student meridian:10042` (flags before the subcommand) silently fell back to the default `run` command and discarded the subcommand and its argument with no error at all. Fixed by rejecting unexpected leftover positional args instead of ignoring them.
4. Each review's findings were fixed (or explicitly deferred with a reason, when non-blocking and out of scope) before merging, and the PR was commented with a summary of what the review found and what changed, before squash-merging and deleting the branch.

If you want the raw prompts rather than this summary, the git history's PR descriptions and review comments on each PR in this repo capture the substance of what was asked and found at each step.
