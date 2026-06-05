// biome-ignore lint/suspicious/noExplicitAny: Runtime module shape varies by mount surface
export type ReactDeps = any;
// biome-ignore lint/suspicious/noExplicitAny: Props vary by mounted page
export type ReactComponent = (props: any) => any;

let cachedDeps: ReactDeps | null = null;

// Loads the core React + i18n runtime (lazily, cached) used to mount the app.
export async function loadCoreDeps() {
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

// Wraps a React component in the shared StrictMode + i18n provider tree.
export function buildTree(
  deps: ReactDeps,
  Component: ReactComponent,
  // biome-ignore lint/suspicious/noExplicitAny: Props type varies per page
  props?: any
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
