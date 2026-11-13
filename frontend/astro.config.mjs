// @ts-check
import { defineConfig } from 'astro/config';
import wails from "@wailsio/runtime/plugins/vite";
import tailwindcss from "@tailwindcss/vite";
import react from '@astrojs/react';

// https://astro.build/config
export default defineConfig({
  integrations: [
    react()
  ],
  vite: {
    plugins: [
      tailwindcss(),
      wails("./bindings"),
    ],
  },
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
  },
});
