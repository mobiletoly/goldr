import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "specs",
  fullyParallel: false,
  timeout: 30_000,
  expect: {
    timeout: 10_000,
  },
  use: {
    browserName: "chromium",
    trace: "retain-on-failure",
  },
});
