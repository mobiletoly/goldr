# Releases

Goldr publishes two Go modules from one repository:

| Module | Version tag |
| --- | --- |
| `github.com/mobiletoly/goldr` | `vX.Y.Z` |
| `github.com/mobiletoly/goldr/cmd/goldr` | `cmd/goldr/vX.Y.Z` |

The runtime and CLI use matching versions at the same commit. Go requires the
directory prefix for the nested CLI module's tags. See
[Go module source management](https://go.dev/doc/modules/managing-source).

## Publishing

1. Commit and push the release candidate, including the release workflow and
   its publisher script. Let the existing `CI` workflow qualify that commit.
2. Create the root version tag at that commit and publish its GitHub release.
   Stable versions use `vMAJOR.MINOR.PATCH`; prerelease versions may add a
   SemVer suffix such as `-rc.1`. Mark GitHub prereleases explicitly.
3. Observe the `Release CLI` workflow. It creates the matching CLI tag and
   GitHub release automatically after verifying CI.

Publishing the root GitHub release is the only manual publication step.
A tag push by itself does not publish the CLI. Draft release edits do not
start publication. The workflow listens to `release: published`, which covers
both stable releases and prereleases, and ignores CLI/non-root release tags.
There is no manual dispatch or backfill entry point. See
[GitHub release events](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#release).

Applications update the CLI independently of the runtime dependency:

```sh
go get -tool github.com/mobiletoly/goldr/cmd/goldr@vX.Y.Z
```

No separate binary build or uploaded asset is required for Go module users.
The CLI version command retains its existing build-time version behavior;
the published module version is available with:

```sh
go list -m github.com/mobiletoly/goldr/cmd/goldr
```

## Qualification And Commit Identity

The publisher checks out the release event's commit SHA, peels the root Git
tag through annotated tag objects, and requires that the tag points to that
same commit. It never selects the current branch tip or uses a release's
moving `target_commitish` value as the publication source.

It polls the latest push run of `.github/workflows/ci.yml` for that exact SHA
every 30 seconds for up to 30 minutes. The entire workflow must finish with
`success`, including its full-check and minimum-Go jobs. An older successful
run does not override a newer pending or failed push run for the same SHA.
Missing or pending CI times out; completed unsuccessful CI or API failures
stop publication before any writes. Root-tag identity is rechecked after the
wait. See the
[workflow-runs API](https://docs.github.com/en/rest/actions/workflow-runs#list-workflow-runs-for-a-workflow).

The new CLI tag is a lightweight ref at the verified commit. Existing CLI
tags, including annotated ones, must peel to that commit. A conflicting tag
fails publication and is never moved or replaced.

## Release State And Recovery

The CLI release mirrors the root release's prerelease flag, links its release
notes, and includes the app-local installation command. It sets
`make_latest: false`, preserving the root release as the repository's primary
release. See the
[release API](https://docs.github.com/en/rest/releases/releases#create-a-release).

Attempts for the same root tag are serialized without cancelling an active
attempt. Rerunning an existing Actions run reuses a matching CLI tag and
published CLI release. Existing draft or prerelease-state conflicts fail
rather than silently rewriting publication state. Published versions are
immutable; do not move tags to repair a release.

If CI or publication fails, inspect the failed workflow's logs. Fix or rerun
CI when appropriate, then rerun the existing failed `Release CLI` run. If
tag creation succeeded but release creation failed, the rerun verifies and
reuses that tag before retrying the missing release. Unrelated API errors
are not treated as absent tags or releases. Concurrent manual publication
can fail an attempt safely; rerunning rechecks the current state.

The workflow does not roll back or delete the root release on failure. It
does not synchronize later release-note edits or remove historical releases.
Disabling or reverting the workflow stops future automatic publication;
already-published versions remain in place.

## Workflow Ownership And Validation

`.github/workflows/release-cli.yml` owns the trigger, exact-SHA checkout,
per-tag concurrency, and 35-minute job timeout. The publisher in
`.github/scripts/release-cli.cjs` owns version validation, CI polling, and
idempotent tag/release creation. It uses `actions/github-script`'s GitHub API
client; no npm packages or application dependencies are added.

The publisher job uses the repository `GITHUB_TOKEN` with `contents: write`
and `actions: read`. Repository policies must permit those permissions. No
additional secret or personal token is required. Tag creation and release
creation happen in the same job: events generated by `GITHUB_TOKEN` do not
start another release workflow. See
[workflow token triggering rules](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow).

The CI check job runs the publisher tests with Node 24 before its existing
Go checks. Tests use an in-memory API client and an immediate timer to prove
commit matching, CI gating, prerelease/latest flags, tag immutability, and
safe recovery without making GitHub writes. For changes to release tooling,
also run locally:

```sh
node --test .github/scripts/release-cli.test.cjs
actionlint .github/workflows/ci.yml .github/workflows/release-cli.yml
scripts/check-all.sh
git diff --check
```

Local tests and workflow linting do not qualify a hosted publication. Confirm
the first real run in GitHub Actions after publishing a root release.
