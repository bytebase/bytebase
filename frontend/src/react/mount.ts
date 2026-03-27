import type { SubscriptionData } from "@/react/types";

export type { SubscriptionData };

export interface MountOptions {
  data: SubscriptionData;
  allowEdit: boolean;
  allowManageInstanceLicenses: boolean;
  onUploadLicense: (license: string) => Promise<boolean>;
}

async function loadDeps() {
  const [
    { createElement, StrictMode },
    { createRoot },
    { I18nextProvider },
    i18nModule,
    { SubscriptionPage },
  ] = await Promise.all([
    import("react"),
    import("react-dom/client"),
    import("react-i18next"),
    import("@/react/i18n"),
    import("@/react/pages/settings/SubscriptionPage"),
  ]);
  // Ensure i18n is fully initialized before rendering
  await i18nModule.i18nReady;
  return {
    createElement,
    StrictMode,
    createRoot,
    I18nextProvider,
    i18n: i18nModule.default,
    SubscriptionPage,
  };
}

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore -- React createElement types conflict with Vue JSX in vue-tsc; checked by tsconfig.react.json
function buildTree(
  deps: Awaited<ReturnType<typeof loadDeps>>,
  opts: MountOptions
) {
  const { createElement, StrictMode, I18nextProvider, i18n, SubscriptionPage } =
    deps;
  return createElement(
    StrictMode,
    null,
    createElement(
      I18nextProvider,
      { i18n },
      createElement(SubscriptionPage, opts)
    )
  );
}

let cached: Awaited<ReturnType<typeof loadDeps>> | null = null;

export async function mountSubscriptionPage(
  container: HTMLElement,
  opts: MountOptions
) {
  cached = await loadDeps();
  const root = cached.createRoot(container);
  root.render(buildTree(cached, opts));
  return root;
}

export async function updateSubscriptionPage(
  // biome-ignore lint/suspicious/noExplicitAny: React Root type from dynamic import
  root: any, // eslint-disable-line @typescript-eslint/no-explicit-any
  opts: MountOptions
) {
  if (!cached) cached = await loadDeps();
  root.render(buildTree(cached, opts));
}
