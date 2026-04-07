import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { Combobox, type ComboboxGroup } from "@/react/components/ui/combobox";
import { useVueState } from "@/react/hooks/useVueState";
import { useRoleStore, useSubscriptionV1Store } from "@/store";
import {
  PRESET_PROJECT_ROLES,
  PRESET_ROLES,
  PRESET_WORKSPACE_ROLES,
} from "@/types/iam";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { displayRoleDescription, displayRoleTitle } from "@/utils";

interface RoleSelectProps {
  value: string[];
  onChange: (roles: string[]) => void;
  disabled?: boolean;
  scope?: "project";
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

  const groups: ComboboxGroup[] = useMemo(() => {
    const isCustomRole = (name: string) => !PRESET_ROLES.includes(name);

    const workspace = PRESET_WORKSPACE_ROLES.map((name) => ({
      value: name,
      label: displayRoleTitle(name),
      description: displayRoleDescription(name),
    }));

    const project = PRESET_PROJECT_ROLES.map((name) => ({
      value: name,
      label: displayRoleTitle(name),
      description: displayRoleDescription(name),
    }));

    const custom = roleList
      .filter((r) => !PRESET_ROLES.includes(r.name))
      .map((r) => ({
        value: r.name,
        label: displayRoleTitle(r.name),
        description: displayRoleDescription(r.name),
        disabled: !hasCustomRoleFeature,
        render: () => (
          <div className="flex items-center gap-x-1">
            <span>{displayRoleTitle(r.name)}</span>
            {isCustomRole(r.name) && !hasCustomRoleFeature && (
              <FeatureBadge
                feature={PlanFeature.FEATURE_CUSTOM_ROLES}
                clickable={false}
              />
            )}
          </div>
        ),
      }));

    const result: ComboboxGroup[] = [];
    if (scope !== "project" && workspace.length > 0)
      result.push({
        label: t("role.workspace-roles.self"),
        options: workspace,
      });
    if (project.length > 0)
      result.push({ label: t("role.project-roles.self"), options: project });
    if (custom.length > 0)
      result.push({ label: t("role.custom-roles.self"), options: custom });
    return result;
  }, [roleList, hasCustomRoleFeature, scope, t]);

  const defaultPlaceholder = multiple
    ? t("settings.members.select-role", { count: 2 })
    : t("settings.members.assign-role");

  if (multiple) {
    return (
      <Combobox
        multiple
        value={value}
        onChange={onChange}
        options={groups}
        placeholder={placeholder || defaultPlaceholder}
        disabled={disabled}
        portal
      />
    );
  }

  return (
    <Combobox
      value={value[0] ?? ""}
      onChange={(v) => onChange(v ? [v] : [])}
      options={groups}
      placeholder={placeholder || defaultPlaceholder}
      disabled={disabled}
      portal
    />
  );
}
