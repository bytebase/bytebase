// Use import.meta.glob so vue-tsc does not follow the import into React .tsx files.
// Vite resolves the glob at build time and creates a lazy chunk for the matched module.
const settingsPageLoaders = import.meta.glob("./pages/settings/*.tsx");
const projectPageLoaders = import.meta.glob([
  "./pages/project/*.tsx",
  "./pages/project/plan-detail/ProjectPlanDetailPage.tsx",
]);
const pluginComponentLoaders = import.meta.glob(
  "./plugins/agent/components/AgentWindow.tsx"
);
const workspacePageLoaders = import.meta.glob("./pages/workspace/*.tsx");
const authPageLoaders = import.meta.glob("./pages/auth/*.tsx");
const authComponentLoaders = import.meta.glob([
  "./components/auth/SessionExpiredSurface.tsx",
  "./components/auth/InactiveRemindModal.tsx",
]);
const sqlEditorComponentLoaders = import.meta.glob(
  "./components/sql-editor/*.tsx"
);
const sharedComponentLoaders = import.meta.glob([
  "./components/*.tsx",
  "!./components/*.test.tsx",
]);
const pageLoaders = {
  ...settingsPageLoaders,
  ...projectPageLoaders,
  ...pluginComponentLoaders,
  ...workspacePageLoaders,
  ...authPageLoaders,
  ...authComponentLoaders,
  ...sqlEditorComponentLoaders,
  ...sharedComponentLoaders,
};

// biome-ignore lint/suspicious/noExplicitAny: React types conflict with Vue JSX in vue-tsc
type ReactDeps = any; // eslint-disable-line @typescript-eslint/no-explicit-any
// biome-ignore lint/suspicious/noExplicitAny: Component type checked by tsconfig.react.json
type ReactComponent = (props: any) => any; // eslint-disable-line @typescript-eslint/no-explicit-any

let cachedDeps: ReactDeps | null = null;
const cachedPages = new Map<string, ReactComponent>();

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

const pageDirs = [
  "./pages/settings",
  "./pages/project",
  "./pages/project/plan-detail",
  "./plugins/agent/components",
  "./pages/workspace",
  "./pages/auth",
  "./components/auth",
  "./components/sql-editor",
  "./components",
];

export function resolveReactPagePath(name: string): string | undefined {
  for (const dir of pageDirs) {
    const path = `${dir}/${name}.tsx`;
    if (path in pageLoaders) {
      return path;
    }
  }
  return undefined;
}

async function loadPage(name: string): Promise<ReactComponent> {
  const hit = cachedPages.get(name);
  if (hit) return hit;
  const path = resolveReactPagePath(name);
  const loader = path
    ? (pageLoaders[path] as
        | (() => Promise<Record<string, unknown>>)
        | undefined)
    : undefined;
  if (!loader) throw new Error(`Unknown React page: ${name}`);
  const mod = await loader();
  const Component = mod[name] as ReactComponent;
  cachedPages.set(name, Component);
  return Component;
}

/**
 * Mount a self-contained React page into the given container element.
 * @param page — file name without extension, must match the exported function name
 *               (e.g. "MCPPage", "SubscriptionPage")
 */
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore -- React createElement types conflict with Vue JSX in vue-tsc
function buildTree(
  deps: ReactDeps,
  Component: ReactComponent,
  // biome-ignore lint/suspicious/noExplicitAny: Props type varies per page
  props?: any // eslint-disable-line @typescript-eslint/no-explicit-any
) {
  return deps.createElement(
    deps.StrictMode,
    null,
    deps.createElement(
      deps.I18nextProvider,
      { i18n: deps.i18n },
      deps.createElement(Component, props)
    )
  );
}

/**
 * Mount a React page into the given container element.
 * @param page — file name without extension, must match the exported function name
 * @param props — optional props for pages that need Vue callbacks (e.g. SubscriptionPage)
 */
export async function mountReactPage(
  container: HTMLElement,
  page: string,
  // biome-ignore lint/suspicious/noExplicitAny: Props type varies per page
  props?: any // eslint-disable-line @typescript-eslint/no-explicit-any
) {
  const [deps, Component] = await Promise.all([loadCoreDeps(), loadPage(page)]);
  const root = deps.createRoot(container);
  root.render(buildTree(deps, Component, props));
  return root;
}

/**
 * Re-render an already-mounted React page with new props.
 */
export async function updateReactPage(
  // biome-ignore lint/suspicious/noExplicitAny: React Root type from dynamic import
  root: any, // eslint-disable-line @typescript-eslint/no-explicit-any
  page: string,
  // biome-ignore lint/suspicious/noExplicitAny: Props type varies per page
  props?: any // eslint-disable-line @typescript-eslint/no-explicit-any
) {
  const [deps, Component] = await Promise.all([loadCoreDeps(), loadPage(page)]);
  root.render(buildTree(deps, Component, props));
}
