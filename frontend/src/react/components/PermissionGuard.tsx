import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import type { Permission } from "@/types";
import { hasWorkspacePermissionV2 } from "@/utils";

/**
 * usePermissionCheck returns whether the user has all the required permissions
 * and a title string for use as a native tooltip on disabled controls.
 *
 * Usage:
 * ```tsx
 * const [canEdit, permissionTitle] = usePermissionCheck(["bb.settings.setWorkspaceProfile"]);
 * <input disabled={!canEdit} title={permissionTitle} />
 * ```
 */
export function usePermissionCheck(
  permissions: Permission[]
): [boolean, string | undefined] {
  const { t } = useTranslation();
  const missed = permissions.filter((p) => !hasWorkspacePermissionV2(p));
  if (missed.length === 0) return [true, undefined];
  return [
    false,
    `${t("common.missing-required-permission")}: ${missed.join(", ")}`,
  ];
}

interface PermissionGuardProps {
  readonly permissions: Permission[];
  readonly children: ReactNode;
}

/**
 * PermissionGuard wraps a control with a tooltip that shows missing permissions.
 * The wrapped control must handle its own disabled state.
 *
 * Usage:
 * ```tsx
 * <PermissionGuard permissions={["bb.settings.setWorkspaceProfile"]}>
 *   <input disabled={!canEdit} />
 * </PermissionGuard>
 * ```
 */
export function PermissionGuard({
  permissions,
  children,
}: PermissionGuardProps) {
  const [hasPermission, tooltip] = usePermissionCheck(permissions);

  if (hasPermission) {
    return <>{children}</>;
  }

  return (
    <span className="relative group/perm" title={tooltip}>
      {children}
    </span>
  );
}
