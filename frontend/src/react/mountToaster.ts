// Use import.meta.glob so vue-tsc does not follow the import into React .tsx files.
// Vite resolves the glob at build time and creates a lazy chunk for the matched module.
const toasterLoader = import.meta.glob("./components/ui/toaster.tsx");

// biome-ignore lint/suspicious/noExplicitAny: React types conflict with Vue JSX in vue-tsc
type ReactDeps = any; // eslint-disable-line @typescript-eslint/no-explicit-any

let cachedDeps: ReactDeps | null = null;

async function loadCoreDeps() {
  if (cachedDeps) return cachedDeps;
  const [
    { createElement, StrictMode },
    { createRoot },
    { I18nextProvider },
    i18nModule,
  ] = await Promise.all([
    import("react"),
    import("react-dom/client"),
    import("react-i18next"),
    import("@/react/i18n"),
  ]);
  await i18nModule.i18nReady;
  cachedDeps = {
    createElement,
    StrictMode,
    createRoot,
    I18nextProvider,
    i18n: i18nModule.default,
  };
  return cachedDeps;
}

async function loadToaster() {
  const loader = toasterLoader["./components/ui/toaster.tsx"];
  if (!loader) throw new Error("Toaster not found");
  const mod = (await loader()) as Record<string, unknown>;
  return mod.Toaster as ReactDeps;
}

/**
 * Mount the persistent <Toaster /> into the given container. Called once
 * at app bootstrap; the root lives until page unload. The bb.vue-notification
 * listener is already registered (main.ts side-effect-imports
 * @/react/lib/toast); this function only attaches the Provider so queued
 * toasts can render.
 */
export async function mountToaster(container: HTMLElement) {
  const [deps, Toaster] = await Promise.all([loadCoreDeps(), loadToaster()]);
  const tree = deps.createElement(
    deps.StrictMode,
    null,
    deps.createElement(
      deps.I18nextProvider,
      { i18n: deps.i18n },
      deps.createElement(Toaster)
    )
  );
  const root = deps.createRoot(container);
  root.render(tree);
  return root;
}
