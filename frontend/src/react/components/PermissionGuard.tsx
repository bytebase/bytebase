import { type ReactNode, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useComponentPermissionState } from "@/react/components/ComponentPermissionGuard";
import { useAppStore } from "@/react/stores/app";
import type { Permission } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { BlockTooltip, Tooltip } from "./ui/tooltip";

/**
 * Fire-and-forget hook that kicks off the workspace + (optional) project
 * IAM policy fetches so the Zustand caches read by
 * `useComponentPermissionState` are populated.
 *
 * Defensive: every current `PermissionGuard` caller mounts inside a route
 * shell that already triggers these loaders (`BannersWrapper` →
 * `useWorkspacePermission`, `ProjectRouteShell` / `SQLEditorRouteShell` →
 * `usePermissionDataReady`). Triggering again is a cache hit — both
 * loaders dedupe via the in-flight `*Request` promise or the populated
 * cache. The point of doing it here is to keep `PermissionGuard` working
 * if a future caller mounts it outside such a shell (a modal, a
 * standalone page, a reused widget) so it doesn't sit forever
 * `disabled={true}` against an empty cache.
 *
 * Intentionally does NOT gate `disabled` on a "ready" flag — see the
 * earlier `usePermissionDataReady` integration we removed: gating
 * caused `disabled` to flap on parent re-renders, which both flickered
 * the Tooltip wrapper and silently dropped clicks landing during the
 * load window.
 */
function useTriggerPermissionLoad(project?: Project) {
  const loadWorkspacePermissionState = useAppStore(
    (state) => state.loadWorkspacePermissionState
  );
  const loadProjectIamPolicy = useAppStore(
    (state) => state.loadProjectIamPolicy
  );
  const projectName = project?.name;
  useEffect(() => {
    void loadWorkspacePermissionState();
    if (projectName) void loadProjectIamPolicy(projectName);
  }, [loadWorkspacePermissionState, loadProjectIamPolicy, projectName]);
}

/**
 * usePermissionCheck returns whether the user has all the required permissions
 * and a tooltip message listing missing ones.
 */
export function usePermissionCheck(
  permissions: Permission[],
  project?: Project
): [boolean, string | undefined] {
  const { t } = useTranslation();
  useTriggerPermissionLoad(project);
  const { missedPermissions } = useComponentPermissionState({
    permissions,
    project,
  });
  if (missedPermissions.length === 0) return [true, undefined];
  return [
    false,
    project
      ? t("common.missing-required-permission-for-resource", {
          resource: project.name,
        })
      : t("common.missing-required-permission", {
          permissions: missedPermissions.join(", "),
        }),
  ];
}

interface PermissionGuardRenderProps {
  disabled: boolean;
}

interface PermissionGuardProps {
  readonly permissions: Permission[];
  readonly project?: Project;
  /** Either a ReactNode or a render function receiving `{ disabled }`. */
  readonly children:
    | ReactNode
    | ((props: PermissionGuardRenderProps) => ReactNode);
  /** Use "block" when wrapping section-level block content (e.g. form areas). */
  readonly display?: "inline" | "block";
}

/**
 * PermissionGuard wraps content with a styled tooltip showing missing
 * permissions. The tooltip only renders when the user lacks permission.
 *
 * Supports two patterns:
 *
 * 1. Static children (use `usePermissionCheck` separately for disabled state):
 * ```tsx
 * const [canEdit] = usePermissionCheck(["bb.settings.setWorkspaceProfile"]);
 * <PermissionGuard permissions={["bb.settings.setWorkspaceProfile"]}>
 *   <Button disabled={!canEdit}>Edit</Button>
 * </PermissionGuard>
 * ```
 *
 * 2. Render-prop children (like Vue PermissionGuardWrapper slot props):
 * ```tsx
 * <PermissionGuard permissions={["bb.projects.update"]} project={project}>
 *   {({ disabled }) => <Button disabled={disabled}>Save</Button>}
 * </PermissionGuard>
 * ```
 */
export function PermissionGuard({
  permissions,
  project,
  children,
  display = "inline",
}: PermissionGuardProps) {
  const { t } = useTranslation();
  useTriggerPermissionLoad(project);
  const { missedPermissions } = useComponentPermissionState({
    permissions,
    project,
  });

  const disabled = missedPermissions.length > 0;

  const tooltip = disabled ? (
    <div className="flex flex-col gap-1">
      {project
        ? t("common.missing-required-permission-for-resource", {
            resource: project.name,
          })
        : t("common.missing-required-permission", { permissions: "" })}
      <ul className="list-disc pl-4">
        {missedPermissions.map((p) => (
          <li key={p}>{p}</li>
        ))}
      </ul>
    </div>
  ) : undefined;

  const content =
    typeof children === "function" ? children({ disabled }) : children;

  if (display === "block") {
    return <BlockTooltip content={tooltip}>{content}</BlockTooltip>;
  }
  return <Tooltip content={tooltip}>{content}</Tooltip>;
}
