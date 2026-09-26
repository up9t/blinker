import "@/main.css";
import { createApp } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import App from "@/App.vue";
import Main from "@/pages/Main.vue";
import Overlay from "@/pages/Overlay.vue";
import { loadSettings } from "@/settings";
import { hydrateSettings } from "@/settings";
import { applyTheme } from "@/lib/theme";

const routes = [
  { path: "/", component: Main },
  { path: "/overlay", component: Overlay },
];

export const router = createRouter({
  history: createWebHistory(),
  routes,
});

const app = createApp(App).use(router);

// Load settings from store before mounting
loadSettings().then((settings: any) => {
  hydrateSettings(settings);
  applyTheme(settings.theme);
  app.mount("#app");
});
