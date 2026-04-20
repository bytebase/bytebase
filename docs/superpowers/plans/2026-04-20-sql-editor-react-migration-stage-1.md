# SQL Editor React Migration — Stage 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate `frontend/src/views/sql-editor/EditorPanel/Welcome/Welcome.vue` (77 lines, 2 buttons gated by permissions) to React, establishing the bottom-up React-island pattern for the SQL Editor with exact UX parity.

**Architecture:** Keep the Vue orchestrator (`SQLEditorLayout`, `SQLEditorPage`, all `provide/inject` contexts, all Pinia stores). Embed React leaves inline inside the Vue tree using the existing `frontend/src/react/ReactPageMount.vue`. The React `Welcome` leaf accesses Pinia stores, the router, and permissions directly; it receives exactly one Vue-context-coupled callback (`onChangeConnection`) as a prop from its Vue parent (`StandardPanel.vue`).

**Tech Stack:** React 18, `@base-ui/react`, Tailwind CSS v4, `class-variance-authority`, `react-i18next`, `vitest`, Vue 3 (parent side), Pinia. Existing hooks/utilities: `useVueState`, `usePermissionCheck`.

**Reference spec:** `docs/superpowers/specs/2026-04-20-sql-editor-react-migration-stage-1-design.md`

**Workflow note:** **Do not auto-commit** — user commits manually. Each task ends with a "Stop for user review" checkpoint. The user will stage and commit at their own cadence.

---

## File Structure

| File | Action | Responsibility |
|---|---|---|
| `frontend/src/react/locales/{en-US,zh-CN,es-ES,ja-JP,vi-VN}.json` | Modify | Add two keys: `sql-editor.add-a-new-instance`, `sql-editor.connect-to-a-database`. Strict 1:1 parity enforced by `check-react-i18n.mjs`. |
| `frontend/src/react/components/BytebaseLogo.tsx` | Create | Shared logo: renders workspace custom logo if present, else fallback SVG. Reads `useWorkspaceV1Store` via `useVueState`. Supports optional `redirect` prop for future callers. |
| `frontend/src/react/components/BytebaseLogo.test.tsx` | Create | Unit tests for both logo states. |
| `frontend/src/react/components/sql-editor/WelcomeButton.tsx` | Create | Large square icon-over-label button. Mirrors Vue `Button.vue`. Built on shadcn `Button` with a `cva` variant for 7rem × 7rem shape. |
| `frontend/src/react/components/sql-editor/WelcomeButton.test.tsx` | Create | Unit tests for icon/label slots and click. |
| `frontend/src/react/components/sql-editor/Welcome.tsx` | Create | The leaf. Two conditionally-rendered `WelcomeButton`s gated by `usePermissionCheck`. Router push for Add-Instance; `onChangeConnection` prop callback for Connect. |
| `frontend/src/react/components/sql-editor/Welcome.test.tsx` | Create | Unit tests for permission gating + callback wiring. |
| `frontend/src/react/mount.ts` | Modify | Register `./components/sql-editor/*.tsx` glob so `ReactPageMount` can load `Welcome`. |
| `frontend/src/views/sql-editor/EditorPanel/StandardPanel/StandardPanel.vue` | Modify | Replace `<Welcome v-else />` with `<ReactPageMount v-else page="Welcome" :on-change-connection="changeConnection" />`; add `showConnectionPanel`, `asidePanelTab` to existing `useSQLEditorContext()` destructure; define `changeConnection`. |
| `frontend/src/views/sql-editor/EditorPanel/Welcome/` (entire dir) | Delete | `Welcome.vue`, `Button.vue`, `index.ts` — orphaned after the swap. |

---

## Task 1: Add React i18n keys

**Goal:** Ensure `t("sql-editor.add-a-new-instance")` and `t("sql-editor.connect-to-a-database")` resolve in the React i18n resource. `check-react-i18n.mjs` enforces 1:1 parity across all 5 locales — must add to all five.

**Files:**
- Modify: `frontend/src/react/locales/en-US.json`
- Modify: `frontend/src/react/locales/zh-CN.json`
- Modify: `frontend/src/react/locales/es-ES.json`
- Modify: `frontend/src/react/locales/ja-JP.json`
- Modify: `frontend/src/react/locales/vi-VN.json`

Reference values (already in Vue locales at `frontend/src/locales/*.json`):

| Locale | `add-a-new-instance` | `connect-to-a-database` |
|---|---|---|
| en-US | `Add new instance` | `Connect to database` |
| zh-CN | (copy from `frontend/src/locales/zh-CN.json`) | (copy from same) |
| es-ES | `Agregar nueva instancia` | `Conectar a base de datos` |
| ja-JP | `新しいインスタンスを追加` | (copy from `frontend/src/locales/ja-JP.json`) |
| vi-VN | (copy from `frontend/src/locales/vi-VN.json`) | (copy from same) |

- [ ] **Step 1: Look up the exact values in the Vue locale files**

Run:
```bash
for f in en-US zh-CN es-ES ja-JP vi-VN; do
  echo "=== $f ==="
  grep -n "add-a-new-instance\|connect-to-a-database" \
    "frontend/src/locales/$f.json"
done
```

Record the exact strings from each locale. These are the source of truth — do not translate independently.

- [ ] **Step 2: Find the `sql-editor` block in each React locale file**

Each React locale has a `sql-editor` block (en-US.json line 2102). The keys are alphabetically sorted within the block. Example current state in `en-US.json`:

```json
  "sql-editor": {
    "access-grants": "Access Grants",
    "access-type-unmask": "See unmasked sensitive data",
    "activate-access": "Activate",
    ...
  }
```

- [ ] **Step 3: Add the two keys alphabetically in each locale**

In `frontend/src/react/locales/en-US.json`, inside the `"sql-editor"` object, insert:

```json
    "access-grants": "Access Grants",
    "access-type-unmask": "See unmasked sensitive data",
    "activate-access": "Activate",
    "activate-confirm": "Are you sure you want to activate this access grant?",
    "add-a-new-instance": "Add new instance",
    ...
    "connect-to-a-database": "Connect to database",
    ...
```

Repeat for the other 4 locales using the values from Step 1.

- [ ] **Step 4: Run the sort script to confirm ordering is canonical**

Run:
```bash
node frontend/scripts/sort_i18n_keys.mjs
```

Expected: file is re-sorted in place (no-op if Step 3 was ordered correctly). Inspect the diff — should only touch the 2 new keys.

- [ ] **Step 5: Run the React i18n consistency check**

Run:
```bash
node frontend/scripts/check-react-i18n.mjs
```

Expected: no "missing key" or "consistency" errors for these two keys across the 5 locales. (There will be "unused" warnings for them until Tasks 4/6 land — that is expected for now. If the script fails hard on unused, skip this verification step until after Task 6.)

- [ ] **Step 6: Stop for user review**

Report: "i18n keys added to 5 React locale files. Ready for commit." Wait for user to commit or request changes.

---

## Task 2: Create `BytebaseLogo.tsx` (TDD)

**Goal:** A React-native `BytebaseLogo` that replaces `frontend/src/components/BytebaseLogo.vue`. Welcome uses it without a `redirect` prop, but we build it fully so future React pages (and Stage 9 shell) can reuse.

**Files:**
- Create: `frontend/src/react/components/BytebaseLogo.tsx`
- Create: `frontend/src/react/components/BytebaseLogo.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/react/components/BytebaseLogo.test.tsx`:

```tsx
import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/assets/logo-full.svg", () => ({
  default: "/assets/logo-full.svg",
}));

let BytebaseLogo: typeof import("./BytebaseLogo").BytebaseLogo;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  ({ BytebaseLogo } = await import("./BytebaseLogo"));
});

describe("BytebaseLogo", () => {
  test("renders custom workspace logo when present", () => {
    mocks.useVueState.mockReturnValue("https://example.com/logo.png");
    const { container, render, unmount } = renderIntoContainer(<BytebaseLogo />);
    render();
    const img = container.querySelector("img");
    expect(img).not.toBeNull();
    expect(img?.getAttribute("src")).toBe("https://example.com/logo.png");
    expect(img?.getAttribute("alt")).toBe("branding logo");
    unmount();
  });

  test("renders fallback Bytebase SVG when workspace has no custom logo", () => {
    mocks.useVueState.mockReturnValue("");
    const { container, render, unmount } = renderIntoContainer(<BytebaseLogo />);
    render();
    const img = container.querySelector("img");
    expect(img).not.toBeNull();
    expect(img?.getAttribute("src")).toBe("/assets/logo-full.svg");
    expect(img?.getAttribute("alt")).toBe("Bytebase");
    unmount();
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```bash
pnpm --dir frontend test -- BytebaseLogo.test --run
```

Expected: FAIL with "Cannot find module './BytebaseLogo'".

- [ ] **Step 3: Write the minimal implementation**

Create `frontend/src/react/components/BytebaseLogo.tsx`:

```tsx
import logoFull from "@/assets/logo-full.svg";
import { useVueState } from "@/react/hooks/useVueState";
import { useWorkspaceV1Store } from "@/store";

type Props = {
  /** Optional route name — when set, the logo is wrapped in a link that records the visit. */
  readonly redirect?: string;
};

/**
 * Replaces frontend/src/components/BytebaseLogo.vue. Shows the workspace's
 * custom logo when set, otherwise the bundled Bytebase fallback SVG.
 *
 * The `redirect` prop is supported for future callers. It is not used by
 * the SQL Editor Welcome screen. Router-link behavior will be added when
 * the first React caller needs it.
 */
export function BytebaseLogo(_props: Props) {
  const customLogo = useVueState(
    () => useWorkspaceV1Store().currentWorkspace?.logo ?? ""
  );

  return (
    <div className="shrink-0 max-w-44 flex items-center overflow-hidden">
      <span className="h-full w-full select-none flex flex-row justify-center items-center">
        {customLogo ? (
          <img
            src={customLogo}
            alt="branding logo"
            className="h-full object-contain"
          />
        ) : (
          <img
            src={logoFull}
            alt="Bytebase"
            className="h-8 md:h-10 w-auto object-contain"
          />
        )}
      </span>
    </div>
  );
}
```

Note: The `redirect`/router-link branch from the Vue original is intentionally omitted for now. Welcome does not use it, and adding React Router link support requires more wiring. When a future caller needs it, extend this file.

- [ ] **Step 4: Run the test to verify it passes**

Run:
```bash
pnpm --dir frontend test -- BytebaseLogo.test --run
```

Expected: PASS (2 tests).

- [ ] **Step 5: Stop for user review**

Report: "BytebaseLogo.tsx + test created and passing. Ready for commit."

---

## Task 3: Create `WelcomeButton.tsx` (TDD)

**Goal:** A big square button with vertical icon-over-label layout. Mirrors `frontend/src/views/sql-editor/EditorPanel/Welcome/Button.vue` which wraps naive-ui's `NButton` with custom slot layout (height `7rem`, min-width `7rem`, vertical flex with `gap-2`).

**Files:**
- Create: `frontend/src/react/components/sql-editor/WelcomeButton.tsx`
- Create: `frontend/src/react/components/sql-editor/WelcomeButton.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/react/components/sql-editor/WelcomeButton.test.tsx`:

```tsx
import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

let WelcomeButton: typeof import("./WelcomeButton").WelcomeButton;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

beforeEach(async () => {
  ({ WelcomeButton } = await import("./WelcomeButton"));
});

describe("WelcomeButton", () => {
  test("renders icon above label", () => {
    const { container, render, unmount } = renderIntoContainer(
      <WelcomeButton icon={<span data-testid="icon">I</span>}>
        Hello
      </WelcomeButton>
    );
    render();
    const button = container.querySelector("button");
    expect(button?.textContent).toContain("Hello");
    expect(container.querySelector('[data-testid="icon"]')).not.toBeNull();
    unmount();
  });

  test("invokes onClick when clicked", () => {
    const handler = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <WelcomeButton icon={<span>I</span>} onClick={handler}>
        Click
      </WelcomeButton>
    );
    render();
    const button = container.querySelector("button");
    act(() => {
      button?.click();
    });
    expect(handler).toHaveBeenCalledTimes(1);
    unmount();
  });

  test("applies primary variant classes by default", () => {
    const { container, render, unmount } = renderIntoContainer(
      <WelcomeButton icon={<span>I</span>}>Primary</WelcomeButton>
    );
    render();
    const button = container.querySelector("button");
    expect(button?.className).toContain("bg-accent");
    unmount();
  });

  test("applies secondary variant classes when variant='secondary'", () => {
    const { container, render, unmount } = renderIntoContainer(
      <WelcomeButton icon={<span>I</span>} variant="secondary">
        Secondary
      </WelcomeButton>
    );
    render();
    const button = container.querySelector("button");
    expect(button?.className).toContain("border-control-border");
    unmount();
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```bash
pnpm --dir frontend test -- WelcomeButton.test --run
```

Expected: FAIL with "Cannot find module './WelcomeButton'".

- [ ] **Step 3: Write the minimal implementation**

Create `frontend/src/react/components/sql-editor/WelcomeButton.tsx`:

```tsx
import { Button as BaseButton } from "@base-ui/react/button";
import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps, ReactNode } from "react";
import { cn } from "@/react/lib/utils";

/**
 * Big square icon-over-label button used on the SQL Editor Welcome screen.
 * Mirrors frontend/src/views/sql-editor/EditorPanel/Welcome/Button.vue,
 * which wraps NButton at 7rem × 7rem with a vertical flex content slot.
 */
const welcomeButtonVariants = cva(
  "inline-flex flex-col items-center justify-center gap-2 min-w-28 h-28 px-4 rounded-xs text-sm font-medium cursor-pointer transition-colors focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
  {
    variants: {
      variant: {
        primary: "bg-accent text-accent-text hover:bg-accent-hover",
        secondary:
          "border border-control-border bg-transparent text-control hover:bg-control-bg",
      },
    },
    defaultVariants: {
      variant: "primary",
    },
  }
);

type WelcomeButtonProps = Omit<ComponentProps<"button">, "children"> &
  VariantProps<typeof welcomeButtonVariants> & {
    readonly icon: ReactNode;
    readonly children: ReactNode;
  };

export function WelcomeButton({
  icon,
  children,
  variant,
  className,
  ref,
  ...props
}: WelcomeButtonProps) {
  return (
    <BaseButton
      ref={ref}
      className={cn(welcomeButtonVariants({ variant, className }))}
      {...props}
    >
      <span className="flex items-center justify-center">{icon}</span>
      <span>{children}</span>
    </BaseButton>
  );
}
```

Note on sizing: Vue's original uses `--n-height: 7rem` and `min-width: 7rem`. Tailwind's `h-28` is exactly `7rem` (112px); `min-w-28` is `7rem`. The visual result matches.

- [ ] **Step 4: Run the test to verify it passes**

Run:
```bash
pnpm --dir frontend test -- WelcomeButton.test --run
```

Expected: PASS (4 tests).

- [ ] **Step 5: Stop for user review**

Report: "WelcomeButton.tsx + test created and passing. Ready for commit."

---

## Task 4: Create `Welcome.tsx` (TDD)

**Goal:** The React leaf replacing `Welcome.vue`. Renders `BytebaseLogo` + two conditionally-visible `WelcomeButton`s. "Add a new instance" is gated by `bb.instances.create` and routes via `router.push({ name: INSTANCE_ROUTE_DASHBOARD, hash: "#add" })`. "Connect to a database" is gated by `bb.sql.select` against the current SQL Editor project and calls the `onChangeConnection` prop.

**Files:**
- Create: `frontend/src/react/components/sql-editor/Welcome.tsx`
- Create: `frontend/src/react/components/sql-editor/Welcome.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/react/components/sql-editor/Welcome.test.tsx`:

```tsx
import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  usePermissionCheck: vi.fn<
    (perms: readonly string[], project?: unknown) => [boolean, string | undefined]
  >(),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  routerPush: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/components/PermissionGuard", () => ({
  usePermissionCheck: mocks.usePermissionCheck,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/router", () => ({
  router: { push: mocks.routerPush },
}));

vi.mock("@/router/dashboard/workspaceRoutes", () => ({
  INSTANCE_ROUTE_DASHBOARD: "workspace.instance",
}));

vi.mock("@/store", () => ({
  useSQLEditorStore: vi.fn(),
  useProjectV1Store: vi.fn(),
  useWorkspaceV1Store: vi.fn(),
}));

vi.mock("@/assets/logo-full.svg", () => ({
  default: "/assets/logo-full.svg",
}));

let Welcome: typeof import("./Welcome").Welcome;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  // Default: workspace logo empty, both permissions granted.
  mocks.useVueState.mockReturnValue("");
  mocks.usePermissionCheck.mockReturnValue([true, undefined]);
  ({ Welcome } = await import("./Welcome"));
});

describe("Welcome", () => {
  test("renders both buttons when both permissions present", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();
    expect(container.textContent).toContain("sql-editor.add-a-new-instance");
    expect(container.textContent).toContain("sql-editor.connect-to-a-database");
    unmount();
  });

  test("hides Add-Instance when missing bb.instances.create", () => {
    mocks.usePermissionCheck.mockImplementation((perms) => {
      if (perms.includes("bb.instances.create")) return [false, "missing"];
      return [true, undefined];
    });
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();
    expect(container.textContent).not.toContain("sql-editor.add-a-new-instance");
    expect(container.textContent).toContain("sql-editor.connect-to-a-database");
    unmount();
  });

  test("hides Connect when missing bb.sql.select", () => {
    mocks.usePermissionCheck.mockImplementation((perms) => {
      if (perms.includes("bb.sql.select")) return [false, "missing"];
      return [true, undefined];
    });
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();
    expect(container.textContent).toContain("sql-editor.add-a-new-instance");
    expect(container.textContent).not.toContain("sql-editor.connect-to-a-database");
    unmount();
  });

  test("hides both buttons when neither permission present", () => {
    mocks.usePermissionCheck.mockReturnValue([false, "missing"]);
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();
    expect(container.querySelectorAll("button")).toHaveLength(0);
    unmount();
  });

  test("routes to instance dashboard with #add hash on Add-Instance click", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();
    const buttons = container.querySelectorAll("button");
    // Add-Instance is the first button (matches Vue order).
    act(() => {
      (buttons[0] as HTMLButtonElement).click();
    });
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.instance",
      hash: "#add",
    });
    unmount();
  });

  test("invokes onChangeConnection on Connect click", () => {
    const onChangeConnection = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={onChangeConnection} />
    );
    render();
    const buttons = container.querySelectorAll("button");
    // Connect is the second button (matches Vue order).
    act(() => {
      (buttons[1] as HTMLButtonElement).click();
    });
    expect(onChangeConnection).toHaveBeenCalledTimes(1);
    unmount();
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```bash
pnpm --dir frontend test -- Welcome.test --run
```

Expected: FAIL with "Cannot find module './Welcome'".

- [ ] **Step 3: Write the minimal implementation**

Create `frontend/src/react/components/sql-editor/Welcome.tsx`:

```tsx
import { LayersIcon, LinkIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { BytebaseLogo } from "@/react/components/BytebaseLogo";
import { usePermissionCheck } from "@/react/components/PermissionGuard";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { INSTANCE_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import { useProjectV1Store, useSQLEditorStore } from "@/store";
import { WelcomeButton } from "./WelcomeButton";

export type WelcomeProps = {
  /**
   * Called when the user clicks "Connect to a database". Vue parent
   * passes a callback that sets `asidePanelTab = "SCHEMA"` and
   * `showConnectionPanel = true` on the Vue-side SQL Editor context.
   */
  readonly onChangeConnection: () => void;
};

export function Welcome({ onChangeConnection }: WelcomeProps) {
  const { t } = useTranslation();

  const project = useVueState(() => {
    const projectName = useSQLEditorStore().project;
    return projectName
      ? useProjectV1Store().getProjectByName(projectName)
      : undefined;
  });

  const [showCreateInstanceButton] = usePermissionCheck([
    "bb.instances.create",
  ]);
  const [showConnectButton] = usePermissionCheck(["bb.sql.select"], project);

  const handleCreateInstance = () => {
    router.push({
      name: INSTANCE_ROUTE_DASHBOARD,
      hash: "#add",
    });
  };

  return (
    <div className="w-full flex-1 flex flex-col items-center justify-center gap-y-4">
      <BytebaseLogo />
      <div className="flex items-center flex-wrap gap-4">
        {showCreateInstanceButton && (
          <WelcomeButton
            variant="secondary"
            icon={<LayersIcon strokeWidth={1.5} className="size-8" />}
            onClick={handleCreateInstance}
          >
            {t("sql-editor.add-a-new-instance")}
          </WelcomeButton>
        )}
        {showConnectButton && (
          <WelcomeButton
            variant="primary"
            icon={<LinkIcon strokeWidth={1.5} className="size-8" />}
            onClick={onChangeConnection}
          >
            {t("sql-editor.connect-to-a-database")}
          </WelcomeButton>
        )}
      </div>
    </div>
  );
}
```

Note on variant mapping: The Vue original uses `type="default"` for Add-Instance (which renders like a bordered/secondary button) and `secondary` + `type="primary"` for Connect (which renders as the accent-filled primary). In our simpler 2-variant model: Add-Instance → `secondary`, Connect → `primary`. Verify visually during Task 8.

- [ ] **Step 4: Run the test to verify it passes**

Run:
```bash
pnpm --dir frontend test -- Welcome.test --run
```

Expected: PASS (6 tests).

- [ ] **Step 5: Stop for user review**

Report: "Welcome.tsx + test created and passing. Ready for commit."

---

## Task 5: Register the new glob in `mount.ts`

**Goal:** Make `<ReactPageMount page="Welcome" />` resolve the new file at `frontend/src/react/components/sql-editor/Welcome.tsx`.

**Files:**
- Modify: `frontend/src/react/mount.ts:1-18` and `:52-58`

- [ ] **Step 1: Read current state**

Open `frontend/src/react/mount.ts`. Confirm the current `pageLoaders` and `pageDirs` structure matches what's described below.

- [ ] **Step 2: Add the SQL Editor glob**

Change the top of the file from:

```ts
const settingsPageLoaders = import.meta.glob("./pages/settings/*.tsx");
const projectPageLoaders = import.meta.glob("./pages/project/*.tsx");
const pluginComponentLoaders = import.meta.glob(
  "./plugins/agent/components/AgentWindow.tsx"
);
const workspacePageLoaders = import.meta.glob("./pages/workspace/*.tsx");
const authComponentLoaders = import.meta.glob(
  "./components/auth/SessionExpiredSurface.tsx"
);
const pageLoaders = {
  ...settingsPageLoaders,
  ...projectPageLoaders,
  ...pluginComponentLoaders,
  ...workspacePageLoaders,
  ...authComponentLoaders,
};
```

To:

```ts
const settingsPageLoaders = import.meta.glob("./pages/settings/*.tsx");
const projectPageLoaders = import.meta.glob("./pages/project/*.tsx");
const pluginComponentLoaders = import.meta.glob(
  "./plugins/agent/components/AgentWindow.tsx"
);
const workspacePageLoaders = import.meta.glob("./pages/workspace/*.tsx");
const authComponentLoaders = import.meta.glob(
  "./components/auth/SessionExpiredSurface.tsx"
);
const sqlEditorComponentLoaders = import.meta.glob(
  "./components/sql-editor/*.tsx"
);
const pageLoaders = {
  ...settingsPageLoaders,
  ...projectPageLoaders,
  ...pluginComponentLoaders,
  ...workspacePageLoaders,
  ...authComponentLoaders,
  ...sqlEditorComponentLoaders,
};
```

Then change `pageDirs` from:

```ts
const pageDirs = [
  "./pages/settings",
  "./pages/project",
  "./plugins/agent/components",
  "./pages/workspace",
  "./components/auth",
];
```

To:

```ts
const pageDirs = [
  "./pages/settings",
  "./pages/project",
  "./plugins/agent/components",
  "./pages/workspace",
  "./components/auth",
  "./components/sql-editor",
];
```

Important: The `*.tsx` glob also matches `*.test.tsx` and internal building blocks like `WelcomeButton.tsx` — every file resolves into `pageLoaders`. That is safe because `resolveReactPagePath` requires the caller to pass the explicit basename as the `page` string (e.g. `page="Welcome"`), and nobody will type `page="Welcome.test"` or `page="WelcomeButton"`. This matches the behavior of the existing `pages/settings/*.tsx` glob, which also picks up test files (`environmentSelection.test.tsx`) and non-page utilities (`environmentSelection.ts`) without issue. If a stricter filter becomes valuable later, add an exclusion pattern to the glob — deferred for now.

- [ ] **Step 3: Run type-check**

Run:
```bash
pnpm --dir frontend type-check
```

Expected: pass. (`mount.ts` has no semantic changes, just new glob strings.)

- [ ] **Step 4: Stop for user review**

Report: "mount.ts updated with ./components/sql-editor glob. Ready for commit."

---

## Task 6: Swap `StandardPanel.vue` to use `ReactPageMount`

**Goal:** Replace `<Welcome v-else />` inside `EditorPanel/StandardPanel/StandardPanel.vue` with `<ReactPageMount v-else page="Welcome" :on-change-connection="changeConnection" />`. Pull the Vue-context-coupled action (`changeConnection`) up to this component since `Welcome` no longer has direct access to `useSQLEditorContext`.

**Files:**
- Modify: `frontend/src/views/sql-editor/EditorPanel/StandardPanel/StandardPanel.vue:55` (template) and `:98,105-106` (script)

- [ ] **Step 1: Read the current state**

Open `frontend/src/views/sql-editor/EditorPanel/StandardPanel/StandardPanel.vue`. Confirm:

- Line 55: `<Welcome v-else />`
- Line 90: `import { useSQLEditorContext } from "../../context";`
- Line 98: `import Welcome from "../Welcome";`
- Line 105-106: `const { showAIPanel, editorPanelSize, handleEditorPanelResize } = useSQLEditorContext();`

- [ ] **Step 2: Replace the template, import, and destructure**

Change line 55 from:

```vue
<Welcome v-else />
```

To:

```vue
<ReactPageMount
  v-else
  page="Welcome"
  :on-change-connection="changeConnection"
/>
```

Change line 98 from:

```ts
import Welcome from "../Welcome";
```

To:

```ts
import ReactPageMount from "@/react/ReactPageMount.vue";
```

Change lines 105-106 from:

```ts
const { showAIPanel, editorPanelSize, handleEditorPanelResize } =
  useSQLEditorContext();
```

To:

```ts
const {
  showAIPanel,
  editorPanelSize,
  handleEditorPanelResize,
  showConnectionPanel,
  asidePanelTab,
} = useSQLEditorContext();

const changeConnection = () => {
  asidePanelTab.value = "SCHEMA";
  showConnectionPanel.value = true;
};
```

- [ ] **Step 3: Run type-check**

Run:
```bash
pnpm --dir frontend type-check
```

Expected: pass. The new import path, the added destructure fields, and the new `changeConnection` function are all existing API surface. `ReactPageMount.vue` accepts arbitrary `attrs` via `useAttrs()` so the kebab-case prop is passed through.

- [ ] **Step 4: Run the React i18n consistency check**

Run:
```bash
node frontend/scripts/check-react-i18n.mjs
```

Expected: pass (the two keys added in Task 1 now have a consumer in `Welcome.tsx`).

- [ ] **Step 5: Start the dev server and smoke-test UX parity**

Run (in a background terminal):
```bash
pnpm --dir frontend dev
```

Then in the browser, open the SQL Editor route and verify:

1. Open `/sql-editor` with no active connection → Welcome renders with: Bytebase logo at top, 2 buttons ("Add new instance" + "Connect to database") below.
2. Click "Add new instance" → navigates to `/instance#add`.
3. Go back to `/sql-editor`. Click "Connect to database" → connection panel opens on the right AND the aside panel switches to the SCHEMA tab.
4. Toggle the app language (English ↔ Chinese) → button labels update.
5. Workspace with custom logo → the custom logo renders in place of the Bytebase SVG. (Temporarily set one in `Settings → General → Logo` to verify.)
6. Sign in as a viewer-only user (no `bb.instances.create`) → only the "Connect to a database" button shows.
7. Sign in as a user without `bb.sql.select` → only the "Add new instance" button shows. (May require an account with workspace admin but no project role.)

For each state, take a side-by-side screenshot against the pre-change Vue version (you can capture this by stashing Task 6 changes locally: `git stash push frontend/src/views/sql-editor/EditorPanel/StandardPanel/StandardPanel.vue`, screenshot, `git stash pop`).

Document each state with a screenshot for the PR description.

- [ ] **Step 6: Stop for user review**

Report: "StandardPanel.vue swapped to ReactPageMount. Dev server smoke test PASSED/FAILED (with notes). Screenshots captured. Ready for commit."

---

## Task 7: Verify orphan status and delete the Vue `Welcome/` directory

**Goal:** Confirm no other live caller imports from `EditorPanel/Welcome/`, then delete the orphaned Vue files per playbook §Deletion.

**Files:**
- Delete: `frontend/src/views/sql-editor/EditorPanel/Welcome/Welcome.vue`
- Delete: `frontend/src/views/sql-editor/EditorPanel/Welcome/Button.vue`
- Delete: `frontend/src/views/sql-editor/EditorPanel/Welcome/index.ts`
- Delete: `frontend/src/views/sql-editor/EditorPanel/Welcome/` (the now-empty directory)

- [ ] **Step 1: Search for remaining callers**

Use Grep (or ripgrep directly if needed):

Search pattern 1 — imports of the directory:
```
from ['\"].*EditorPanel/Welcome['\"]
```
Scope: `frontend/src`.

Search pattern 2 — imports of specific files:
```
from ['\"].*EditorPanel/Welcome/Welcome['\"]
from ['\"].*EditorPanel/Welcome/Button['\"]
```
Scope: `frontend/src`.

Search pattern 3 — the Vue component tag:
```
<Welcome[\\s/>]
```
Scope: `frontend/src/views/sql-editor`. (Filter the Vue files — the `Welcome.tsx` in React is not relevant here.)

Expected: After Task 6, all three patterns return **zero matches outside the files being deleted**. If any caller remains, stop and investigate — do NOT delete.

- [ ] **Step 2: Delete the files**

Run:
```bash
rm -rf frontend/src/views/sql-editor/EditorPanel/Welcome
```

- [ ] **Step 3: Run type-check and test**

Run:
```bash
pnpm --dir frontend type-check
pnpm --dir frontend test --run
```

Expected: both pass. If type-check fails with "cannot find module ../Welcome", Task 6's import change was missed — revisit.

- [ ] **Step 4: Stop for user review**

Report: "Vue Welcome directory deleted. Type-check and tests pass. Ready for commit."

---

## Task 8: Final verification

**Goal:** Run the full verification suite before the PR is opened.

- [ ] **Step 1: Run auto-fix**

Run:
```bash
pnpm --dir frontend fix
```

Expected: no changes, or only trivial formatting adjustments to the new files.

- [ ] **Step 2: Run the CI-equivalent check**

Run:
```bash
pnpm --dir frontend check
```

Expected: pass (ESLint + Biome + React i18n consistency).

- [ ] **Step 3: Run type-check**

Run:
```bash
pnpm --dir frontend type-check
```

Expected: pass. This covers both the Vue tsconfig (via `vue-tsc`) and the React tsconfig (`tsconfig.react.json`).

- [ ] **Step 4: Run the full test suite**

Run:
```bash
pnpm --dir frontend test --run
```

Expected: all tests pass — existing + the 3 new test files (`BytebaseLogo.test.tsx`, `WelcomeButton.test.tsx`, `Welcome.test.tsx`).

- [ ] **Step 5: Manual UX-parity screenshot capture**

Confirm screenshots from Task 6 Step 5 cover these five states:

1. Workspace with custom logo.
2. Workspace with fallback Bytebase SVG.
3. User missing `bb.instances.create` → only Connect button.
4. User missing `bb.sql.select` → only Add-Instance button.
5. User with both permissions → both buttons.

Each screenshot should have a corresponding side-by-side with the pre-change Vue version.

- [ ] **Step 6: Stop for user review**

Final report template:

```
Stage 1 complete.

Summary:
- Added 2 i18n keys to 5 React locales
- Created: BytebaseLogo.tsx, WelcomeButton.tsx, Welcome.tsx (+ 3 test files)
- Modified: mount.ts (1 new glob), StandardPanel.vue (import + destructure + changeConnection)
- Deleted: frontend/src/views/sql-editor/EditorPanel/Welcome/ (3 files)

Verification:
- pnpm fix: clean
- pnpm check: pass
- pnpm type-check: pass
- pnpm test: all pass (N tests, M new)
- Manual UX parity: 5 states verified with screenshots (attached)

Ready for PR.
```
