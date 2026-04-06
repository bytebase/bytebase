// Use import.meta.glob so vue-tsc does not follow the import into React .tsx files.
const sidebarLoader = import.meta.glob("./components/DashboardSidebar.tsx");

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

async function loadSidebar() {
  const loader = sidebarLoader["./components/DashboardSidebar.tsx"];
  if (!loader) throw new Error("DashboardSidebar not found");
  const mod = (await loader()) as Record<string, unknown>;
  return mod.DashboardSidebar as ReactDeps;
}

export async function mountSidebar(container: HTMLElement, locale: string) {
  const [deps, DashboardSidebar] = await Promise.all([
    loadCoreDeps(),
    loadSidebar(),
  ]);
  if (deps.i18n.language !== locale) {
    await deps.i18n.changeLanguage(locale);
  }
  const tree = deps.createElement(
    deps.StrictMode,
    null,
    deps.createElement(
      deps.I18nextProvider,
      { i18n: deps.i18n },
      deps.createElement(DashboardSidebar)
    )
  );
  const root = deps.createRoot(container);
  root.render(tree);
  return root;
}

export async function updateSidebarLocale(
  // biome-ignore lint/suspicious/noExplicitAny: React Root type from dynamic import
  root: any, // eslint-disable-line @typescript-eslint/no-explicit-any
  locale: string
) {
  const [deps, DashboardSidebar] = await Promise.all([
    loadCoreDeps(),
    loadSidebar(),
  ]);
  if (deps.i18n.language !== locale) {
    await deps.i18n.changeLanguage(locale);
  }
  const tree = deps.createElement(
    deps.StrictMode,
    null,
    deps.createElement(
      deps.I18nextProvider,
      { i18n: deps.i18n },
      deps.createElement(DashboardSidebar)
    )
  );
  root.render(tree);
}
