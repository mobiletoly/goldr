// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

const { setTimeout: sleep } = require('node:timers/promises');

const versionPattern = /^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-(?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*))*)?$/;

async function tagCommit(github, repository, tag) {
  let object;
  try {
    const response = await github.rest.git.getRef({ ...repository, ref: `tags/${tag}` });
    object = response.data.object;
  } catch (error) {
    if (error.status === 404) return null;
    throw error;
  }
  while (object.type === 'tag') {
    const response = await github.rest.git.getTag({ ...repository, tag_sha: object.sha });
    object = response.data.object;
  }
  if (object.type !== 'commit') throw new Error(`Tag ${tag} does not point to a commit`);
  return object.sha;
}

async function waitForCI(github, repository, sha, core, pause) {
  for (let attempt = 0; attempt < 60; attempt++) {
    const response = await github.rest.actions.listWorkflowRuns({
      ...repository, workflow_id: 'ci.yml', event: 'push', head_sha: sha, per_page: 100,
    });
    const run = response.data.workflow_runs
      .filter(run => run.head_sha === sha && run.event === 'push')
      .sort((a, b) => b.id - a.id)[0];
    if (run?.status === 'completed') {
      if (run.conclusion !== 'success') {
        throw new Error(`CI for ${sha} finished with ${run.conclusion}: ${run.html_url}`);
      }
      core.info(`CI succeeded for ${sha}: ${run.html_url}`);
      return;
    }
    core.info(`Waiting for CI for ${sha} (${attempt + 1}/60)`);
    if (attempt < 59) await pause(30_000);
  }
  throw new Error(`CI for ${sha} did not succeed within 30 minutes; rerun after CI succeeds`);
}

module.exports = async function publishCLI({ github, context, core, pause = sleep }) {
  const release = context.payload.release;
  const version = release.tag_name;
  if (!version.startsWith('v')) {
    core.info(`Skipping non-root release ${version}`);
    return;
  }
  if (!versionPattern.test(version)) throw new Error(`Invalid root module version: ${version}`);

  const repository = context.repo;
  const sha = context.sha;
  if (await tagCommit(github, repository, version) !== sha) {
    throw new Error(`Root tag ${version} does not match release commit ${sha}`);
  }
  await waitForCI(github, repository, sha, core, pause);
  // Recheck after waiting so a moved root tag cannot publish a mismatched companion.
  if (await tagCommit(github, repository, version) !== sha) {
    throw new Error(`Root tag ${version} does not match release commit ${sha}`);
  }

  const cliTag = `cmd/goldr/${version}`;
  const existingSHA = await tagCommit(github, repository, cliTag);
  if (existingSHA !== null && existingSHA !== sha) {
    throw new Error(`Existing CLI tag ${cliTag} points to ${existingSHA}, not ${sha}; refusing to move it`);
  }
  if (existingSHA === null) {
    await github.rest.git.createRef({ ...repository, ref: `refs/tags/${cliTag}`, sha });
  }

  let existingRelease;
  try {
    existingRelease = (await github.rest.repos.getReleaseByTag({ ...repository, tag: cliTag })).data;
  } catch (error) {
    if (error.status !== 404) throw error;
  }
  if (existingRelease) {
    if (existingRelease.draft || existingRelease.prerelease !== release.prerelease) {
      throw new Error(`Existing CLI release ${cliTag} has conflicting draft/prerelease state`);
    }
    core.info(`CLI release already exists: ${existingRelease.html_url}`);
    return;
  }

  const created = await github.rest.repos.createRelease({
    ...repository,
    tag_name: cliTag,
    target_commitish: sha,
    name: `Goldr CLI ${version}`,
    body: `CLI companion to [Goldr ${version}](${release.html_url}).\n\n` +
      'Install or update the app-local tool:\n\n```sh\n' +
      `go get -tool github.com/${repository.owner}/${repository.repo}/cmd/goldr@${version}\n\`\`\`\n`,
    draft: false,
    prerelease: release.prerelease,
    make_latest: 'false',
  });
  core.info(`Published CLI release: ${created.data.html_url}`);
};
