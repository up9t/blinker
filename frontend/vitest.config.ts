import { defineConfig } from "vitest/config";

// Deliberately not reusing astro.config.mjs: unit tests run in node and never touch the browser build.
export default defineConfig({
  test: {
    environment: "node",
    include: ["src/**/*.test.ts"],
  },
});
