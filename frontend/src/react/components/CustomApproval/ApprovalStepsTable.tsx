import { ArrowDown, ArrowUp, Plus, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { RoleSelect } from "@/react/pages/settings/shared/RoleSelect";
import { PresetRoleType } from "@/types";
import { displayRoleTitle } from "@/utils";
import { Button } from "../ui/button";

interface ApprovalStepsTableProps {
  roles: string[];
  editable: boolean;
  allowAdmin: boolean;
  onRolesChange: (roles: string[]) => void;
}

export function ApprovalStepsTable({
  roles,
  editable,
  allowAdmin,
  onRolesChange,
}: ApprovalStepsTableProps) {
  const { t } = useTranslation();

  const reorder = (index: number, offset: -1 | 1) => {
    const target = index + offset;
    if (target < 0 || target >= roles.length) return;
    const newRoles = [...roles];
    newRoles[index] = roles[target];
    newRoles[target] = roles[index];
    onRolesChange(newRoles);
  };

  const addStep = () => {
    onRolesChange([...roles, PresetRoleType.WORKSPACE_ADMIN]);
  };

  const removeStep = (index: number) => {
    onRolesChange(roles.filter((_, i) => i !== index));
  };

  return (
    <div>
      <table className="w-full border-collapse border border-control-border text-sm">
        <thead>
          <tr className="bg-control-bg">
            <th className="w-20 border border-control-border px-3 py-2 text-center font-medium text-control">
              {t("custom-approval.approval-flow.node.order")}
            </th>
            <th className="border border-control-border px-3 py-2 text-left font-medium text-control">
              {t("custom-approval.approval-flow.node.approver")}
            </th>
            {editable && (
              <th className="border border-control-border px-3 py-2 text-left font-medium text-control">
                {t("common.operations")}
              </th>
            )}
          </tr>
        </thead>
        <tbody>
          {roles.map((role, index) => (
            <tr
              key={index}
              className={index % 2 === 1 ? "bg-control-bg/50" : "bg-white"}
            >
              <td className="border border-control-border px-3 py-2 text-center text-control">
                {index + 1}
              </td>
              <td className="border border-control-border px-3 py-2">
                {editable ? (
                  <RoleSelect
                    value={role ? [role] : []}
                    onChange={(vals) => {
                      const newRoles = [...roles];
                      newRoles[index] = vals[0] ?? "";
                      onRolesChange(newRoles);
                    }}
                    multiple={false}
                  />
                ) : (
                  <span className="text-control">{displayRoleTitle(role)}</span>
                )}
              </td>
              {editable && (
                <td className="border border-control-border px-3 py-2">
                  <div className="flex gap-x-1">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={index === 0 || !allowAdmin}
                      onClick={() => reorder(index, -1)}
                    >
                      <ArrowUp className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={index === roles.length - 1 || !allowAdmin}
                      onClick={() => reorder(index, 1)}
                    >
                      <ArrowDown className="h-4 w-4" />
                    </Button>
                    {allowAdmin && (
                      <Button
                        variant="outline"
                        size="sm"
                        title={t("custom-approval.approval-flow.node.delete")}
                        onClick={() => removeStep(index)}
                      >
                        <Trash2 className="h-3 w-3" />
                      </Button>
                    )}
                  </div>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
      {editable && allowAdmin && (
        <div className="mt-4">
          <Button variant="outline" onClick={addStep}>
            <Plus className="h-4 w-4" />
            {t("custom-approval.approval-flow.node.add")}
          </Button>
        </div>
      )}
    </div>
  );
}
