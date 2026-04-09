import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import type { Permission } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";
import { BlockTooltip, Tooltip } from "./ui/tooltip";

/**
 * usePermissionCheck returns whether the user has all the required permissions
 * and a tooltip message listing missing ones.
 */
export function usePermissionCheck(
  permissions: Permission[],
  project?: Project
): [boolean, string | undefined] {
  const { t } = useTranslation();
  const missed = project
    ? permissions.filter((p) => !hasProjectPermissionV2(project, p))
    : permissions.filter((p) => !hasWorkspacePermissionV2(p));
  if (missed.length === 0) return [true, undefined];
  return [
    false,
    project
      ? t("common.missing-required-permission-for-resource", {
          resource: project.name,
        })
      : t("common.missing-required-permission", {
          permissions: missed.join(", "),
        }),
  ];
}

interface PermissionGuardProps {
  readonly permissions: Permission[];
  readonly project?: Project;
  readonly children: ReactNode;
  /** Use "block" when wrapping section-level block content (e.g. form areas). */
  readonly display?: "inline" | "block";
}

/**
 * PermissionGuard wraps block content with a styled tooltip showing missing
 * permissions. The tooltip only renders when the user lacks permission.
 *
 * Usage:
 * ```tsx
 * const [canEdit] = usePermissionCheck(["bb.settings.setWorkspaceProfile"]);
 * <PermissionGuard permissions={["bb.settings.setWorkspaceProfile"]}>
 *   <div>
 *     <input disabled={!canEdit} />
 *   </div>
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
  const missed = project
    ? permissions.filter((p) => !hasProjectPermissionV2(project, p))
    : permissions.filter((p) => !hasWorkspacePermissionV2(p));

  const tooltip =
    missed.length > 0 ? (
      <div className="flex flex-col gap-1">
        {project
          ? t("common.missing-required-permission-for-resource", {
              resource: project.name,
            })
          : t("common.missing-required-permission", { permissions: "" })}
        <ul className="list-disc pl-4">
          {missed.map((p) => (
            <li key={p}>{p}</li>
          ))}
        </ul>
      </div>
    ) : undefined;

  if (display === "block") {
    return <BlockTooltip content={tooltip}>{children}</BlockTooltip>;
  }
  return <Tooltip content={tooltip}>{children}</Tooltip>;
}
