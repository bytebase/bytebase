import { ArrowDown, ArrowUp, Plus, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { RoleSelect } from "@/react/components/RoleSelect";
import { PresetRoleType } from "@/types";
import { displayRoleTitle } from "@/utils";
import { Button } from "../ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../ui/table";

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
      <Table className="border border-control-border">
        <TableHeader>
          <TableRow className="bg-control-bg">
            <TableHead className="w-20 text-center text-control">
              {t("custom-approval.approval-flow.node.order")}
            </TableHead>
            <TableHead className="text-control">
              {t("custom-approval.approval-flow.node.approver")}
            </TableHead>
            {editable && (
              <TableHead className="text-control">
                {t("common.operations")}
              </TableHead>
            )}
          </TableRow>
        </TableHeader>
        <TableBody>
          {roles.map((role, index) => (
            <TableRow key={index}>
              <TableCell className="text-center">{index + 1}</TableCell>
              <TableCell>
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
              </TableCell>
              {editable && (
                <TableCell>
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
                </TableCell>
              )}
            </TableRow>
          ))}
        </TableBody>
      </Table>
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
