<script lang="ts">
  import { onDestroy, onMount } from "svelte";

  type Project = { name: string; pinned: boolean };
  type SaveResponse = { project?: Project; errors?: { name?: string }; error?: string };

  export let initial: Project;
  export let saveURL: string;
  export let cancelURL: string;
  export let root: HTMLElement;

  let name = initial.name;
  let pinned = initial.pinned;
  let saved = initial;
  let nameError = "";
  let status = "";
  let saving = false;
  let request: AbortController | null = null;
  $: dirty = name !== saved.name || pinned !== saved.pinned;

  onMount(() => window.htmx.process(root));
  onDestroy(() => request?.abort());

  async function save() {
    request?.abort();
    const controller = new AbortController();
    request = controller;
    saving = true;
    nameError = "";
    status = "";
    try {
      const csrfToken = document.querySelector<HTMLMetaElement>('meta[name="csrf-token"]')?.content ?? "";
      const response = await fetch(saveURL, {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken },
        body: JSON.stringify({ name, pinned }),
        signal: controller.signal
      });
      const payload = (await response.json()) as SaveResponse;
      if (response.status === 422) {
        nameError = payload.errors?.name ?? "Check the project name.";
        return;
      }
      if (!response.ok || !payload.project) {
        status = payload.error ?? "Save failed.";
        return;
      }
      name = payload.project.name;
      pinned = payload.project.pinned;
      saved = payload.project;
      status = "Saved";
    } catch (error) {
      if (!(error instanceof DOMException && error.name === "AbortError")) status = "Save failed.";
    } finally {
      if (request === controller) {
        request = null;
        saving = false;
      }
    }
  }
</script>

<section class="editor-card" aria-label="Project editor">
  <label for="project-name">Project name</label>
  <input id="project-name" bind:value={name} />
  {#if nameError}<p class="error" role="alert">{nameError}</p>{/if}
  <label class="checkbox-row"><input type="checkbox" bind:checked={pinned} /> Pin this project</label>
  <p class="preview" data-testid="preview">Preview: {name || "Untitled"}{pinned ? " (pinned)" : ""}</p>
  <p class="state" data-testid="dirty-state">{dirty ? "Unsaved changes" : "No unsaved changes"}</p>
  {#if status}<p role="status">{status}</p>{/if}
  <div class="actions">
    <button type="button" disabled={saving} onclick={save}>{saving ? "Saving..." : "Save"}</button>
    <a href={cancelURL}>Cancel</a>
  </div>
</section>
