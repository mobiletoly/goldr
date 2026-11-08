const assert = require('node:assert/strict');
const { test } = require('node:test');
const publishCLI = require('./release-cli.cjs');

function fixture(version = 'v0.3.1', prerelease = false) {
  const sha = 'a'.repeat(40);
  const cliTag = `cmd/goldr/${version}`;
  const refs = new Map([[`tags/${version}`, { type: 'commit', sha }]]);
  const objects = new Map();
  const releases = new Map();
  const calls = [];
  const writes = [];
  const success = { id: 20, head_sha: sha, event: 'push', status: 'completed', conclusion: 'success', html_url: 'https://example.com/ci/20' };
  const state = { runs: [[success]], sleeps: 0 };
  const missing = () => Object.assign(new Error('Not Found'), { status: 404 });
  const context = {
    repo: { owner: 'mobiletoly', repo: 'goldr' }, sha,
    payload: { release: { tag_name: version, target_commitish: 'moving-branch', prerelease, html_url: `https://github.com/mobiletoly/goldr/releases/tag/${version}` } },
  };
  const github = { rest: {
    git: {
      async getRef(args) {
        calls.push(['getRef', args]);
        if (!refs.has(args.ref)) throw missing();
        return { data: { object: refs.get(args.ref) } };
      },
      async getTag(args) {
        calls.push(['getTag', args]);
        assert.ok(objects.has(args.tag_sha));
        return { data: { object: objects.get(args.tag_sha) } };
      },
      async createRef(args) {
        calls.push(['createRef', args]);
        assert.equal(refs.has(args.ref.slice(5)), false);
        refs.set(args.ref.slice(5), { type: 'commit', sha: args.sha });
        writes.push(['tag', args]);
      },
    },
    actions: {
      async listWorkflowRuns(args) {
        calls.push(['listWorkflowRuns', args]);
        assert.equal(args.workflow_id, 'ci.yml');
        assert.equal(args.event, 'push');
        assert.equal(args.head_sha, sha);
        return { data: { workflow_runs: state.runs.length > 1 ? state.runs.shift() : state.runs[0] } };
      },
    },
    repos: {
      async getReleaseByTag(args) {
        calls.push(['getReleaseByTag', args]);
        if (!releases.has(args.tag)) throw missing();
        return { data: releases.get(args.tag) };
      },
      async createRelease(args) {
        calls.push(['createRelease', args]);
        assert.equal(releases.has(args.tag_name), false);
        const release = { ...args, html_url: `https://example.com/releases/${args.tag_name}` };
        releases.set(args.tag_name, release);
        writes.push(['release', args]);
        return { data: release };
      },
    },
  } };
  const input = {
    github, context, core: { info() {} },
    async pause(ms) { assert.equal(ms, 30_000); state.sleeps++; },
  };
  return { sha, cliTag, refs, objects, releases, calls, writes, success, state, input };
}

test('publishes a same-commit stable companion after CI and reuses it on rerun', async () => {
  const f = fixture();
  await publishCLI(f.input);
  assert.deepEqual(f.writes.map(([kind]) => kind), ['tag', 'release']);
  assert.equal(f.calls.findIndex(([name]) => name === 'listWorkflowRuns') < f.calls.findIndex(([name]) => name === 'createRef'), true);
  assert.equal(f.refs.get(`tags/${f.cliTag}`).sha, f.sha);
  const release = f.releases.get(f.cliTag);
  assert.equal(release.target_commitish, f.sha);
  assert.equal(release.name, 'Goldr CLI v0.3.1');
  assert.equal(release.prerelease, false);
  assert.equal(release.draft, false);
  assert.equal(release.make_latest, 'false');
  assert.match(release.body, /https:\/\/github.com\/mobiletoly\/goldr\/releases\/tag\/v0\.3\.1/);
  assert.match(release.body, /go get -tool github.com\/mobiletoly\/goldr\/cmd\/goldr@v0\.3\.1/);
  await publishCLI(f.input);
  assert.equal(f.writes.length, 2);
});

test('peels annotated root and CLI tags and mirrors prerelease status', async () => {
  const f = fixture('v0.3.1-rc.1', true);
  f.refs.set('tags/v0.3.1-rc.1', { type: 'tag', sha: 'root-tag' });
  f.objects.set('root-tag', { type: 'tag', sha: 'nested-tag' });
  f.objects.set('nested-tag', { type: 'commit', sha: f.sha });
  f.refs.set(`tags/${f.cliTag}`, { type: 'tag', sha: 'cli-tag' });
  f.objects.set('cli-tag', { type: 'commit', sha: f.sha });
  await publishCLI(f.input);
  assert.deepEqual(f.writes.map(([kind]) => kind), ['release']);
  assert.equal(f.releases.get(f.cliTag).prerelease, true);
  assert.equal(f.releases.get(f.cliTag).make_latest, 'false');
});

test('recovers from release creation failure without recreating the CLI tag', async () => {
  const f = fixture();
  const createRelease = f.input.github.rest.repos.createRelease;
  f.input.github.rest.repos.createRelease = async () => { throw new Error('service unavailable'); };
  await assert.rejects(publishCLI(f.input), /service unavailable/);
  assert.deepEqual(f.writes.map(([kind]) => kind), ['tag']);
  f.input.github.rest.repos.createRelease = createRelease;
  await publishCLI(f.input);
  assert.deepEqual(f.writes.map(([kind]) => kind), ['tag', 'release']);
});

test('refuses a conflicting CLI commit without moving its tag or publishing', async () => {
  const f = fixture();
  f.refs.set(`tags/${f.cliTag}`, { type: 'commit', sha: 'b'.repeat(40) });
  await assert.rejects(publishCLI(f.input), /refusing to move it/);
  assert.deepEqual(f.writes, []);
});

test('refuses existing draft or prerelease-state conflicts', async () => {
  for (const release of [{ draft: true, prerelease: false }, { draft: false, prerelease: true }]) {
    const f = fixture();
    f.refs.set(`tags/${f.cliTag}`, { type: 'commit', sha: f.sha });
    f.releases.set(f.cliTag, release);
    await assert.rejects(publishCLI(f.input), /conflicting draft\/prerelease state/);
    assert.deepEqual(f.writes, []);
  }
});

test('waits for the latest push CI run instead of accepting an earlier success', async () => {
  const f = fixture();
  f.state.runs = [[f.success, { ...f.success, id: 21, status: 'in_progress', conclusion: null }], [{ ...f.success, id: 21 }]];
  await publishCLI(f.input);
  assert.equal(f.state.sleeps, 1);
  assert.equal(f.writes.length, 2);
});

test('failed CI blocks all writes', async () => {
  const f = fixture();
  f.state.runs = [[{ ...f.success, conclusion: 'failure' }]];
  await assert.rejects(publishCLI(f.input), /CI .* finished with failure/);
  assert.deepEqual(f.writes, []);
});

test('missing or unrelated successful CI times out without publishing', async () => {
  for (const runs of [[], [{ head_sha: 'other-commit', event: 'push', status: 'completed', conclusion: 'success' }]]) {
    const f = fixture();
    f.state.runs = [runs];
    await assert.rejects(publishCLI(f.input), /did not succeed within 30 minutes/);
    assert.equal(f.state.sleeps, 59);
    assert.deepEqual(f.writes, []);
  }
});

test('a root tag moved during the CI wait blocks publication', async () => {
  const f = fixture();
  f.state.runs = [[{ ...f.success, status: 'queued', conclusion: null }], [f.success]];
  f.input.pause = async () => f.refs.set('tags/v0.3.1', { type: 'commit', sha: 'b'.repeat(40) });
  await assert.rejects(publishCLI(f.input), /Root tag .* does not match release commit/);
  assert.deepEqual(f.writes, []);
});

test('API errors do not count as missing tags or successful CI', async () => {
  for (const [group, method] of [['git', 'getRef'], ['actions', 'listWorkflowRuns'], ['repos', 'getReleaseByTag']]) {
    const f = fixture();
    f.refs.set(`tags/${f.cliTag}`, { type: 'commit', sha: f.sha });
    f.input.github.rest[group][method] = async () => { throw Object.assign(new Error('forbidden'), { status: 403 }); };
    await assert.rejects(publishCLI(f.input), /forbidden/);
    assert.deepEqual(f.writes, []);
  }
});

test('ignores companion releases and rejects noncanonical root versions before API calls', async () => {
  const companion = fixture('cmd/goldr/v0.3.1');
  await publishCLI(companion.input);
  assert.deepEqual(companion.calls, []);
  for (const version of ['v03.1.0', 'v0.3', 'v0.3.1-01', 'v0.3.1+build']) {
    const f = fixture(version);
    await assert.rejects(publishCLI(f.input), /Invalid root module version/);
    assert.deepEqual(f.calls, []);
  }
});
