import { expect, test } from "@playwright/test";

test("React island saves, unmounts for boosted navigation, and remounts", async ({ page }) => {
  await page.addInitScript(() => {
    const state = window as typeof window & { islandEvents?: string[] };
    state.islandEvents = [];
    document.addEventListener("client-island:mount", () => state.islandEvents?.push("mount"));
    document.addEventListener("client-island:unmount", () => state.islandEvents?.push("unmount"));
  });
  await page.goto("/");
  await expect(page.getByRole("region", { name: "Project editor" })).toBeVisible();
  await page.getByLabel("Project name").fill("   ");
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page.getByRole("alert")).toHaveText("Enter a project name.");
  await page.getByLabel("Project name").fill("Calm React editor");
  await page.getByLabel("Pin this project").check();
  await expect(page.getByTestId("dirty-state")).toHaveText("Unsaved changes");
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page.getByRole("status")).toHaveText("Saved");
  await expect(page.getByTestId("dirty-state")).toHaveText("No unsaved changes");
  await page.evaluate(() => ((window as typeof window & { shellMarker?: string }).shellMarker = "same-document"));
  await page.getByRole("link", { name: "Cancel" }).click();
  await expect(page.getByRole("heading", { name: "About the React island" })).toBeVisible();
  expect(await page.evaluate(() => (window as typeof window & { shellMarker?: string }).shellMarker)).toBe("same-document");
  expect(await page.evaluate(() => (window as typeof window & { islandEvents?: string[] }).islandEvents)).toEqual(["mount", "unmount"]);
  await page.goBack();
  await expect(page.getByLabel("Project name")).toHaveValue("Calm React editor");
  await expect(page.getByLabel("Pin this project")).toBeChecked();
  expect(await page.evaluate(() => (window as typeof window & { islandEvents?: string[] }).islandEvents)).toEqual(["mount", "unmount", "mount"]);
});
