<!-- Thanks for contributing to rovo! Keep PRs small and focused (doc 26 §4). -->

## Summary

<!-- What does this change do, and why? One or two sentences. -->

Closes #

## Type

<!-- PR title must follow Conventional Commits, e.g. feat(dispatch): add offer cascade timeout.
     Allowed types: feat, fix, perf, refactor, docs, test, build, ci, chore, revert. -->

- [ ] feat
- [ ] fix
- [ ] perf / refactor
- [ ] docs
- [ ] test
- [ ] build / ci / chore

## Screenshots / recordings

<!-- For UI changes. Delete if not applicable. -->

## Checklist

- [ ] `make check` is green (lint + unit + integration + contract tests)
- [ ] Tests added or updated for the change
- [ ] DB migrations are **expand/contract** safe (additive; destructive changes ship a release later)
- [ ] API/OpenAPI changes reviewed by both backend and frontend owners
- [ ] User-facing strings are in **en and te** (i18next), no hardcoded copy
- [ ] Accessibility checked (WCAG 2.2 AA; keyboard + contrast) for UI changes
- [ ] Error handling, authorization (role + ownership) and logging in place
- [ ] Docs / ADR / runbook updated if behaviour or ops changed
- [ ] No secrets committed; `.env` stays local, `.env.example` documents new vars
- [ ] Commits signed off (`Signed-off-by:`, DCO)

<!-- By submitting this PR you certify the Developer Certificate of Origin (DCO)
     and license your contribution under Apache-2.0. See CONTRIBUTING.md. -->
