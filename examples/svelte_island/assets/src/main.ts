import { mount, unmount } from "svelte";
import ProjectEditor from "./ProjectEditor.svelte";
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

const components = new Map<HTMLElement, Record<string, unknown>>();

function mountIslands(container: ParentNode) {
  const candidates: HTMLElement[] = [];
  if (container instanceof HTMLElement && container.matches('[data-client-island="project-editor"]')) {
    candidates.push(container);
  }
  candidates.push(...container.querySelectorAll<HTMLElement>('[data-client-island="project-editor"]'));
  for (const element of candidates) {
    if (components.has(element)) continue;
    const component = mount(ProjectEditor, {
      target: element,
      props: {
        initial: { name: element.dataset.projectName ?? "", pinned: element.dataset.projectPinned === "true" },
        saveURL: element.dataset.saveUrl ?? "",
        cancelURL: element.dataset.cancelUrl ?? "",
        root: element
      }
    });
    components.set(element, component);
    document.dispatchEvent(new CustomEvent("client-island:mount", { detail: { framework: "svelte" } }));
  }
}

document.addEventListener("htmx:before:cleanup", (event) => {
  const target = event.target;
  if (!(target instanceof Element)) return;
  const island = target.closest<HTMLElement>('[data-client-island="project-editor"]');
  if (!island) return;
  const component = components.get(island);
  if (!component) return;
  components.delete(island);
  void unmount(component);
  document.dispatchEvent(new CustomEvent("client-island:unmount", { detail: { framework: "svelte" } }));
});

window.htmx.onLoad((element) => mountIslands(element));
mountIslands(document);
