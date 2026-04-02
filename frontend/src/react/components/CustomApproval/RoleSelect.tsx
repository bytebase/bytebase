import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import { useVueState } from "@/react/hooks/useVueState";
import { useRoleStore } from "@/store";
import { PRESET_WORKSPACE_ROLES } from "@/types/iam";
import { displayRoleTitle } from "@/utils";

interface RoleSelectProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}

export function RoleSelect({ value, onChange, disabled }: RoleSelectProps) {
  const customRoles = useVueState(() => useRoleStore().roleList);

  const roles = [...PRESET_WORKSPACE_ROLES, ...customRoles.map((r) => r.name)];

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
