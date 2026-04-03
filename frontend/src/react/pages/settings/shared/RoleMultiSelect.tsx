import { Check, ChevronDown, X } from "lucide-react";
import { useCallback, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { useClickOutside } from "@/react/hooks/useClickOutside";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useRoleStore, useSubscriptionV1Store } from "@/store";
import {
  PRESET_PROJECT_ROLES,
  PRESET_ROLES,
  PRESET_WORKSPACE_ROLES,
} from "@/types/iam";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { displayRoleDescription, displayRoleTitle } from "@/utils";

export function RoleMultiSelect({
  value,
  onChange,
  disabled,
}: {
  value: string[];
  onChange: (roles: string[]) => void;
  disabled?: boolean;
}) {
  const { t } = useTranslation();
  const roleStore = useRoleStore();
  const subscriptionStore = useSubscriptionV1Store();
  const roleList = useVueState(() => [...roleStore.roleList]);
  const hasCustomRoleFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(PlanFeature.FEATURE_CUSTOM_ROLES)
  );
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleClickOutside = useCallback(() => {
    setOpen(false);
    setSearch("");
  }, []);
  useClickOutside(containerRef, open, handleClickOutside);

  const groups = useMemo(() => {
    const kw = search.toLowerCase();
    const matchRole = (name: string) =>
      !kw || displayRoleTitle(name).toLowerCase().includes(kw);

    const workspace = PRESET_WORKSPACE_ROLES.filter(matchRole);
    const project = PRESET_PROJECT_ROLES.filter(matchRole);
    const custom = roleList
      .filter((r) => !PRESET_ROLES.includes(r.name))
      .map((r) => r.name)
      .filter(matchRole);

    const result: { label: string; roles: string[] }[] = [];
    if (workspace.length > 0)
      result.push({
        label: t("role.workspace-roles.self"),
        roles: workspace,
      });
    if (project.length > 0)
      result.push({ label: t("role.project-roles.self"), roles: project });
    if (custom.length > 0)
      result.push({ label: t("role.custom-roles.self"), roles: custom });
    return result;
  }, [roleList, search, t]);

  const isCustomRole = (name: string) => !PRESET_ROLES.includes(name);

  const toggle = (roleName: string) => {
    if (disabled) return;
    // Block selecting custom roles without the plan feature
    if (isCustomRole(roleName) && !hasCustomRoleFeature) return;
    if (value.includes(roleName)) {
      onChange(value.filter((r) => r !== roleName));
    } else {
      onChange([...value, roleName]);
    }
  };

  const remove = (roleName: string) => {
    if (disabled) return;
    onChange(value.filter((r) => r !== roleName));
  };

  return (
    <div ref={containerRef} className="relative">
      {/* Trigger */}
      <div
        className={cn(
          "flex flex-wrap items-center gap-1 min-h-[2.25rem] w-full rounded-xs border border-control-border bg-transparent px-2 py-1 text-sm cursor-pointer",
          disabled && "opacity-50 cursor-not-allowed",
          open && "ring-2 ring-accent border-accent"
        )}
        onClick={() => {
          if (!disabled) {
            setOpen(!open);
            requestAnimationFrame(() => inputRef.current?.focus());
          }
        }}
      >
        {value.map((roleName) => (
          <span
            key={roleName}
            className="inline-flex items-center gap-x-1 rounded-xs bg-gray-100 px-1.5 py-0.5 text-xs"
          >
            {displayRoleTitle(roleName)}
            {!disabled && (
              <button
                type="button"
                className="hover:text-error"
                onClick={(e) => {
                  e.stopPropagation();
                  remove(roleName);
                }}
              >
                <X className="h-3 w-3" />
              </button>
            )}
          </span>
        ))}
        {open && (
          <input
            ref={inputRef}
            className="flex-1 min-w-[4rem] outline-hidden text-sm bg-transparent"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={
              value.length === 0
                ? t("settings.members.select-role", { count: 2 })
                : ""
            }
          />
        )}
        {!open && value.length === 0 && (
          <span className="text-control-placeholder">
            {t("settings.members.select-role", { count: 2 })}
          </span>
        )}
        <ChevronDown className="ml-auto h-4 w-4 shrink-0 text-control-light" />
      </div>

      {/* Dropdown */}
      {open && (
        <div className="absolute z-50 mt-1 w-full bg-white border border-control-border rounded-sm shadow-lg max-h-60 overflow-auto">
          {groups.length === 0 && (
            <div className="px-3 py-2 text-sm text-control-light">
              {t("common.no-data")}
            </div>
          )}
          {groups.map((group) => (
            <div key={group.label}>
              <div className="px-3 py-1.5 text-xs font-medium text-control-light uppercase tracking-wide bg-gray-50">
                {group.label}
              </div>
              {group.roles.map((roleName) => {
                const selected = value.includes(roleName);
                const isCustom = isCustomRole(roleName);
                const blocked = isCustom && !hasCustomRoleFeature;
                return (
                  <div
                    key={roleName}
                    className={cn(
                      "flex items-center gap-x-2 px-3 py-1.5 text-sm hover:bg-gray-50",
                      selected && "bg-accent/5",
                      blocked
                        ? "opacity-50 cursor-not-allowed"
                        : "cursor-pointer"
                    )}
                    onClick={() => toggle(roleName)}
                  >
                    <div
                      className={cn(
                        "h-4 w-4 rounded-xs border flex items-center justify-center shrink-0",
                        selected
                          ? "bg-accent border-accent text-white"
                          : "border-control-border"
                      )}
                    >
                      {selected && <Check className="h-3 w-3" />}
                    </div>
                    <div className="flex flex-col">
                      <div className="flex items-center gap-x-1">
                        <span>{displayRoleTitle(roleName)}</span>
                        {blocked && (
                          <FeatureBadge
                            feature={PlanFeature.FEATURE_CUSTOM_ROLES}
                            clickable={false}
                          />
                        )}
                      </div>
                      {displayRoleDescription(roleName) && (
                        <span className="text-xs text-control-light">
                          {displayRoleDescription(roleName)}
                        </span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
