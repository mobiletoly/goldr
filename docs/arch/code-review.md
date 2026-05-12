# Code Review

Code review should protect goldr's correctness, coherence, simplicity, and
maintainability.

Use this file as the shared review pattern catalog for human and agent reviews.

This is not a replacement for `scripts/check-all.sh`.

Automated checks catch mechanical issues. Review should catch architectural and
behavioral drift.

## Review Priorities

Review findings should focus on:
- correctness bugs
- framework contract violations
- hidden complexity
- unnecessary abstractions
- unsupported parallel systems
- stale docs or examples
- test gaps for externally visible behavior
- test bloat that locks down implementation details without protecting behavior

Do not turn review into broad style commentary.

Prefer issues that are concrete, actionable, and tied to goldr's architecture.

## Pattern Format

Each review pattern should describe:
- what to look for
- bad shape
- good shape
- why it matters

Keep patterns short and current.

Remove patterns that no longer describe real review risk.

## Bug Fix Without Reproducer

Look for bug fixes or review-comment fixes that change production behavior
before proving the bug with a focused failing test.

Bad shape:
- production code is changed before a failing test exists
- a broad test is added after the fix but would not have failed before it
- a review bug comment is resolved by inspection only when a small reproducer
  is practical
- the fix claims an untestable exception without explaining why

Good shape:
- add the smallest test that reproduces the bug first
- run that test and confirm it fails for the intended reason
- fix the behavior and make the same test pass
- if a failing test is impractical, state why and use the closest available
  verification

Why it matters:

Regression tests turn bug fixes into durable framework knowledge. They also
protect review fixes from becoming plausible-looking changes that do not
actually reproduce the reported failure.

## Unsupported Convention Defensiveness

Look for code that special-cases conventions goldr does not support or
advertise.

Bad shape:
- dedicated rejection branches for syntax from other frameworks
- compatibility paths for patterns goldr never documented
- error messages that teach unsupported external conventions
- tests that make unsupported external conventions look like product surface

Good shape:
- validate goldr's documented allowed forms
- reject invalid input through general naming or contract rules
- keep error messages tied to goldr concepts
- test supported behavior and generic invalid cases

Why it matters:

Special-casing unsupported external conventions makes the framework appear to
have inherited or considered those conventions. It adds maintenance burden
without product value and weakens goldr's Go-native identity.

## Scattered Convention Vocabulary

Look for framework convention terms repeated as raw strings across scanner,
generator, CLI, docs, or tests.

Bad shape:
- repeated literals such as file names, route prefixes, suffixes, or render-unit
  names inside behavior code
- separate code paths spelling the same convention differently
- tests re-encoding conventions in ways that can drift from implementation
- a large registry abstraction introduced just to avoid a few literals

Good shape:
- small constants close to the implementation that owns the convention
- direct branch logic that still reads like the filesystem convention
- tests that assert behavior, not the private constant names
- no generic registry unless multiple real call sites need it

Why it matters:

Filesystem conventions are product surface. Scattered literals can drift
quietly, while over-abstracted registries make simple conventions harder to
read. Prefer a small, boring vocabulary source before adding a larger
abstraction.

## Parallel Systems

Look for changes that create a second way to do the same framework task.

Bad shape:
- filesystem routes plus runtime route registration
- fragments discovered by both naming convention and manual registry
- duplicate configuration paths for the same behavior
- old and new conventions supported indefinitely during v0

Good shape:
- one documented path for each framework concept
- cleanup of obsolete behavior during v0
- migration only when explicitly required by an active spec

Why it matters:

Parallel systems make goldr harder to explain, test, document, and maintain.
During v0, breaking cleanup is usually better than compatibility baggage.

## Speculative Abstraction

Look for abstractions added before the code has repeated real pressure.

Bad shape:
- generic interfaces with one implementation
- helpers that hide simple control flow
- factories, registries, or lifecycle hooks without current need
- exported types added for possible future use

Good shape:
- direct code until repeated use justifies extraction
- small unexported helpers when they remove real duplication
- minimal public API

Why it matters:

Every abstraction becomes framework surface area. goldr should stay easy to read
and easy to reason about.

## Hidden Magic

Look for behavior that cannot be understood from nearby code or documented
filesystem conventions.

Bad shape:
- reflection-driven behavior where direct code would work
- implicit registration
- annotation-driven behavior
- YAML-driven framework behavior
- automatic mutation or synchronization not visible to the caller

Good shape:
- explicit calls
- visible filesystem conventions
- behavior traceable through ordinary Go code

Why it matters:

Inspectability is a product feature. Hidden behavior makes both human and agent
maintenance worse.

## Test Bloat

Look for tests that are numerous but weakly connected to framework behavior.

Bad shape:
- tests that lock down private implementation steps
- broad fixture matrices without a clear contract
- duplicated tests that differ only in incidental setup
- tests added only to satisfy line coverage

Good shape:
- targeted tests for user-visible behavior and framework contracts
- small invalid-case coverage for important boundaries
- helper functions only when they make intent clearer

Why it matters:

Tests should protect behavior without making refactoring expensive. goldr wants
minimal, high-signal tests.

## Stale Documentation

Look for docs, specs, comments, or examples that preserve abandoned history.

Bad shape:
- docs that describe old naming conventions
- examples using deprecated patterns
- comments explaining previous designs
- implemented specs left as the only source of truth

Good shape:
- durable docs describe current behavior only
- examples demonstrate the recommended path
- obsolete specs are cleaned up or marked implemented

Why it matters:

During v0, stale docs are especially harmful because they teach unstable early
decisions as if they are product contracts.
