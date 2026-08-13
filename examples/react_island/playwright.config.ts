import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "tests",
  use: {
    baseURL: process.env.GOLDR_ISLAND_BASE_URL,
    browserName: "chromium"
  }
});
