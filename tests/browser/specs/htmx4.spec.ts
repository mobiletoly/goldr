import { expect, test, type Page } from "@playwright/test";

const chatBaseURL = requiredBaseURL("GOLDR_CHAT_BASE_URL");
const fullFeatureBaseURL = requiredBaseURL("GOLDR_FULL_FEATURE_BASE_URL");

const htmxCoreURL = "https://cdn.jsdelivr.net/npm/htmx.org@4.0.0";
const htmxSSEURL = "https://cdn.jsdelivr.net/npm/htmx.org@4.0.0/dist/ext/hx-sse.min.js";

function requiredBaseURL(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`${name} must be set by scripts/check-htmx-browser.sh`);
  }
  return value.replace(/\/$/, "");
}

function browserDiagnostics(page: Page) {
  const consoleErrors: string[] = [];
  const pageErrors: string[] = [];
  const requestFailures: string[] = [];
  const scriptResponses: Array<{ url: string; status: number }> = [];
  const expectedHTTPErrorStatuses: number[] = [];

  page.on("console", message => {
    if (message.type() === "error") {
      consoleErrors.push(message.text());
    }
  });
  page.on("pageerror", error => pageErrors.push(error.message));
  page.on("requestfailed", request => {
    requestFailures.push(`${request.method()} ${request.url()}: ${request.failure()?.errorText ?? "unknown"}`);
  });
  page.on("response", response => {
    if (response.request().resourceType() === "script" && response.url().startsWith("https://cdn.jsdelivr.net/npm/htmx.org@4.0.0")) {
      scriptResponses.push({ url: response.url(), status: response.status() });
    }
  });

  return {
    async assertHTMXLoaded(expectedURLs: string[]) {
      await expect.poll(() => page.locator('script[src*="htmx.org@4.0.0"]').count()).toBe(expectedURLs.length);
      await expect.poll(() => page.evaluate(() => Boolean((window as Window & { htmx?: unknown }).htmx))).toBe(true);
      for (const expectedURL of expectedURLs) {
        await expect.poll(() => scriptResponses.some(response => response.url === expectedURL && response.status === 200)).toBe(true);
      }
    },
    allowExpectedHTTPError(status: number) {
      expectedHTTPErrorStatuses.push(status);
    },
    assertClean() {
      const unexpectedConsoleErrors = [...consoleErrors];
      for (const status of expectedHTTPErrorStatuses) {
        const marker = `status of ${status}`;
        const index = unexpectedConsoleErrors.findIndex(message => message.includes(marker));
        if (index !== -1) {
          unexpectedConsoleErrors.splice(index, 1);
        }
      }
      expect(unexpectedConsoleErrors, "browser console errors").toEqual([]);
      expect(pageErrors, "browser page errors").toEqual([]);
      expect(requestFailures, "browser request failures").toEqual([]);
    },
  };
}

test("chat performs HTMX 4 validation, delayed submission, and named SSE swap", async ({ browser }) => {
  const page = await browser.newPage();
  const diagnostics = browserDiagnostics(page);

  await page.goto(`${chatBaseURL}/`);
  await expect(page.getByRole("heading", { name: "Goldr Chat" })).toBeVisible();
  await diagnostics.assertHTMXLoaded([htmxCoreURL, htmxSSEURL]);

  const sseRequest = page.waitForRequest(request => new URL(request.url()).pathname === "/chat/events");
  await page.getByLabel("Name").fill("Browser Ada");
  await page.getByRole("button", { name: "Enter chat" }).click();
  await sseRequest;
  expect(new URL(page.url()).pathname).toBe("/chat");

  await page.locator("#composer textarea").fill("   ");
  const validationResponse = page.waitForResponse(response => new URL(response.url()).pathname === "/chat/message" && response.status() === 422);
  await page.getByRole("button", { name: "Send" }).click();
  await validationResponse;
  diagnostics.allowExpectedHTTPError(422);
  await expect(page.locator("#composer .error")).toHaveText("Enter a message.");
  expect(new URL(page.url()).pathname).toBe("/chat");

  const body = `Browser HTMX 4 message ${Date.now()}`;
  await page.locator("#composer textarea").fill(body);
  const messageResponse = page.waitForResponse(response => new URL(response.url()).pathname === "/chat/message" && response.status() === 200);
  await page.getByRole("button", { name: "Send" }).click();
  await expect(page.locator("#send-progress")).toBeVisible();
  await messageResponse;
  await expect(page.locator("#messages")).toContainText(body, { timeout: 15_000 });

  diagnostics.assertClean();
  await page.close();
});

test("full-feature applies HTMX 4 status rules and inherited CSRF headers", async ({ browser }) => {
  const page = await browser.newPage();
  const diagnostics = browserDiagnostics(page);

  await page.goto(`${fullFeatureBaseURL}/users`);
  await expect(page.getByRole("heading", { name: "User Directory" })).toBeVisible();
  await diagnostics.assertHTMXLoaded([htmxCoreURL]);
  await expect(page.locator('meta[name="csrf-token"]')).toHaveAttribute("content", /.+/);
  expect(await page.locator("body").getAttribute("hx-headers:inherited")).toContain("X-CSRF-Token");

  const invalidRequest = page.waitForRequest(request => new URL(request.url()).pathname === "/users/create");
  const invalidResponse = page.waitForResponse(response => new URL(response.url()).pathname === "/users/create" && response.status() === 422);
  await page.locator("#contact-name").fill("");
  await page.getByRole("button", { name: "Add contact" }).click();
  expect((await invalidRequest).headers()["x-csrf-token"]).toBeTruthy();
  await invalidResponse;
  diagnostics.allowExpectedHTTPError(422);
  await expect(page.locator("#contact-name-error")).toHaveText("Name is required.");
  expect(new URL(page.url()).pathname).toBe("/users");

  const contactName = `Browser HTMX 4 user ${Date.now()}`;
  const createRequest = page.waitForRequest(request => new URL(request.url()).pathname === "/users/create");
  const createResponse = page.waitForResponse(response => new URL(response.url()).pathname === "/users/create" && response.status() === 200);
  await page.locator("#contact-name").fill(contactName);
  await page.getByRole("button", { name: "Add contact" }).click();
  expect((await createRequest).headers()["x-csrf-token"]).toBeTruthy();
  await createResponse;
  await expect(page.locator("#users-table")).toContainText(contactName);

  await page.getByRole("button", { name: "Active only", exact: true }).click();
  await expect(page.locator("#users-table")).toContainText(contactName);
  await expect(page.locator("#users-table")).not.toContainText("Katherine Johnson");

  diagnostics.assertClean();
  await page.close();
});
