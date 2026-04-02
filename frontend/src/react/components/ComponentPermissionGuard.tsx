import { ShieldAlert } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import type { Permission } from "@/types";
import { hasWorkspacePermissionV2 } from "@/utils";

interface ComponentPermissionGuardProps {
  readonly permissions: Permission[];
  readonly children: ReactNode;
  readonly className?: string;
}

/**
 * ComponentPermissionGuard gates an entire component behind a permission check.
 *
 * - If the user has all required permissions, children are rendered normally.
 * - If the user is missing permissions, an error alert is shown listing the
 *   missing permissions — matching the Vue `ComponentPermissionGuard` behavior.
 */
export function ComponentPermissionGuard({
  permissions,
  children,
  className,
}: ComponentPermissionGuardProps) {
  const { t } = useTranslation();
  const missed = permissions.filter((p) => !hasWorkspacePermissionV2(p));

  if (missed.length === 0) {
    return <>{children}</>;
  }

  return (
    <div className={className}>
      <div
        role="alert"
        className="relative w-full rounded-xs border border-error/30 bg-error/5 text-error px-4 py-3 text-sm flex gap-x-3"
      >
        <ShieldAlert className="h-5 w-5 shrink-0 mt-0.5" />
        <div className="flex flex-col gap-2">
          <h5 className="font-medium leading-tight">
            {t("common.missing-required-permission", { permissions: "" })}
          </h5>
          <div>
            {t("common.required-permission")}
            <ul className="list-disc pl-4">
              {missed.map((p) => (
                <li key={p}>{p}</li>
              ))}
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}
