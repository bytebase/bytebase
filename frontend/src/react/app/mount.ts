import { watch } from "vue";
import { locale } from "@/plugins/i18n";
import { buildTree, loadCoreDeps, type ReactComponent } from "@/react/mount";

// import.meta.glob keeps vue-tsc from following into the .tsx file. Vite
// resolves the glob at build time and produces a lazy chunk.
const reactAppLoaders = import.meta.glob("./ReactApp.tsx");

async function loadReactApp(): Promise<ReactComponent> {
  const loader = reactAppLoaders["./ReactApp.tsx"] as
    | (() => Promise<Record<string, unknown>>)
    | undefined;
  if (!loader) {
    throw new Error("mountReactApp: ReactApp loader not registered");
  }
  const mod = await loader();
  return mod.ReactApp as ReactComponent;
}

export async function mountReactApp(selector: string) {
  const container = document.querySelector(selector);
  if (!container) {
    throw new Error(`mountReactApp: missing container ${selector}`);
  }

  const [deps, ReactApp] = await Promise.all([loadCoreDeps(), loadReactApp()]);

  // Sync initial locale before first paint.
  if (deps.i18n.language !== locale.value) {
    await deps.i18n.changeLanguage(locale.value);
  }

  const root = deps.createRoot(container);
  root.render(buildTree(deps, ReactApp));

  // Ongoing Vue→React locale sync for the long-lived React app.
  watch(locale, async (next) => {
    if (deps.i18n.language !== next) {
      await deps.i18n.changeLanguage(next);
    }
  });

  return root;
}
