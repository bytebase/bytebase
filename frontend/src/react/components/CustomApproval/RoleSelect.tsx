import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import { useVueState } from "@/react/hooks/useVueState";
import { useRoleStore } from "@/store";
import {
  PRESET_PROJECT_ROLES,
  PRESET_ROLES,
  PRESET_WORKSPACE_ROLES,
} from "@/types/iam";
import { displayRoleTitle } from "@/utils";

interface RoleSelectProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}

export function RoleSelect({ value, onChange, disabled }: RoleSelectProps) {
  const allRoles = useVueState(() => useRoleStore().roleList);

  const customRoles = allRoles
    .filter((r) => !PRESET_ROLES.includes(r.name))
    .map((r) => r.name);

  const roles = [
    ...PRESET_WORKSPACE_ROLES,
    ...PRESET_PROJECT_ROLES,
    ...customRoles,
  ];

  return (
    <Select
      value={value}
      onValueChange={(v) => v !== null && onChange(v)}
      disabled={disabled}
    >
      <SelectTrigger className="w-48">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {roles.map((role) => (
          <SelectItem key={role} value={role}>
            {displayRoleTitle(role)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
