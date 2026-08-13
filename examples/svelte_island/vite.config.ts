import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

export default defineConfig({
  plugins: [svelte()],
  build: {
    emptyOutDir: true,
    outDir: "assets/build",
    rollupOptions: {
      input: "assets/src/main.ts",
      output: {
        entryFileNames: "app.js",
        assetFileNames: "app.css"
      }
    }
  }
});
