---
name: write-tests
description: Write Go unit tests for the code most recently implemented on the current feature branch of the timely take-home project. Use after implement-step, before opening the PR.
---

# Write unit tests for this step

Look at the diff on the current branch versus `master` (`git diff master...HEAD`) to see what was just implemented, then write tests for it.

## What to cover

- **The happy path** for the code just written.
- **The specific data quirk(s) relevant to this step**, if any apply:
  - Apex: a request with `rank: 500` must be classified as backup/alternate, not a numbered priority.
  - Cornerstone: emails differing only in case/whitespace must resolve to the same student; a course appearing only in `alternate_requests.csv` must be elective.
  - Helix: a `helixCourseUuid` missing from `course_map.json` must produce a clear "unresolvable course" result, not a panic or a silently dropped request. `chainId`/`chainSeq` ordering must be preserved.
  - Catalog/validation: a request pointing at a course code not in the catalog; a student with one half of a corequisite pair but not the other, checked in both declared directions since the catalog data is one-directional.
- **Malformed input**: empty fields, an unknown course code, a missing crosswalk/map entry — assert it's handled the way `implement-step` decided (structured issue, not a crash).

## Style

- Table-driven tests (`[]struct{ name string; ... }` + `t.Run`) for anything with more than ~2 cases.
- Prefer small inline fixtures over reading the full `data/` files, except for one or two tests that exercise a real fixture end-to-end to catch parsing surprises the inline cases miss.
- Test the package's exported behavior, not internals — don't add exported functions just to make them testable.
- No comments explaining what a test does — the test name and table case name should already say that.

## Before finishing

Run `go test ./... -run . -v` (or just `go test ./...` if verbose isn't needed) and make sure everything passes. If a test reveals a real bug in the implementation, fix the implementation — don't weaken the test to match broken behavior.
