import { ArrowDown, ArrowUp, Plus, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { RoleSelect } from "@/components/RoleSelect";
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
      <div className="overflow-hidden rounded-sm border border-block-border">
        <Table>
          <TableHeader className="bg-control-bg">
            <TableRow>
              <TableHead className="w-20 text-center">
                {t("custom-approval.approval-flow.node.order")}
              </TableHead>
              <TableHead>
                {t("custom-approval.approval-flow.node.approver")}
              </TableHead>
              {editable && <TableHead>{t("common.operations")}</TableHead>}
            </TableRow>
          </TableHeader>
          <TableBody>
            {roles.map((role, index) => (
              <TableRow key={index}>
                <TableCell className="py-2 text-center">{index + 1}</TableCell>
                <TableCell className="py-2">
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
                    <span className="text-control">
                      {displayRoleTitle(role)}
                    </span>
                  )}
                </TableCell>
                {editable && (
                  <TableCell className="py-2">
                    <div className="flex gap-x-2">
                      <Button
                        appearance="outline"
                        size="sm"
                        disabled={index === 0 || !allowAdmin}
                        onClick={() => reorder(index, -1)}
                      >
                        <ArrowUp className="size-4" />
                      </Button>
                      <Button
                        appearance="outline"
                        size="sm"
                        disabled={index === roles.length - 1 || !allowAdmin}
                        onClick={() => reorder(index, 1)}
                      >
                        <ArrowDown className="size-4" />
                      </Button>
                      {allowAdmin && (
                        <Button
                          appearance="outline"
                          size="sm"
                          title={t("custom-approval.approval-flow.node.delete")}
                          onClick={() => removeStep(index)}
                        >
                          <Trash2 className="size-3" />
                        </Button>
                      )}
                    </div>
                  </TableCell>
                )}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      {editable && allowAdmin && (
        <div className="mt-4">
          <Button appearance="outline" onClick={addStep}>
            <Plus className="size-4" />
            {t("custom-approval.approval-flow.node.add")}
          </Button>
        </div>
      )}
    </div>
  );
}
