# Notification System Migration — Vue → React (Base UI Toast) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `NotificationContext.vue` + Naive UI's `NNotificationProvider` with a React renderer built on Base UI Toast. React becomes the sole renderer; Vue's existing `pushNotification()` API keeps working by emitting a window event that the React side catches.

**Architecture:** Module-level `toastManager = createToastManager()` lives outside any React tree so non-component code can call it. A long-lived React root mounted at `<div id="bb-toaster-root">` renders `<Toaster />`, which iterates the manager's toasts via `useToastManager()`. Vue's Pinia `notificationStore` dispatches `bb.vue-notification` window events; a module-eval-time listener forwards them to the manager.

**Tech Stack:** React, `@base-ui/react/toast`, `class-variance-authority`, Tailwind CSS v4, Vue 3 (Pinia store stays, only renderer dies), Vite `import.meta.glob`.

**Companion docs:**
- Spec: `docs/plans/2026-05-13-notification-react-migration-design.md`
- Playbook: `docs/plans/2026-04-08-react-migration-playbook.md`

---

## Branch setup

- [ ] **Step 1: Pull latest main and create branch**

```bash
git checkout main && git pull --ff-only
git checkout -b chore/frontend/notification-react-migration
```

- [ ] **Step 2: Baseline green**

```bash
pnpm --dir frontend type-check
pnpm --dir frontend test
```

Expected: type-check exit 0; vitest reports all tests pass.

---

## Task 1: Add shadcn-style wrappers around Base UI Toast parts

Static UI primitives. No runtime wiring yet.

**Files:**
- Create: `frontend/src/react/components/ui/toast.tsx`

- [ ] **Step 1: Write `toast.tsx`**

```tsx
import { Toast as BaseToast } from "@base-ui/react/toast";
import { cva, type VariantProps } from "class-variance-authority";
import { CheckCircle2, Info, AlertTriangle, XCircle, X } from "lucide-react";
import type { ComponentProps, ReactNode } from "react";
import { cn } from "@/react/lib/utils";

// Map BBNotificationStyle ("SUCCESS" | "INFO" | "WARN" | "CRITICAL") onto
// the visual variant of the toast container.
export type ToastVariant = "success" | "info" | "warning" | "error";

const toastRootVariants = cva(
  // Base: absolutely-positioned card. Base UI Toast.Root supplies its own
  // transform-based animation slot via CSS variables; we layer surface
  // styling on top.
  [
    "absolute right-0 bottom-0",
    "w-(--toast-width) max-w-[calc(100vw-2rem)]",
    "rounded-md border bg-popover text-popover-foreground shadow-md",
    "px-4 py-3 pr-10",
    // Base UI emits these CSS vars; we use them for the stack/expand transforms.
    "transform [transition:transform_250ms,opacity_250ms]",
    "[transform:translateY(calc(var(--toast-swipe-movement-y,0px)+var(--toast-index)*-12px))_scale(calc(1-var(--toast-index)*0.05))]",
    "[&[data-expanded]]:[transform:translateY(calc(var(--toast-offset-y,0px)*-1-var(--toast-index)*16px))]",
    "[&[data-starting-style]]:opacity-0",
    "[&[data-ending-style]]:opacity-0",
  ].join(" "),
  {
    variants: {
      variant: {
        success: "border-success/40",
        info: "border-info/40",
        warning: "border-warning/40",
        error: "border-error/40",
      },
    },
    defaultVariants: { variant: "info" },
  }
);

const iconVariants = cva("size-5 shrink-0 mt-0.5", {
  variants: {
    variant: {
      success: "text-success",
      info: "text-info",
      warning: "text-warning",
      error: "text-error",
    },
  },
  defaultVariants: { variant: "info" },
});

const iconMap: Record<ToastVariant, typeof CheckCircle2> = {
  success: CheckCircle2,
  info: Info,
  warning: AlertTriangle,
  error: XCircle,
};

type ToastRootProps = Omit<
  ComponentProps<typeof BaseToast.Root>,
  "className"
> &
  VariantProps<typeof toastRootVariants> & {
    className?: string;
    showIcon?: boolean;
    children?: ReactNode;
  };

function ToastRoot({
  variant = "info",
  showIcon = true,
  className,
  children,
  ...props
}: ToastRootProps) {
  const Icon = iconMap[variant ?? "info"];
  return (
    <BaseToast.Root
      {...props}
      className={cn(toastRootVariants({ variant }), className)}
    >
      <div className="flex items-start gap-x-3">
        {showIcon ? <Icon className={iconVariants({ variant })} /> : null}
        <div className="flex min-w-0 flex-1 flex-col gap-y-1">{children}</div>
      </div>
    </BaseToast.Root>
  );
}

function ToastTitle({
  className,
  ...props
}: ComponentProps<typeof BaseToast.Title>) {
  return (
    <BaseToast.Title
      {...props}
      className={cn("text-sm font-medium leading-5", className)}
    />
  );
}

function ToastDescription({
  className,
  ...props
}: ComponentProps<typeof BaseToast.Description>) {
  return (
    <BaseToast.Description
      {...props}
      className={cn(
        "text-sm leading-5 text-muted-foreground whitespace-pre-wrap",
        className
      )}
    />
  );
}

function ToastAction({
  className,
  ...props
}: ComponentProps<typeof BaseToast.Action>) {
  return (
    <BaseToast.Action
      {...props}
      className={cn(
        "mt-1 inline-flex w-fit text-sm font-medium text-accent hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        className
      )}
    />
  );
}

function ToastClose({
  className,
  "aria-label": ariaLabel,
  ...props
}: ComponentProps<typeof BaseToast.Close>) {
  return (
    <BaseToast.Close
      {...props}
      aria-label={ariaLabel ?? "Close"}
      className={cn(
        "absolute right-2 top-2 inline-flex size-7 items-center justify-center rounded-sm text-muted-foreground opacity-60 transition-opacity hover:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        className
      )}
    >
      <X className="size-4" />
    </BaseToast.Close>
  );
}

export { ToastRoot, ToastTitle, ToastDescription, ToastAction, ToastClose };
```

- [ ] **Step 2: Verify type-check + lint**

```bash
pnpm --dir frontend type-check
pnpm --dir frontend fix
```

Expected: both green; `fix` may reorganize imports.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/react/components/ui/toast.tsx
git commit -m "feat(react/ui): add shadcn-style wrappers around Base UI Toast parts

Static primitives only; no runtime wiring yet. cva variants on
ToastRoot map BBNotificationStyle (SUCCESS/INFO/WARN/CRITICAL) onto
visual variants (success/info/warning/error). ToastClose has a default
'Close' aria-label that callers can override via i18n.

Follows the dialog.tsx pattern: Base UI primitive + cn() + tokens; no
raw colors; portal/layering handled in toaster.tsx (next commit).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Add the toast manager + window listener

Module-level standalone manager. Importable from any code (React, Vue, framework-neutral). Listener registers at module-eval time.

**Files:**
- Create: `frontend/src/react/lib/toast.ts`
- Create: `frontend/src/react/lib/toast.test.ts`

- [ ] **Step 1: Write the failing unit test for the mapping function**

```ts
// frontend/src/react/lib/toast.test.ts
import { describe, expect, test, vi, beforeEach } from "vitest";

const addMock = vi.fn();
vi.mock("@base-ui/react/toast", () => ({
  createToastManager: () => ({ add: addMock, close: vi.fn() }),
}));

import { pushReactNotification, mapNotificationToToast } from "./toast";

describe("mapNotificationToToast", () => {
  beforeEach(() => {
    addMock.mockReset();
  });

  test("SUCCESS maps to type=success, priority=low, timeout=6000", () => {
    expect(
      mapNotificationToToast({
        module: "bytebase",
        style: "SUCCESS",
        title: "Saved",
      })
    ).toMatchObject({
      type: "success",
      priority: "low",
      timeout: 6000,
      title: "Saved",
    });
  });

  test("CRITICAL maps to type=error, priority=high, timeout=10000", () => {
    expect(
      mapNotificationToToast({
        module: "bytebase",
        style: "CRITICAL",
        title: "Boom",
      })
    ).toMatchObject({
      type: "error",
      priority: "high",
      timeout: 10000,
    });
  });

  test("WARN maps to type=warning, INFO maps to type=info", () => {
    expect(
      mapNotificationToToast({
        module: "bytebase",
        style: "WARN",
        title: "Heads up",
      })
    ).toMatchObject({ type: "warning" });
    expect(
      mapNotificationToToast({
        module: "bytebase",
        style: "INFO",
        title: "FYI",
      })
    ).toMatchObject({ type: "info" });
  });

  test("manualHide=true sets timeout=0 (manager treats 0 as manual)", () => {
    expect(
      mapNotificationToToast({
        module: "bytebase",
        style: "INFO",
        title: "T",
        manualHide: true,
      })
    ).toMatchObject({ timeout: 0 });
  });

  test("description string passes through; link/linkTitle become actionProps", () => {
    const mapped = mapNotificationToToast({
      module: "bytebase",
      style: "INFO",
      title: "T",
      description: "details",
      link: "https://example.com",
      linkTitle: "Open",
    });
    expect(mapped.description).toBe("details");
    expect(mapped.actionProps).toMatchObject({
      "aria-label": "Open",
      onClick: expect.any(Function),
    });
  });
});

describe("pushReactNotification", () => {
  beforeEach(() => addMock.mockReset());

  test("calls toastManager.add with mapped options", () => {
    pushReactNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: "Saved",
    });
    expect(addMock).toHaveBeenCalledTimes(1);
    expect(addMock.mock.calls[0][0]).toMatchObject({
      title: "Saved",
      type: "success",
      priority: "low",
      timeout: 6000,
    });
  });

  test("ignores notifications with module !== 'bytebase'", () => {
    pushReactNotification({
      // @ts-expect-error — test guards a runtime filter
      module: "other",
      style: "INFO",
      title: "ignored",
    });
    expect(addMock).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
pnpm --dir frontend test -- frontend/src/react/lib/toast.test.ts
```

Expected: FAIL with "Cannot find module './toast'" or similar.

- [ ] **Step 3: Write `frontend/src/react/lib/toast.ts`**

```ts
import { createToastManager } from "@base-ui/react/toast";
import type { ToastManagerAddOptions } from "@base-ui/react/toast";
import type { NotificationCreate } from "@/types/notification";

const NOTIFICATION_DURATION_MS = 6000;
const CRITICAL_NOTIFICATION_DURATION_MS = 10000;

const VUE_NOTIFICATION_EVENT = "bb.vue-notification";

/**
 * Module-level toast manager — created once, lives outside any React tree.
 * Callers anywhere (React components, Zustand slices, plain TS modules,
 * the Vue side via the window-event bridge below) can call .add() / .close().
 *
 * The <Toaster /> component subscribes via Base UI's useToastManager() hook
 * and renders each toast.
 */
export const toastManager = createToastManager();

type ToastOptions = ToastManagerAddOptions<Record<string, unknown>>;

/**
 * Convert the project's NotificationCreate shape into Base UI Toast options.
 * Pure function — exported for testing.
 */
export function mapNotificationToToast(item: NotificationCreate): ToastOptions {
  const type =
    item.style === "SUCCESS"
      ? "success"
      : item.style === "WARN"
        ? "warning"
        : item.style === "CRITICAL"
          ? "error"
          : "info";
  const priority: "low" | "high" = item.style === "CRITICAL" ? "high" : "low";
  const timeout = item.manualHide
    ? 0
    : item.style === "CRITICAL"
      ? CRITICAL_NOTIFICATION_DURATION_MS
      : NOTIFICATION_DURATION_MS;

  const actionProps =
    item.link && item.linkTitle
      ? {
          "aria-label": item.linkTitle,
          onClick: () => {
            window.open(item.link, "_blank", "noopener,noreferrer");
          },
          children: item.linkTitle,
        }
      : undefined;

  return {
    title: item.title,
    description:
      typeof item.description === "string" ? item.description : undefined,
    type,
    priority,
    timeout,
    actionProps,
  };
}

/**
 * Push a notification through the React toast renderer. Safe to call from
 * any context (component, store, plain TS module). Filters by
 * module === "bytebase" to match the previous Vue NotificationContext.
 */
export function pushReactNotification(item: NotificationCreate): void {
  if (item.module !== "bytebase") return;
  toastManager.add(mapNotificationToToast(item));
}

// Module-eval-time listener: catch notifications originating on the Vue side
// (Pinia notificationStore.pushNotification) and forward them to the toast
// manager. Registered exactly once per module load; main.ts imports this
// module during app bootstrap, before any pushNotification fires.
if (typeof window !== "undefined") {
  window.addEventListener(VUE_NOTIFICATION_EVENT, (event: Event) => {
    const detail = (event as CustomEvent<NotificationCreate>).detail;
    if (detail) pushReactNotification(detail);
  });
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
pnpm --dir frontend test -- frontend/src/react/lib/toast.test.ts
```

Expected: all 7 tests pass.

- [ ] **Step 5: Lint**

```bash
pnpm --dir frontend fix
pnpm --dir frontend type-check
```

Expected: green.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/react/lib/toast.ts frontend/src/react/lib/toast.test.ts
git commit -m "feat(react/lib): add toast manager + Vue-event bridge

Module-level createToastManager() (no React tree dependency) plus a
pushReactNotification(NotificationCreate) helper that maps the existing
project shape onto Base UI Toast options:

  SUCCESS/INFO/WARN/CRITICAL  ->  success/info/warning/error
  CRITICAL                    ->  priority='high', 10s timeout
  others                      ->  priority='low', 6s timeout
  manualHide=true             ->  timeout=0 (manual close)
  link + linkTitle            ->  actionProps (opens in new tab)
  module !== 'bytebase'       ->  ignored (mirrors NotificationContext filter)

Listener for 'bb.vue-notification' window events is registered at module
eval time so it's live before any pushNotification fires. <Toaster /> in
the next commit subscribes to the manager.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Add the `<Toaster />` component

The mount component that consumes the manager and renders each toast. Portals each Toast.Root into the overlay layer for accessibility-policy consistency.

**Files:**
- Create: `frontend/src/react/components/ui/toaster.tsx`

- [ ] **Step 1: Write `toaster.tsx`**

```tsx
import { Toast as BaseToast } from "@base-ui/react/toast";
import { useTranslation } from "react-i18next";
import {
  ToastRoot,
  ToastTitle,
  ToastDescription,
  ToastAction,
  ToastClose,
  type ToastVariant,
} from "./toast";
import { toastManager } from "@/react/lib/toast";
import { getLayerRoot, LAYER_Z_INDEX } from "./layer";

const TOAST_LIMIT = 5;

// Map Base UI's type string (which we set in toast.ts) onto our visual variant.
function variantFromType(type: string | undefined): ToastVariant {
  if (type === "success" || type === "warning" || type === "error") {
    return type;
  }
  return "info";
}

function ToastList() {
  const { toasts } = BaseToast.useToastManager();
  const { t } = useTranslation();
  return (
    <>
      {toasts.map((toast) => (
        <ToastRoot
          key={toast.id}
          toast={toast}
          variant={variantFromType(toast.type)}
        >
          <ToastClose aria-label={t("common.close")} />
          {toast.title ? <ToastTitle>{toast.title}</ToastTitle> : null}
          {toast.description ? (
            <ToastDescription>{toast.description}</ToastDescription>
          ) : null}
          {toast.actionProps ? <ToastAction {...toast.actionProps} /> : null}
        </ToastRoot>
      ))}
    </>
  );
}

/**
 * The Toaster shell — mounted once, persistent for the app lifetime.
 *
 * Structure: Provider supplies the context bound to the standalone
 * toastManager. The whole Viewport is portaled into getLayerRoot("overlay")
 * so toasts inherit the overlay family's aria-hidden / inert behavior
 * (e.g. session-expired surface at the 'critical' layer obscures them).
 */
export function Toaster() {
  return (
    <BaseToast.Provider toastManager={toastManager} limit={TOAST_LIMIT}>
      <BaseToast.Portal container={getLayerRoot("overlay")}>
        <BaseToast.Viewport
          className="fixed bottom-4 right-4 flex w-(--toast-width) flex-col gap-2"
          style={{
            // Tailwind v4 reads CSS vars; expose toast width here so the
            // toast card class can reference it. Width matches naive-ui's
            // default.
            ["--toast-width" as string]: "24rem",
            zIndex: LAYER_Z_INDEX.overlay,
          }}
        >
          <ToastList />
        </BaseToast.Viewport>
      </BaseToast.Portal>
    </BaseToast.Provider>
  );
}
```

- [ ] **Step 2: Verify**

```bash
pnpm --dir frontend type-check
pnpm --dir frontend fix
```

Expected: green.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/react/components/ui/toaster.tsx
git commit -m "feat(react/ui): add Toaster shell component

Renders <BaseToast.Provider> bound to the module-level toastManager,
viewport pinned to bottom-right at the overlay layer z-index. Each toast
portals into getLayerRoot('overlay') so the critical layer (session-
expired surface) can obscure it via the layer policy.

Title/description/action button + Close (with i18n aria-label from
common.close). Variant is read from Base UI's type field set by
pushReactNotification's style->type mapping.

Not yet mounted anywhere — that lands in the next commit alongside the
mountToaster wiring.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Wire the persistent React root for the Toaster

Boot a dedicated React root from `main.ts` after Vue starts. Follows the existing `mountSidebar` pattern.

**Files:**
- Create: `frontend/src/react/mountToaster.ts`
- Modify: `frontend/index.html`
- Modify: `frontend/src/main.ts`

- [ ] **Step 1: Add the DOM hook to `index.html`**

Read the current head of `frontend/index.html`, find `<div id="app"></div>`, add the toaster root after it:

```html
    <div id="app"></div>
    <!-- React Toaster root: hosts the persistent toast renderer.
         Visually empty; toasts portal into getLayerRoot("overlay"). -->
    <div id="bb-toaster-root"></div>
```

- [ ] **Step 2: Write `frontend/src/react/mountToaster.ts`**

```ts
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
 * at app bootstrap; the root lives until page unload.
 *
 * Importing this file also pulls in @/react/lib/toast, which registers the
 * `bb.vue-notification` window listener at module-eval time. Call this
 * function before any Vue code calls pushNotification.
 */
export async function mountToaster(container: HTMLElement) {
  // Side-effect import: registers the bb.vue-notification window listener.
  await import("@/react/lib/toast");
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
```

- [ ] **Step 3: Wire `mountToaster` into `frontend/src/main.ts`**

Read `frontend/src/main.ts`, find the section where `app.mount("#app")` happens (search for the end of the async IIFE), and add the Toaster mount immediately after Vue mounts:

```ts
// ... existing imports ...
import { mountToaster } from "./react/mountToaster";

// ... existing bootstrap IIFE ...
  app.mount("#app");

  // Boot the React toaster after Vue is mounted. This also registers the
  // bb.vue-notification window listener (side effect of importing
  // @/react/lib/toast inside mountToaster), so any subsequent
  // pushNotification call reaches the React renderer.
  const toasterRoot = document.getElementById("bb-toaster-root");
  if (toasterRoot) {
    void mountToaster(toasterRoot);
  }
})();
```

The exact insertion location is right after the existing `app.mount("#app")` line. Add the `import { mountToaster }` line near the other `./react/` imports if any, otherwise alongside other module imports at the top.

- [ ] **Step 4: Verify**

```bash
pnpm --dir frontend type-check
pnpm --dir frontend fix
pnpm --dir frontend test -- frontend/src/react/lib/toast.test.ts
```

Expected: all green.

- [ ] **Step 5: Commit**

```bash
git add frontend/index.html frontend/src/react/mountToaster.ts frontend/src/main.ts
git commit -m "feat(react): mount persistent Toaster root at app bootstrap

Add <div id='bb-toaster-root'> to index.html (visually empty; toasts
portal into the overlay layer). mountToaster() follows the existing
mountSidebar pattern: lazy-load via import.meta.glob to keep vue-tsc
away from .tsx, wrap with StrictMode + I18nextProvider, createRoot()
once.

Importing mountToaster pulls in @/react/lib/toast as a side effect,
which registers the bb.vue-notification window listener at module-eval
time. Bootstrap order: Vue app mounts -> Toaster mounts -> any
subsequent pushNotification reaches the React renderer.

Still inert: nothing emits bb.vue-notification yet. The next commit
flips the Pinia store and tears down the Vue renderer atomically.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Atomic switch — flip the Pinia store, drop the Vue renderer

This is the single cutover commit. All-or-nothing to avoid double-rendering.

**Files:**
- Modify: `frontend/src/store/modules/notification.ts`
- Modify: `frontend/src/react/stores/app/notification.ts`
- Modify: `frontend/src/react/stores/app/types.ts`
- Modify: `frontend/src/react/shell-bridge.ts`
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/shell-bridge.test.ts`
- Delete: `frontend/src/NotificationContext.vue`

- [ ] **Step 1: Rewrite the Pinia store to emit window events**

Replace the entire contents of `frontend/src/store/modules/notification.ts`:

```ts
import { defineStore } from "pinia";
import { v1 as uuidv1 } from "uuid";
import type { Notification, NotificationCreate } from "@/types";

const VUE_NOTIFICATION_EVENT = "bb.vue-notification";

/**
 * Notification store — kept as a Pinia store for backward compatibility
 * with the 17 Vue-side pushNotification() callers. Internally it no longer
 * queues anything; instead each push dispatches a window CustomEvent that
 * the React toast manager (frontend/src/react/lib/toast.ts) catches and
 * renders. Vue retains no renderer.
 *
 * This store dies entirely in Phase B3 (Pinia -> Zustand).
 */
export const useNotificationStore = defineStore("notification", {
  state: () => ({}),
  actions: {
    pushNotification(notificationCreate: NotificationCreate) {
      const notification: Notification = {
        id: uuidv1(),
        createdTs: Date.now() / 1000,
        ...notificationCreate,
      };
      if (typeof window !== "undefined") {
        window.dispatchEvent(
          new CustomEvent<Notification>(VUE_NOTIFICATION_EVENT, {
            detail: notification,
          })
        );
      }
    },
  },
});

export const pushNotification = (notificationCreate: NotificationCreate) => {
  useNotificationStore().pushNotification(notificationCreate);
};
```

This drops `appendNotification`, `removeNotification`, `tryPopNotification`, `notificationByModule`, and `findNotification` — none have any external consumer after this commit. Vue callers continue to use `pushNotification` unchanged.

- [ ] **Step 2: Swap React notify slice to call toastManager directly**

Replace `frontend/src/react/stores/app/notification.ts`:

```ts
import { pushReactNotification } from "@/react/lib/toast";
import type { AppSliceCreator, NotificationSlice } from "./types";

export const createNotificationSlice: AppSliceCreator<NotificationSlice> = () => ({
  notify: (notification) => {
    pushReactNotification(notification);
  },
});
```

The `notifications` array state is gone — no consumer reads it.

- [ ] **Step 3: Update the slice type**

Read `frontend/src/react/stores/app/types.ts`, find the `NotificationSlice` type:

```ts
export type NotificationSlice = {
  notifications: NotificationCreate[];
  notify: (notification: NotificationCreate) => void;
};
```

Replace with:

```ts
export type NotificationSlice = {
  notify: (notification: NotificationCreate) => void;
};
```

- [ ] **Step 4: Strip the obsolete bridge from `react/shell-bridge.ts`**

Read `frontend/src/react/shell-bridge.ts` and replace its contents:

```ts
export const ReactShellBridgeEvent = {
  localeChange: "bb.react-locale-change",
  quickstartReset: "bb.react-quickstart-reset",
} as const;

export type ReactShellBridgeEventName =
  (typeof ReactShellBridgeEvent)[keyof typeof ReactShellBridgeEvent];

export type ReactQuickstartResetDetail = {
  keys: string[];
};

export function emitReactLocaleChange(lang: string) {
  window.dispatchEvent(
    new CustomEvent(ReactShellBridgeEvent.localeChange, { detail: lang })
  );
}

export function emitReactQuickstartReset(detail: ReactQuickstartResetDetail) {
  window.dispatchEvent(
    new CustomEvent(ReactShellBridgeEvent.quickstartReset, { detail })
  );
}
```

This removes the `notification` event key and `emitReactNotification`. Notification flow now goes Vue→`bb.vue-notification`→`toastManager` directly; no React-to-Vue notification bridge.

- [ ] **Step 5: Unwrap `App.vue`**

Read `frontend/src/App.vue`, find the notification wrappers (lines 9–20 in the current file):

```vue
    <NNotificationProvider
      ...
    >
      <OverlayStackManager>
        <NotificationContext>
          ...
        </NotificationContext>
      </OverlayStackManager>
    </NNotificationProvider>
```

Replace with just the inner content (keep `OverlayStackManager`):

```vue
      <OverlayStackManager>
        ...
      </OverlayStackManager>
```

Also remove from the script imports section:

```ts
import { NConfigProvider, NNotificationProvider } from "naive-ui";
// becomes:
import { NConfigProvider } from "naive-ui";

// and delete:
import NotificationContext from "./NotificationContext.vue";
```

If `NNotificationProvider` had any prop attributes (placement, max, etc.) on the opening tag in App.vue, those go away too — Base UI Toast handles positioning at the React layer.

- [ ] **Step 6: Update test mocks**

Read `frontend/src/shell-bridge.test.ts`. Find the block mocking `NotificationContext.vue`:

```ts
vi.mock("./components/misc/OverlayStackManager.vue", async () => { ... });
```

And the `useNotificationStore` mock around line 22–30:

```ts
const mocks = vi.hoisted(() => ({
  // ...
  tryPopNotification: vi.fn(),
  // ...
}));

// later:
useNotificationStore: () => ({
  // ...
  tryPopNotification: mocks.tryPopNotification,
  // ...
}),
```

Remove `tryPopNotification` from `mocks` and from the `useNotificationStore` mock object. If the test imports `NotificationContext.vue` directly, drop that mock entry.

Then read `frontend/src/layouts/layout-bridge.test.ts` and check for `NotificationContext` or `tryPopNotification` mock entries — remove if present.

Final pass:

```bash
grep -rn "NotificationContext\|tryPopNotification" frontend/src
```

Expected: zero matches outside this commit's deletions.

- [ ] **Step 7: Delete `NotificationContext.vue`**

```bash
git rm frontend/src/NotificationContext.vue
```

- [ ] **Step 8: Verify all gates**

```bash
pnpm --dir frontend fix
pnpm --dir frontend type-check
pnpm --dir frontend test
pnpm --dir frontend check
```

Expected: all green. The full test suite must still pass (~1871 tests). The `check` script runs eslint + biome + react-i18n + react-layering + locale-sort.

- [ ] **Step 9: Manual smoke (see full checklist in spec §"Manual smoke checklist")**

Run dev server and exercise:

```bash
pnpm --dir frontend dev
```

Quick subset:
- Trigger a SUCCESS toast (save anything) → green, auto-dismisses ~6s
- Trigger a CRITICAL toast (force an error, e.g. SQL execution error) → red, ~10s
- Trigger a toast with `manualHide: true` → stays until close button click
- Open a Dialog, then trigger a toast → toast appears above the dialog backdrop
- Trigger session-expired (clear auth cookie + reload) → critical layer obscures any toasts

Full checklist is in the spec. Do not commit until you've smoked at least the four above.

- [ ] **Step 10: Commit**

```bash
git add -A frontend
git commit -m "refactor(notification): React Base UI Toast replaces NotificationContext.vue

Atomic cutover from Vue/Naive UI notifications to React/Base UI Toast.
After this commit:
- The Pinia notificationStore is a thin pass-through that emits
  CustomEvent('bb.vue-notification') for each pushNotification. No
  internal queue; no renderer.
- React @/react/stores/app/notification's notify() calls toastManager.add()
  directly through pushReactNotification (no longer emits a window event).
- react/shell-bridge.ts loses the bb.react-notification event +
  emitReactNotification; bridge direction is now Vue -> React only.
- App.vue drops <NNotificationProvider> + <NotificationContext> wrappers.
- NotificationContext.vue + its test mocks deleted.

149 callers of pushNotification (17 Vue, 132 React) keep their API. Vue
callers route via the window event; React callers route via the slice ->
manager direct path. End state: one renderer (React).

Phase B2 dependencies retired: -1 .vue file (NotificationContext), -1
Naive UI provider mount. Pinia store stays until Phase B3.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Pre-PR validation

- [ ] **Step 1: Walk the pre-PR checklist**

Open `docs/pre-pr-checklist.md` and verify every item. In particular:

- No `frontend/src/NotificationContext.vue` left behind: `find frontend/src -name "NotificationContext.vue"` — should be empty.
- No leftover `tryPopNotification` calls: `grep -rn "tryPopNotification" frontend/src` — empty.
- No leftover `emitReactNotification`: `grep -rn "emitReactNotification" frontend/src` — empty.
- No leftover `bb.react-notification`: `grep -rn "bb.react-notification" frontend/src` — empty.
- No raw `z-index` introduced anywhere except `LAYER_Z_INDEX.overlay` in `toaster.tsx`.

- [ ] **Step 2: Push and open PR**

```bash
git push -u origin chore/frontend/notification-react-migration
gh pr create --title "refactor(notification): migrate to React Base UI Toast" --body "$(cat <<'EOF'
## Summary

Replaces Vue's NotificationContext.vue + Naive UI NNotificationProvider with a React renderer built on Base UI Toast. React becomes the sole renderer; Vue's pushNotification API stays for backwards compatibility with its 17 callers, routing through a window event into the React side.

- 5 commits, ~3 files added / ~6 modified / 1 deleted
- Pinia notificationStore retained as a thin pass-through (Phase B3 work)
- App.vue still Vue; only the notification wrapper is stripped

## Test plan

- [x] `pnpm --dir frontend type-check`
- [x] `pnpm --dir frontend test`
- [x] `pnpm --dir frontend check`
- [ ] Manual smoke (see spec §"Manual smoke checklist"):
  - [ ] SUCCESS / INFO / WARN / CRITICAL variants
  - [ ] manualHide
  - [ ] link + linkTitle action button
  - [ ] toast above dialogs / sheets
  - [ ] critical layer (session-expired) obscures toasts
  - [ ] auth middleware error path still produces a toast

Companion docs:
- Spec: `docs/plans/2026-05-13-notification-react-migration-design.md`
- Plan: `docs/plans/2026-05-13-notification-react-migration-plan.md`

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

## Out-of-scope follow-ups

- **Phase B3**: Pinia `notificationStore` → Zustand. After that, the `bb.vue-notification` window event can collapse into a direct function call.
- **Phase B2**: When `App.vue` migrates to React, the `<OverlayStackManager>` wrapper goes too; the Toaster's persistent root can fold into the main React shell root.
- **Toast API extensions** (action toasts that don't just open links, promise toasts via Base UI's `toastManager.promise()`, sticky toasts with custom dismiss buttons). Not needed for parity.

---

## Self-review notes (inline)

**Spec coverage check:**
- §"Architecture" → Tasks 2 (manager) + 3 (Toaster) + 5 (Pinia rewrite) cover the diagram end-to-end. ✓
- §"File layout / Created" → Task 1 (toast.tsx), Task 2 (toast.ts), Task 3 (toaster.tsx), Task 4 (mountToaster.ts). ✓
- §"File layout / Modified" → Task 5 covers notification.ts (slice), notification.ts (Pinia), shell-bridge.ts, App.vue, main.ts (via Task 4). ✓
- §"API parity" → mapping function tested in Task 2 covers SUCCESS/INFO/WARN/CRITICAL, manualHide, link/linkTitle. ✓
- §"Accessibility" → `priority` mapping in Task 2; `aria-label={t("common.close")}` in Task 3. ✓
- §"Layering" → `Toast.Portal container={getLayerRoot("overlay")}` in Task 3; `LAYER_Z_INDEX.overlay` on viewport. ✓
- §"i18n" → `common.close` reused; no new keys. ✓
- §"Risks" #1 (listener order) → Task 2 module-eval-time listener registration; Task 4 mounts after Vue. ✓
- §"Risks" #2 (duplicate toasts) → Task 5 is atomic. ✓
- §"Risks" #3 (function-typed description) → grep audit found zero callers; mapping types description as string only. ✓
- §"Risks" #4 (test mocks) → Task 5 Step 6. ✓
- §"Risks" #5 (Pinia consumers of tryPopNotification) → Task 5 Step 1 drops it; Step 6 verifies grep is empty. ✓

**Type consistency check:** `pushReactNotification` signature and `mapNotificationToToast` return type are used consistently across Tasks 2, 3, 5. `NotificationCreate` shape comes from the existing `@/types/notification` and is not modified.

**Placeholder scan:** No TBDs. Every code step has full code. Every command has an expected outcome.

**Estimated total:** 5 logical commits (branch setup + Tasks 1–5) plus Task 6 (push + PR). Roughly 1 day of focused work.
