# Documentation

goldr uses three documentation lanes.

## `docs/spec/`

Short-lived planning and implementation specifications.

Specs may describe proposed behavior, active decisions, implementation phases, and temporary constraints.

Active implementation specs must follow `docs/spec/README.md`.

Completed specs should be removed or folded into durable documentation once their behavior is implemented.

## `docs/user/`

Durable user and developer documentation for people building applications with goldr.

This directory describes supported behavior only.

It should be current-state and timeless during v0.

Do not preserve historical notes, deprecated alternatives, or migration commentary unless they are directly needed by current users.

## `docs/arch/`

Durable maintainer and architecture documentation.

This directory explains internal design, invariants, tradeoffs, generated-code behavior, and long-term architectural rules.

Code review patterns belong in `docs/arch/code-review.md`.

It should explain the current architecture, not preserve a history of abandoned designs.

Remove stale architecture notes aggressively when the design changes.

Completed specs must not remain the only source of truth for shipped behavior.
