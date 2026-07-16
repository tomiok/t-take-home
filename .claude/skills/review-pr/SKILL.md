---
name: review-pr
description: Review the current branch's diff before merging, against EXERCISE.md and the project's design decisions. Use as the last step before merging a PR in the timely take-home project.
---

# Review before merge

Diff the current branch against `master` (`git diff master...HEAD`, plus `gh pr view --json title,body,files` if a PR is already open). Review it as a colleague would in a real PR review — you are the gate before merge.

## Check, in order

1. **Correctness against EXERCISE.md.** Does this step actually do what the spec asks for this part of the pipeline? Re-read the relevant Appendix entry for the source this step touches, and the "What to build" section if this is unification/validation/CLI. Watch specifically for the known quirks (Apex rank-500 sentinel, Cornerstone email normalization + primary/alternate split, Helix dangling UUID + chain ordering, one-directional catalog corequisites) — a step that ignores one of these is a correctness bug, not a style nit.
2. **Test coverage.** Does `write-tests` actually exercise the quirk(s) relevant to this step, or just the happy path? Missing coverage of a known quirk is a blocking finding.
3. **Scope and simplification.** Flag speculative abstraction, unused exports, premature interfaces, or config that isn't needed yet. Flag comments that explain "what" instead of "why," or that reference this task/PR by name.
4. **Consistency with the established domain shape.** If this step is an adapter, does it produce the same `CourseRequest`-shaped output as prior adapters, or did it quietly diverge?
5. **Build health.** `go build ./...`, `go vet ./...`, `go test ./...` all pass.

## Output

Report findings as **blocking** (must fix before merge) or **non-blocking** (worth a follow-up, doesn't gate this merge). If there are no blocking findings, say so explicitly and approve. If there are, list them concretely (file:line, what's wrong, what to do instead) — don't just say "looks fine" if you skipped a check.

Do not merge the PR yourself as part of this review — report the verdict and let the calling step decide whether to fix-and-re-review or merge.
