# Spec Authoring Policy

`docs/spec/` contains short-lived planning and implementation specs.

Specs are execution documents, not permanent documentation. They guide focused work, keep scope
explicit, and define validation and cleanup before work is done.

Permanent framework behavior belongs in durable docs:

- `docs/user/`
- `docs/arch/`
- `README.md` when relevant
- package documentation
- examples
- tests

Do not treat `docs/spec/` as a long-term design archive.

## When A Spec Is Required

Create a spec before implementation when a change affects architecture, public behavior, or more
than one subsystem.

This includes:

- routing or filesystem conventions
- CLI behavior
- generated project structure
- templ integration
- HTMX behavior
- form handling
- development server behavior
- code generation
- public APIs
- breaking changes
- examples or durable documentation

Small local fixes may skip a spec when they are isolated, obvious, low risk, and covered by existing
tests.

When unsure about architecture or public behavior, write a short spec.

## Lifecycle

Specs that drive implementation must declare:

```md
status: draft | active | implemented | abandoned
created: YYYY-MM-DD
updated: YYYY-MM-DD
```

Optional metadata such as `owner:` is allowed when it helps coordination.

Only `active` specs should drive implementation.

Status meanings:

- `draft`: still shaping the work; not approved for implementation.
- `active`: approved execution document for implementation.
- `implemented`: work is complete, validated, and folded into durable docs where needed.
- `abandoned`: no longer relevant.

Implemented specs should be deleted during cleanup once they no longer provide useful context.

## File Naming

Most specs should be standalone files named by creation date:

```text
YYYY-MM-DD-short-slug.md
```

Use standalone specs when the work is one focused implementation slice and does
not need a multi-track roadmap.

Example:

```text
2026-05-12-asset-fingerprint-manifest.md
```

Use grouped specs only when the work explicitly needs an umbrella plus multiple
child tracks. Grouped specs use this shape:

```text
<group><sequence>-<short-slug>.md
```

Where:

- `<group>` is a single lowercase ASCII letter such as `a`, `b`, or `c`.
- `<sequence>` is a four-digit number.
- `<short-slug>` is lowercase ASCII words separated by hyphens.

The `0000` sequence is reserved for the umbrella spec in that group.

Child specs under the same umbrella must use the same group letter and the next
available sequence numbers.

Example:

```text
a0000-dev-tools-umbrella.md
a0001-goldr-new.md
a0002-goldr-dev.md
a0003-goldr-build.md

b0000-large-project-support-umbrella.md
b0001-action-routes.md
b0002-route-url-helpers.md
b0003-error-hooks.md
```

Do not use one global numeric sequence for unrelated initiatives.

Do not mix child specs from different umbrellas under the same group letter.

Do not reuse a group letter for a new initiative while specs from the previous
initiative remain in `docs/spec/`.

`README.md` is the only non-spec file in this directory and does not follow a
spec filename pattern.

## Required Structure

Use this structure for active implementation specs unless there is a strong reason not to:

```md
# Spec: <short title>

status: draft | active | implemented | abandoned
created: YYYY-MM-DD
updated: YYYY-MM-DD

## 1. Goal

## 2. Non-Goals

## 3. Background

## 4. Desired Behavior

## 5. Locked Decisions And Invariants

## 6. Rules And Failure Modes

## 7. Existing Patterns To Reuse

## 8. Agent Containment

## 9. Proposed Design

## 10. Implementation Plan

## 11. Acceptance Criteria

## 12. Validation Commands

## 13. Documentation Updates

## 14. Cleanup / Legacy Removal

## 15. Open Questions
```

Keep specs as small as possible while still executable. Split large specs instead of creating one
document that invites scope drift.

## Handoff-Ready Specs

Active standalone specs and active child specs must be handoff-ready.

Handoff-ready means another engineer or agent can implement the spec without reading prior chat,
guessing missing decisions, or inventing scope.

An active implementation spec must include:

- the exact goal and observable outcome
- explicit non-goals
- the supported behavior and any rejected behavior
- concrete interfaces, commands, paths, data shapes, or package boundaries affected by the work
- locked decisions and invariants that implementation must preserve
- rules and failure modes that matter for the slice
- existing patterns, helpers, packages, examples, tests, or docs that must be reused
- containment rules for allowed files, forbidden files, API/dependency constraints, refactor
  constraints, and stop conditions
- phased implementation checklist items
- targeted tests or an explicit reason tests are not needed
- validation commands
- durable docs and examples impact
- cleanup and legacy removal requirements
- open questions that are answered, deferred, or converted into follow-up specs before
  implementation

Do not mark a spec `active` if a reader still needs to decide what to build.

If implementation exposes a missing decision, stop and update the spec before continuing.

For very small active specs, sections may be brief, but do not omit locked
decisions, failure modes, reuse expectations, or containment when the work
affects routing, code generation, CLI behavior, development tooling, public
APIs, examples, or multiple packages.

## Umbrella Specs

Umbrella specs may be lighter than child implementation specs.

They may use compact tracker checklists and do not need to duplicate the full implementation
structure for every future track.

Umbrella specs must still make progress trackable:

- each track should have status
- each track should name its child spec when it exists
- each track should record docs impact
- each track should record examples impact
- each track should record validation state or next validation action

The next planned or active implementation track must have a child spec.

Future tracks may remain placeholders until they become the next slice of work.

Do not implement directly from an umbrella spec unless the change is only umbrella maintenance.

## Goals And Non-Goals

Goals must describe observable outcomes:

- what user-visible behavior changes
- what developer workflow changes
- what files, packages, or commands are affected
- what should not change

Non-goals are mandatory for active specs.

If implementation needs to cross a non-goal, stop and update the spec before continuing.

Do not silently drift.

## Locked Decisions, Reuse, And Containment

Locked decisions and invariants name behavior, boundaries, and architectural
constraints that must not change during implementation. They should be concrete
enough for a reviewer to tell whether the implementation preserved them.

Rules and failure modes should cover behavior that can be reached through
current supported writers, parsers, APIs, commands, filesystem shapes, or user
input. Do not require code or tests for unreachable defensive states. If a
spec keeps defensive handling for a state, it must name the current entry point
that can produce that state and explain why the handling belongs in this
slice.

Existing patterns to reuse should name the local command, package, helper,
test pattern, example, or documentation style that should guide the work.
Specs should prefer existing Goldr patterns over parallel new systems.

Agent containment should state:

- allowed files or directories
- forbidden files or directories
- public API constraints
- dependency constraints
- refactor constraints
- stop conditions

If a phase, validation command, documentation update, cleanup item, or
acceptance criterion names a source path, durable doc, generated artifact,
config, or script, that path should also appear in the implementation
touchpoints or containment rules.

## Architecture Quality Gates

Architecture or package-boundary specs must name the durable owner of the core
behavior and the contracts it owns.

Do not introduce packages, facades, wrappers, shims, mirrored types, aliases,
or forwarding helpers mainly to hide dependency direction, avoid touching the
real owner, or preserve a weak old structure. A new package must name the
durable responsibility it owns, who should depend on it, who must not depend on
it, and why that responsibility does not belong in an existing package.

If a temporary bridge or compatibility path is intentionally retained, the spec
must name the owner, reason, guardrail, and removal condition before
implementation proceeds.

Prefer explicit v0 contract cleanup over legacy compatibility when no concrete
current compatibility requirement is recorded in the spec.

If clean ownership is blocked by import cycles or unclear responsibility, stop
and update the spec with the available owner options before continuing.

## Anti-Pressure Rules

A checklist is not permission to force a pass. If validation evidence fails,
mark the spec blocked or update the spec with the next concrete design step
before changing downstream behavior.

Do not make a test, example, or review comment pass by adding route-specific
strings, path-substring checks, fixture-specific branches, command-specific
special cases, or one-off generated output tweaks unless those strings or
branches are already product contracts or are introduced as product contracts
in the spec before code changes.

Tests, examples, and review comments are validation inputs, not implementation
selectors. If implementation needs a string, path, or branch condition, the
spec must explain the Goldr contract that owns it.

## Implementation Plan

Non-trivial specs must be phased.

Checklist items must be actionable and verifiable.

An actionable checklist item names a concrete code, test, docs, validation, or cleanup result that a
reviewer can confirm.

Checklist items should not describe intentions, hopes, or broad quality goals.

When phases are helpful, keep phases as headings and put checkbox items under
each phase. Each phase should include the tests and documentation updates tied
to the behavior changed by that phase when practical. Use the final phase for
full verification and cleanup, not as the only place where tests or docs
appear.

Each source phase that changes behavior should include representative
happy-path tests and reachable edge-case tests, or explain why tests are not
needed. Name the important edge categories for the phase, such as validation
failures, missing files, stale generated output, unsupported route shapes, bad
CLI flags, empty results, or malformed user input.

Good:

```md
- [ ] `users/by_id/page.go` maps to `/users/{id}`.
- [ ] `orgs/by_org_id/users/by_user_id/page.go` maps to `/orgs/{org_id}/users/{user_id}`.
- [ ] Directories beginning with `_` are rejected by route scanning.
- [ ] `docs/user/routes.md` documents `by_<param>/` dynamic route directories.
- [ ] `scripts/check-all.sh` passes after scanner tests are added.
```

Bad:

```md
- [ ] Improve routing.
- [ ] Make developer experience better.
- [ ] Add tests.
- [ ] Update docs.
```

Update checklist items only when the code, tests, docs, or validation evidence exists.
Once a checklist item is verifiably complete, tick it before continuing to the
next phase or reporting the implementation as complete. Do not leave completed,
verifiable work unchecked as a progress note for later cleanup.

## Drift Control

You must not expand scope silently.

During implementation:

- do not add features not listed in the spec
- do not introduce new architectural concepts without updating the spec
- do not rewrite unrelated code
- do not rename public APIs unless the spec requires it
- do not create a second way to do something
- do not leave obsolete behavior half-supported unless compatibility is explicit

If new information changes scope, architecture, public behavior, validation, or non-goals, update
the spec before continuing.

## Pre-v0 Breaking Changes

goldr is pre-v0.

Specs may intentionally introduce breaking changes when they improve architecture, simplicity,
Go-native behavior, inspectability, public API clarity, or long-term maintainability.

Before v0, do not preserve compatibility with weak early conventions unless a spec explicitly
requires it.

If a spec introduces a breaking change, it must state:

- what breaks
- why the new design is better
- what obsolete code, docs, examples, or generated output must be removed
- what validation proves the new behavior

Pre-v0 freedom is not permission for churn. It is permission to improve the foundation.

## Project Rules Still Apply

All specs must follow `AGENTS.md` if available.

Do not duplicate root architectural rules here. In particular, filesystem conventions must stay
Go-native, HTMX must remain visible, runtime magic is disallowed, public APIs should be added
cautiously, and examples are product surface.

Specs must use plain ASCII text unless there is a concrete product reason not to.

Avoid em dashes, smart quotes, decorative bullets, non-ASCII punctuation, and invisible Unicode
whitespace.

## Go, Dependencies, And Public API

Specs that add dependencies, public APIs, exported packages, generated APIs, or toolchain
requirements must call that out explicitly.

They must explain:

- why the Go standard library is not enough
- why a small local implementation is not enough
- why the new API surface is necessary now
- what long-term maintenance burden is being accepted

Use the latest stable Go release during v0 development.

Prefer recent standard library APIs available in the project target Go version over custom helpers.

Keep dependencies and public API surface minimal.

## Validation, Docs, And Cleanup

Every active spec must list validation commands. If a command is not available yet, say so
explicitly.

Do not claim validation passed unless it was actually run.

Tests should be targeted and minimal.

Add or update tests when they protect framework behavior, public APIs, regressions, or architectural
invariants.

Prefer the smallest test that would fail for the behavior or bug the change is meant to cover.

Avoid broad matrices, large fixture trees, brittle incidental-internal assertions, and large golden
files unless they protect a real framework contract.

Each active spec should state whether it affects:

- `README.md`
- `docs/user/`
- `docs/arch/`
- examples
- generated starter app
- CLI help
- package docs

If behavior changes, durable docs must change in the same implementation slice.

If a spec changes supported user behavior, update `docs/user/` in the same slice.

If a spec changes framework architecture, generated-code behavior, internal invariants, or
maintainer expectations, update `docs/arch/` in the same slice.

If framework usage changes, update the example app in the same slice or explain why no example
update is needed.

A child spec is not complete if it leaves durable docs or examples stale.

CLI-visible changes should include CLI dogfood in the phase that implements
the CLI surface. Browser-visible example changes should include browser
validation when the behavior cannot be proven by unit tests alone.

### Docs-Only Validation Exception

Use this exception only when every touched file is a Markdown file directly
under `docs/spec/`.

For those changes:

- run the handoff-quality pass from this README
- do not require Go tests, linters, generated artifact checks, browser checks,
  or `scripts/check-all.sh`
- optionally run `git diff --check` when useful

This exception does not apply when a change touches source files, tests,
generated files, durable docs outside `docs/spec/`, examples, configs, or any
non-Markdown file. Specs that plan future source work must still include the
correct source validation commands for that future implementation spec.

Cleanup is part of the work. The cleanup section should identify obsolete code, docs, examples,
compatibility paths, generated files, and tests.

During v0, cleanup should be aggressive.

Durable docs should describe the current intended design, not the history of old decisions.

Remove stale docs, examples, generated output, and code instead of preserving deprecated behavior or
documenting long migration history, unless compatibility is explicitly required by the spec.

No legacy leftovers unless explicitly justified.

## Handoff Quality Pass

After drafting or editing an active spec, run a separate review pass focused
only on handoff quality before marking the spec active or reporting the spec
edit complete.

The pass must verify:

1. The spec is self-contained and does not rely on chat context.
2. Rules are deterministic and unambiguous.
3. Locked decisions, invariants, and non-goals are clear.
4. Existing patterns to reuse are named where applicable.
5. Containment rules cover every path named by phase items, validation
   commands, cleanup items, and planned docs updates.
6. Acceptance criteria are concrete and testable.
7. Validation commands are explicit, or the docs-only validation exception is
   explicitly applicable.
8. Rollout, rollback, cleanup, or legacy-removal expectations are clear enough
   for the slice.

Avoid vague placeholder words unless the spec provides an explicit default.

## Definition Of Done

A spec is done only when:

- all required checklist items are complete
- tests were added or updated, or the spec explains why not
- validation commands were run, or the spec explains why not
- durable docs were updated when behavior changed
- examples were updated when usage changed
- obsolete behavior and docs were removed
- no parallel obsolete behavior remains unless explicitly documented
- handoff quality was reviewed for any active spec changes
- the spec status is changed to `implemented`

## Instructions

When implementing an active spec, you must:

1. Read the entire spec first.
2. Confirm the handoff-quality pass is satisfied.
3. Identify the current phase.
4. Stay inside the containment rules.
5. Reuse the existing patterns named by the spec.
6. Make the smallest coherent change.
7. Avoid unrelated refactors.
8. Keep filesystem conventions Go-native.
9. Tick checklist items when they are actually complete and verifiable.
10. Run listed validation commands when possible.
11. Report commands that were not run.
12. Update docs and examples when required.
13. Stop and ask for spec changes if implementation would violate non-goals,
    locked decisions, invariants, or containment.

Optimize for correctness, coherence, and maintainability.
