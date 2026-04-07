import { Check, ChevronDown, Search, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/react/components/FeatureBadge";
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

interface RoleSelectProps {
  /** Selected role(s). For single mode pass a single-element array. */
  value: string[];
  onChange: (roles: string[]) => void;
  disabled?: boolean;
  /** Restrict to project roles only. */
  scope?: "project";
  /** Single-select mode: selecting a role replaces the previous selection and closes the dropdown. */
  multiple?: boolean;
  placeholder?: string;
}

export function RoleSelect({
  value,
  onChange,
  disabled,
  scope,
  multiple = true,
  placeholder,
}: RoleSelectProps) {
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
  const dropdownRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [dropdownStyle, setDropdownStyle] = useState<React.CSSProperties>({});

  const defaultPlaceholder = multiple
    ? t("settings.members.select-role", { count: 2 })
    : t("settings.members.assign-role");

  // Calculate dropdown position relative to viewport (for portal rendering)
  useEffect(() => {
    if (!open || !containerRef.current) return;
    const rect = containerRef.current.getBoundingClientRect();
    setDropdownStyle({
      position: "fixed",
      top: rect.bottom + 4,
      left: rect.left,
      width: rect.width,
    });
  }, [open]);

  // Click outside: close if click is outside both trigger and portal dropdown
  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      const target = e.target as Node;
      if (containerRef.current?.contains(target)) return;
      if (dropdownRef.current?.contains(target)) return;
      setOpen(false);
      setSearch("");
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

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
    if (scope !== "project" && workspace.length > 0)
      result.push({
        label: t("role.workspace-roles.self"),
        roles: workspace,
      });
    if (project.length > 0)
      result.push({ label: t("role.project-roles.self"), roles: project });
    if (custom.length > 0)
      result.push({ label: t("role.custom-roles.self"), roles: custom });
    return result;
  }, [roleList, search, scope, t]);

  const isCustomRole = (name: string) => !PRESET_ROLES.includes(name);

  const selectRole = (roleName: string) => {
    if (disabled) return;
    if (isCustomRole(roleName) && !hasCustomRoleFeature) return;

    if (multiple) {
      if (value.includes(roleName)) {
        onChange(value.filter((r) => r !== roleName));
      } else {
        onChange([...value, roleName]);
      }
    } else {
      onChange([roleName]);
      setOpen(false);
      setSearch("");
    }
  };

  const remove = (roleName: string) => {
    if (disabled) return;
    onChange(value.filter((r) => r !== roleName));
  };

  const displayValue = multiple
    ? null
    : value.length > 0
      ? displayRoleTitle(value[0])
      : null;

  return (
    <div ref={containerRef} className="relative">
      {/* Trigger */}
      <div
        className={cn(
          "flex flex-wrap items-center gap-1 min-h-[2.25rem] w-full rounded-xs border border-control-border bg-transparent px-2 py-1 text-sm cursor-pointer",
          disabled && "opacity-50 cursor-not-allowed",
          open && "border-accent"
        )}
        onClick={() => {
          if (!disabled) {
            setOpen(!open);
            requestAnimationFrame(() => inputRef.current?.focus());
          }
        }}
      >
        {/* Multi: show chips */}
        {multiple &&
          value.map((roleName) => (
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
        {/* Single: show selected label */}
        {!multiple && (
          <span
            className={cn(
              "flex-1",
              !displayValue && "text-control-placeholder"
            )}
          >
            {displayValue || placeholder || defaultPlaceholder}
          </span>
        )}
        {/* Multi placeholder */}
        {multiple && value.length === 0 && (
          <span className="text-control-placeholder">
            {placeholder || defaultPlaceholder}
          </span>
        )}
        <ChevronDown className="ml-auto h-4 w-4 shrink-0 text-control-light" />
      </div>

      {/* Dropdown (portal to escape overflow containers) */}
      {open &&
        createPortal(
          <div
            ref={dropdownRef}
            style={dropdownStyle}
            className="z-[999] bg-white border border-control-border rounded-sm shadow-lg max-h-72 flex flex-col"
          >
            {/* Search input inside dropdown */}
            <div className="flex items-center gap-x-2 px-3 py-2 border-b sticky top-0 bg-white shrink-0">
              <Search className="h-4 w-4 text-control-light shrink-0" />
              <input
                ref={inputRef}
                className="flex-1 outline-hidden border-none shadow-none text-sm bg-transparent"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={t("common.filter")}
              />
            </div>

            <div className="overflow-auto">
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
                        onClick={() => selectRole(roleName)}
                      >
                        <div
                          className={cn(
                            "h-4 w-4 border flex items-center justify-center shrink-0",
                            multiple ? "rounded-xs" : "rounded-full",
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
          </div>,
          document.body
        )}
    </div>
  );
}
