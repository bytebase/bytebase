import { buildTree, loadCoreDeps, type ReactComponent } from "@/react/mount";

// import.meta.glob keeps vue-tsc from following into the .tsx file (it stays in
// the React tsconfig project). Vite resolves the glob at build time.
const appRootLoaders = import.meta.glob("./AppRoot.tsx");

// Mounts the single React-Router application root (replaces the Vue
// `createApp(App)` mount + the separate `#react-app` overlay mount). The
// global overlays (Toaster, AgentWindow, SessionExpiredSurface, Watermark)
// live in RootLayout, so this one root hosts everything.
export async function mountReactRouterApp(selector: string) {
  const container = document.querySelector(selector);
  if (!container) {
    throw new Error(`mountReactRouterApp: missing container ${selector}`);
  }
  const loader = appRootLoaders["./AppRoot.tsx"] as
    | (() => Promise<Record<string, unknown>>)
    | undefined;
  if (!loader) {
    throw new Error("mountReactRouterApp: AppRoot loader not registered");
  }
  const [deps, mod] = await Promise.all([loadCoreDeps(), loader()]);
  const AppRoot = mod.AppRoot as ReactComponent;
  const root = deps.createRoot(container);
  root.render(buildTree(deps, AppRoot));
  return root;
}
