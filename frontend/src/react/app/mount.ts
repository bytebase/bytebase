import { buildTree, loadCoreDeps, type ReactComponent } from "@/react/mount";

// Vite resolves the glob at build time and produces a lazy chunk.
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

  const root = deps.createRoot(container);
  root.render(buildTree(deps, ReactApp));

  return root;
}
