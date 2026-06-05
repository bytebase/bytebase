# RouterLink Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a shared React `RouterLink` for in-app navigation and migrate repeated internal anchor patterns found in the React frontend.

**Architecture:** `RouterLink` is router-aware application infrastructure, not a shadcn-style visual primitive. It renders a real anchor with a resolved `href`, preserves native browser link behavior, and intercepts only normal same-tab left-click navigation for SPA routing through the existing `@/react/router`.

**Tech Stack:** React, TypeScript, Vitest, React DOM test utilities, existing Bytebase React router (`router`, `RouteTarget`), Tailwind utility classes supplied by callers.

---

## File Structure

- Create `frontend/src/react/components/RouterLink.tsx`
  - Owns internal route link behavior.
  - Accepts `to: RouteTarget`.
  - Extends normal anchor props except `href`.
  - Does not own visual variants or link colors.

- Create `frontend/src/react/components/RouterLink.test.tsx`
  - Unit tests for href resolution, click interception, and native-link escape hatches.

- Modify `frontend/src/react/pages/project/ProjectSettingsPage.tsx`
  - Replace the approval-flow tooltip `Button` with `RouterLink`.

- Modify `frontend/src/react/pages/project/ProjectDataExportPage.tsx`
  - Replace inline approval/settings route anchors in the deprecation banner.
  - Replace issue links only after checking row click behavior and nested anchors.

- Modify `frontend/src/react/components/header/HeaderBreadcrumb.tsx`
  - Replace workspace breadcrumb anchor.

- Modify `frontend/src/react/components/sql-editor/AsidePanel.tsx`
  - Replace projects link anchor.

- Modify `frontend/src/react/components/sql-review/ResourceLink.tsx`
  - Replace environment/project resource anchors.

- Modify `frontend/src/react/pages/settings/EnvironmentsPage.tsx`
  - Replace environment internal anchor, preserving `stopPropagation`.

- Modify `frontend/src/react/pages/project/ProjectMaskingExemptionPage.tsx`
  - Replace user profile internal anchor.

- Modify `frontend/src/react/components/revision/RevisionDetailPanel.tsx`
  - Replace task link anchor, preserving modifier-click behavior via `RouterLink`.

- Modify `frontend/src/react/pages/project/DatabaseChangelogDetailPage.tsx`
  - Replace task link anchor.

- Modify `frontend/src/react/components/ProjectSidebar.tsx`
  - Replace sidebar route anchors after preserving active classes and group toggle behavior.

- Modify `frontend/src/react/components/DashboardSidebar.tsx`
  - Replace dashboard sidebar route anchors after preserving active classes and group toggle behavior.

- Review but do not automatically migrate:
  - `frontend/src/react/components/IssueTable.tsx`
  - `frontend/src/react/pages/project/ProjectDataExportPage.tsx` issue-list row links
  - Href-only internal anchors in issue-detail pages and `GroupsPage.tsx`
  - Any `href="#"` controls, which may need buttons instead of links.

## Second-Pass Migration Targets

Follow-up exploration found more React internal navigation patterns after the
initial PR pass. Migrate the clearly navigational cases below; leave action
buttons, menu actions, and table-row click handlers alone unless the target is
already a real anchor.

- Auth route links currently rendered as `href="#"`.
- Settings and landing-page internal anchors resolved with `router.push` or
  `router.resolve`.
- Database/project internal anchors and link-styled spans.
- Internal links that intentionally open in a new tab, preserving `target`,
  `rel`, and propagation handlers.
- Pure navigation button CTAs where the button only routes and has no workflow
  side effects.

---

## Task 1: Add RouterLink Tests

**Files:**
- Create: `frontend/src/react/components/RouterLink.test.tsx`

- [ ] **Step 1: Write the failing test file**

```tsx
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { RouterLink } from "./RouterLink";

const pushMock = vi.fn();
const resolveMock = vi.fn((to: unknown) => {
  if (typeof to === "string") {
    return { href: to, fullPath: to };
  }
  return { href: "/setting/custom-approval", fullPath: "/setting/custom-approval" };
});

vi.mock("@/react/router", () => ({
  router: {
    push: pushMock,
    resolve: resolveMock,
  },
}));

describe("RouterLink", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    pushMock.mockClear();
    resolveMock.mockClear();
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
  });

  const render = (element: React.ReactElement) => {
    act(() => root.render(element));
    return container.querySelector("a") as HTMLAnchorElement;
  };

  test("renders the resolved href", () => {
    const to = { name: "workspace.custom-approval" };

    const link = render(<RouterLink to={to}>Approval</RouterLink>);

    expect(resolveMock).toHaveBeenCalledWith(to);
    expect(link.getAttribute("href")).toBe("/setting/custom-approval");
  });

  test("routes normal same-tab left clicks through the app router", () => {
    const to = { name: "workspace.custom-approval" };
    const link = render(<RouterLink to={to}>Approval</RouterLink>);

    const event = new MouseEvent("click", { bubbles: true, button: 0, cancelable: true });
    act(() => link.dispatchEvent(event));

    expect(event.defaultPrevented).toBe(true);
    expect(pushMock).toHaveBeenCalledWith(to);
  });

  test("does not intercept meta-click navigation", () => {
    const to = { name: "workspace.custom-approval" };
    const link = render(<RouterLink to={to}>Approval</RouterLink>);

    const event = new MouseEvent("click", {
      bubbles: true,
      button: 0,
      cancelable: true,
      metaKey: true,
    });
    act(() => link.dispatchEvent(event));

    expect(event.defaultPrevented).toBe(false);
    expect(pushMock).not.toHaveBeenCalled();
  });

  test("does not intercept middle-click navigation", () => {
    const to = { name: "workspace.custom-approval" };
    const link = render(<RouterLink to={to}>Approval</RouterLink>);

    const event = new MouseEvent("click", { bubbles: true, button: 1, cancelable: true });
    act(() => link.dispatchEvent(event));

    expect(event.defaultPrevented).toBe(false);
    expect(pushMock).not.toHaveBeenCalled();
  });

  test("does not intercept non-self targets", () => {
    const to = { name: "workspace.custom-approval" };
    const link = render(
      <RouterLink to={to} target="_blank">
        Approval
      </RouterLink>
    );

    const event = new MouseEvent("click", { bubbles: true, button: 0, cancelable: true });
    act(() => link.dispatchEvent(event));

    expect(event.defaultPrevented).toBe(false);
    expect(pushMock).not.toHaveBeenCalled();
  });

  test("runs onClick before routing and respects preventDefault", () => {
    const to = { name: "workspace.custom-approval" };
    const onClick = vi.fn((event: React.MouseEvent<HTMLAnchorElement>) => {
      event.preventDefault();
    });
    const link = render(
      <RouterLink to={to} onClick={onClick}>
        Approval
      </RouterLink>
    );

    const event = new MouseEvent("click", { bubbles: true, button: 0, cancelable: true });
    act(() => link.dispatchEvent(event));

    expect(onClick).toHaveBeenCalledOnce();
    expect(pushMock).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
pnpm --dir frontend vitest run src/react/components/RouterLink.test.tsx
```

Expected: FAIL because `./RouterLink` does not exist.

---

## Task 2: Implement RouterLink

**Files:**
- Create: `frontend/src/react/components/RouterLink.tsx`

- [ ] **Step 1: Implement the component**

```tsx
import type { AnchorHTMLAttributes, MouseEvent, ReactNode } from "react";
import { router, type RouteTarget } from "@/react/router";

type RouterLinkProps = Omit<
  AnchorHTMLAttributes<HTMLAnchorElement>,
  "href"
> & {
  readonly to: RouteTarget;
  readonly children: ReactNode;
};

function shouldHandleClick(
  event: MouseEvent<HTMLAnchorElement>,
  target: AnchorHTMLAttributes<HTMLAnchorElement>["target"],
  download: AnchorHTMLAttributes<HTMLAnchorElement>["download"]
): boolean {
  if (event.defaultPrevented) return false;
  if (event.button !== 0) return false;
  if (event.metaKey || event.altKey || event.ctrlKey || event.shiftKey) {
    return false;
  }
  if (target && target !== "_self") return false;
  if (download !== undefined) return false;
  return true;
}

export function RouterLink({
  to,
  target,
  download,
  onClick,
  children,
  ...props
}: RouterLinkProps) {
  const href = router.resolve(to).href;

  const handleClick = (event: MouseEvent<HTMLAnchorElement>) => {
    onClick?.(event);
    if (!shouldHandleClick(event, target, download)) {
      return;
    }
    event.preventDefault();
    router.push(to);
  };

  return (
    <a
      {...props}
      href={href}
      target={target}
      download={download}
      onClick={handleClick}
    >
      {children}
    </a>
  );
}
```

- [ ] **Step 2: Run the focused test**

Run:

```bash
pnpm --dir frontend vitest run src/react/components/RouterLink.test.tsx
```

Expected: PASS.

---

## Task 3: Migrate Approval Tooltip and Simple Inline Route Links

**Files:**
- Modify: `frontend/src/react/pages/project/ProjectSettingsPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectDataExportPage.tsx`
- Modify: `frontend/src/react/components/header/HeaderBreadcrumb.tsx`
- Modify: `frontend/src/react/components/sql-editor/AsidePanel.tsx`

- [ ] **Step 1: Replace the approval-flow tooltip link**

In `ProjectSettingsPage.tsx`, add:

```tsx
import { RouterLink } from "@/react/components/RouterLink";
```

Replace the tooltip `Button` link with:

```tsx
<RouterLink
  to={{ name: WORKSPACE_ROUTE_CUSTOM_APPROVAL }}
  className="text-accent underline text-left hover:text-accent-hover"
>
  {t("project.settings.issue-related.view-approval-flow")}
</RouterLink>
```

- [ ] **Step 2: Replace ProjectDataExportPage banner links**

Replace each Trans component route anchor shaped like this:

```tsx
<a
  className="text-accent underline hover:text-accent-hover"
  href={customApprovalHref}
  onClick={(e) => {
    e.preventDefault();
    router.push(customApprovalHref);
  }}
/>
```

with this:

```tsx
<RouterLink
  className="text-accent underline hover:text-accent-hover"
  to={{ name: WORKSPACE_ROUTE_CUSTOM_APPROVAL }}
/>
```

For the project settings link, use:

```tsx
<RouterLink
  className="text-accent underline hover:text-accent-hover"
  to={{
    name: PROJECT_V1_ROUTE_SETTINGS,
    params: { projectId },
  }}
/>
```

Then remove `projectSettingsHref` and `customApprovalHref` if they are no longer used.

- [ ] **Step 3: Replace HeaderBreadcrumb workspace link**

Replace:

```tsx
<a
  href={workspaceHref}
  className="inline-flex items-center gap-x-1.5 rounded-xs px-2 py-1 text-sm font-medium text-control hover:bg-control-bg cursor-pointer no-underline"
  onClick={(e) => {
    if (!e.metaKey && !e.ctrlKey) {
      e.preventDefault();
      void navigate.push({ name: WORKSPACE_ROUTE_LANDING });
    }
  }}
>
```

with:

```tsx
<RouterLink
  to={{ name: WORKSPACE_ROUTE_LANDING }}
  className="inline-flex items-center gap-x-1.5 rounded-xs px-2 py-1 text-sm font-medium text-control hover:bg-control-bg cursor-pointer no-underline"
>
```

Remove `workspaceHref` if unused.

- [ ] **Step 4: Replace AsidePanel projects link**

Replace the projects anchor with:

```tsx
<RouterLink
  to={{
    name: PROJECT_V1_ROUTE_DASHBOARD,
    params: { projectId },
  }}
  className="text-accent hover:underline"
>
```

If the existing route target differs, preserve the exact existing target object and pass it to `to`.

- [ ] **Step 5: Run focused tests for touched areas when available**

Run:

```bash
pnpm --dir frontend vitest run src/react/components/RouterLink.test.tsx src/react/components/header/DashboardHeader.test.tsx src/react/components/sql-editor/AsidePanel.test.tsx
```

Expected: PASS, or report if one of the named test files does not exist and continue with full frontend tests in Task 8.

---

## Task 4: Migrate Raw Internal Resource Links

**Files:**
- Modify: `frontend/src/react/components/sql-review/ResourceLink.tsx`
- Modify: `frontend/src/react/pages/settings/EnvironmentsPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectMaskingExemptionPage.tsx`
- Modify: `frontend/src/react/components/revision/RevisionDetailPanel.tsx`
- Modify: `frontend/src/react/pages/project/DatabaseChangelogDetailPage.tsx`

- [ ] **Step 1: Replace SQL review resource links**

In `ResourceLink.tsx`, replace:

```tsx
<a
  href={`/${resource}`}
  className="inline-flex items-center gap-x-1 normal-link"
  onClick={(e) => {
    e.preventDefault();
    router.push({ path: `/${resource}` });
  }}
>
```

with:

```tsx
<RouterLink
  to={{ path: `/${resource}` }}
  className="inline-flex items-center gap-x-1 normal-link"
>
```

For the environment link, preserve its existing class:

```tsx
<RouterLink
  to={{ path: `/${resource}` }}
  className="inline-flex items-center gap-x-1"
>
```

Remove the `router` import if unused.

- [ ] **Step 2: Replace EnvironmentsPage internal environment link**

Preserve the existing `stopPropagation()` behavior by passing an `onClick` that does not call `preventDefault`:

```tsx
<RouterLink
  to={{ path: `/${formatEnvironmentName(environment.id)}` }}
  className="hover:underline"
  onClick={(e) => {
    e.stopPropagation();
  }}
>
```

- [ ] **Step 3: Replace ProjectMaskingExemptionPage user link**

Replace:

```tsx
<a
  href={`/users/${userEmail}`}
  className="normal-link font-medium"
  onClick={(e) => {
    e.preventDefault();
    router.push(`/users/${userEmail}`);
  }}
>
```

with:

```tsx
<RouterLink
  to={`/users/${userEmail}`}
  className="normal-link font-medium"
>
```

- [ ] **Step 4: Replace task links in revision and changelog detail pages**

For `taskFullLink`, replace anchors with:

```tsx
<RouterLink
  to={{ path: taskFullLink }}
  className="normal-link"
>
```

Use the exact existing class names and children from each file. Remove local modifier-click guards because `RouterLink` owns that behavior.

- [ ] **Step 5: Run focused tests**

Run:

```bash
pnpm --dir frontend vitest run src/react/components/RouterLink.test.tsx src/react/pages/project/DatabaseChangelogDetailPage.test.tsx
```

Expected: PASS.

---

## Task 5: Migrate Sidebar Navigation Links

**Files:**
- Modify: `frontend/src/react/components/ProjectSidebar.tsx`
- Modify: `frontend/src/react/components/DashboardSidebar.tsx`
- Test: `frontend/src/react/components/DashboardSidebar.test.tsx`

- [ ] **Step 1: Replace ProjectSidebar child and leaf anchors**

Replace child links:

```tsx
<a
  key={`${parentIndex}-${j}`}
  href={resolveHref(child.path)}
  className={`${childRouteClass} cursor-pointer no-underline text-inherit ${classes.join(" ")}`}
  onClick={(e) => handleItemClick(e, child.path!)}
>
```

with:

```tsx
<RouterLink
  key={`${parentIndex}-${j}`}
  to={{
    name: child.path,
    params: { projectId },
  }}
  className={`${childRouteClass} cursor-pointer no-underline text-inherit ${classes.join(" ")}`}
>
```

Replace leaf links the same way with `item.path`.

- [ ] **Step 2: Replace ProjectSidebar home link**

Use the existing resolved home route target if available. If only `homeHref` exists, introduce a `homeRoute` value:

```tsx
const homeRoute = resolveHomeRouteTarget();
```

Then render:

```tsx
<RouterLink to={homeRoute} className="p-2 shrink-0 m-auto cursor-pointer">
```

If the file only has `resolveHomeRoute().fullPath`, use:

```tsx
<RouterLink to={homeHref} className="p-2 shrink-0 m-auto cursor-pointer">
```

- [ ] **Step 3: Remove obsolete click helpers**

Remove `handleItemClick`, `handleHomeClick`, and `resolveHref` only if no longer used by group toggles or other code.

- [ ] **Step 4: Replace DashboardSidebar anchors**

Apply the same pattern to DashboardSidebar route anchors:

```tsx
<RouterLink
  to={{ name: child.name }}
  className={existingClassName}
>
```

Preserve active route class calculation and all icon/text children.

- [ ] **Step 5: Run sidebar tests**

Run:

```bash
pnpm --dir frontend vitest run src/react/components/DashboardSidebar.test.tsx
```

Expected: PASS.

---

## Task 6: Review Table and Row-Link Candidates Before Migrating

**Files:**
- Review: `frontend/src/react/components/IssueTable.tsx`
- Review: `frontend/src/react/pages/project/ProjectDataExportPage.tsx`
- Review: `frontend/src/react/components/ProjectTable.tsx`

- [ ] **Step 1: Identify nested anchor and row-click behavior**

For each table, list whether it has:

```text
- row-level onClick navigation
- nested anchor links inside that row
- stopPropagation calls
- href="#" action controls
```

- [ ] **Step 2: Migrate only safe nested anchors**

Only replace anchors that already have a real internal `href` and route to the same destination as the anchor:

```tsx
<RouterLink to={issueUrl} className="font-medium text-main text-base hover:underline min-w-0 block">
```

Do not migrate `href="#"` controls in this task.

- [ ] **Step 3: Convert action-only href="#" controls separately**

For action controls that do not represent navigation, use `Button` or a real `<button>` according to surrounding patterns. Do not use `RouterLink`.

- [ ] **Step 4: Run table tests**

Run:

```bash
pnpm --dir frontend vitest run src/react/components/IssueTable.test.tsx src/react/components/ProjectTable.test.tsx
```

Expected: PASS.

---

## Task 7: Review Href-Only Internal Anchors

**Files:**
- Review: `frontend/src/react/pages/project/issue-detail/components/IssueDetailCommentList.tsx`
- Review: `frontend/src/react/pages/project/issue-detail/components/IssueDetailDatabaseChangeView.tsx`
- Review: `frontend/src/react/pages/project/issue-detail/components/IssueDetailDatabaseCreateView.tsx`
- Review: `frontend/src/react/pages/project/issue-detail/components/IssueDetailLabels.tsx`
- Review: `frontend/src/react/pages/settings/GroupsPage.tsx`

- [ ] **Step 1: Check whether SPA interception is desirable**

For each href-only internal anchor, confirm it is not intentionally doing a full-page load.

- [ ] **Step 2: Migrate only same-tab app navigation**

Replace safe anchors with:

```tsx
<RouterLink to={existingRouteTargetOrPath} className={existingClassName}>
  {existingChildren}
</RouterLink>
```

Do not migrate anchors with:

```tsx
target="_blank"
download
href starting with "http"
href starting with "mailto:"
```

- [ ] **Step 3: Run focused issue-detail tests**

Run:

```bash
pnpm --dir frontend vitest run src/react/pages/project/plan-detail/components/PlanDetailApprovalFlow.test.tsx src/react/pages/project/ProjectIssueDetailPage.test.tsx
```

Expected: PASS, or report unavailable focused coverage and rely on full tests in Task 8.

---

## Task 8: Full Frontend Verification

**Files:**
- Verify all modified frontend files.

- [ ] **Step 1: Run frontend autofix**

Run:

```bash
pnpm --dir frontend fix
```

Expected: exit 0.

- [ ] **Step 2: Run frontend check**

Run:

```bash
pnpm --dir frontend check
```

Expected: exit 0.

- [ ] **Step 3: Run frontend type-check**

Run:

```bash
pnpm --dir frontend type-check
```

Expected: exit 0.

- [ ] **Step 4: Run frontend tests**

Run:

```bash
pnpm --dir frontend test
```

Expected: exit 0.

---

## Self-Review

- Spec coverage: The plan adds `RouterLink`, migrates the approval tooltip, and includes all strong candidates reported by the subagents: project/sidebar/dashboard/header/data-export/resource/environment/masking/revision/changelog links.
- Guardrails: External links, `mailto:`, `download`, `target="_blank"`, and `href="#"` action controls are explicitly excluded from automatic RouterLink migration.
- Type consistency: The component uses the existing `RouteTarget` type and `router.resolve`/`router.push` APIs from `@/react/router`.
