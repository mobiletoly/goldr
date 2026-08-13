import React, { useEffect, useMemo, useRef, useState } from "react";
import { createRoot, type Root } from "react-dom/client";
import "./style.css";

type Htmx = {
  onLoad(callback: (element: Element) => void): void;
  process(element: Element): void;
};

declare global {
  interface Window {
    htmx: Htmx;
  }
}

type Project = {
  name: string;
  pinned: boolean;
};

type EditorProps = {
  initial: Project;
  saveURL: string;
  cancelURL: string;
  root: HTMLElement;
};

type SaveResponse = {
  project?: Project;
  errors?: { name?: string };
  error?: string;
};

const roots = new Map<HTMLElement, Root>();

function ProjectEditor({ initial, saveURL, cancelURL, root }: EditorProps) {
  const [name, setName] = useState(initial.name);
  const [pinned, setPinned] = useState(initial.pinned);
  const [saved, setSaved] = useState(initial);
  const [nameError, setNameError] = useState("");
  const [status, setStatus] = useState("");
  const [saving, setSaving] = useState(false);
  const request = useRef<AbortController | null>(null);
  const dirty = useMemo(() => name !== saved.name || pinned !== saved.pinned, [name, pinned, saved]);

  useEffect(() => {
    window.htmx.process(root);
    return () => request.current?.abort();
  }, [root]);

  async function save() {
    request.current?.abort();
    const controller = new AbortController();
    request.current = controller;
    setSaving(true);
    setNameError("");
    setStatus("");
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
        setNameError(payload.errors?.name ?? "Check the project name.");
        return;
      }
      if (!response.ok || !payload.project) {
        setStatus(payload.error ?? "Save failed.");
        return;
      }
      setName(payload.project.name);
      setPinned(payload.project.pinned);
      setSaved(payload.project);
      setStatus("Saved");
    } catch (error) {
      if (!(error instanceof DOMException && error.name === "AbortError")) {
        setStatus("Save failed.");
      }
    } finally {
      if (request.current === controller) {
        request.current = null;
        setSaving(false);
      }
    }
  }

  return (
    <section className="editor-card" aria-label="Project editor">
      <label htmlFor="project-name">Project name</label>
      <input id="project-name" value={name} onChange={(event) => setName(event.target.value)} />
      {nameError && <p className="error" role="alert">{nameError}</p>}
      <label className="checkbox-row">
        <input type="checkbox" checked={pinned} onChange={(event) => setPinned(event.target.checked)} />
        Pin this project
      </label>
      <p className="preview" data-testid="preview">Preview: {name || "Untitled"}{pinned ? " (pinned)" : ""}</p>
      <p className="state" data-testid="dirty-state">{dirty ? "Unsaved changes" : "No unsaved changes"}</p>
      {status && <p role="status">{status}</p>}
      <div className="actions">
        <button type="button" disabled={saving} onClick={save}>{saving ? "Saving..." : "Save"}</button>
        <a href={cancelURL}>Cancel</a>
      </div>
    </section>
  );
}

function mountIslands(container: ParentNode) {
  const candidates: HTMLElement[] = [];
  if (container instanceof HTMLElement && container.matches('[data-client-island="project-editor"]')) {
    candidates.push(container);
  }
  candidates.push(...container.querySelectorAll<HTMLElement>('[data-client-island="project-editor"]'));
  for (const element of candidates) {
    if (roots.has(element)) continue;
    const root = createRoot(element);
    roots.set(element, root);
    root.render(<ProjectEditor
      initial={{ name: element.dataset.projectName ?? "", pinned: element.dataset.projectPinned === "true" }}
      saveURL={element.dataset.saveUrl ?? ""}
      cancelURL={element.dataset.cancelUrl ?? ""}
      root={element}
    />);
    document.dispatchEvent(new CustomEvent("client-island:mount", { detail: { framework: "react" } }));
  }
}

document.addEventListener("htmx:before:cleanup", (event) => {
  const target = event.target;
  if (!(target instanceof Element)) return;
  const island = target.closest<HTMLElement>('[data-client-island="project-editor"]');
  if (!island) return;
  const root = roots.get(island);
  if (!root) return;
  roots.delete(island);
  root.unmount();
  document.dispatchEvent(new CustomEvent("client-island:unmount", { detail: { framework: "react" } }));
});

window.htmx.onLoad((element) => mountIslands(element));
mountIslands(document);
