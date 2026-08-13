import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    emptyOutDir: true,
    outDir: "assets/build",
    rollupOptions: {
      input: "assets/src/main.tsx",
      output: {
        entryFileNames: "app.js",
        assetFileNames: "app.css"
      }
    }
  }
});
