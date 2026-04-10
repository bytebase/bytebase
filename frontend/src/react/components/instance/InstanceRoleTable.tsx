import { useTranslation } from "react-i18next";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import type { InstanceRole } from "@/types/proto-es/v1/instance_role_service_pb";

interface InstanceRoleTableProps {
  instanceRoleList: InstanceRole[];
}

export function InstanceRoleTable({
  instanceRoleList,
}: InstanceRoleTableProps) {
  const { t } = useTranslation();

  return (
    <Table className="border border-block-border rounded-sm">
      <TableHeader>
        <TableRow className="bg-control-bg">
          <TableHead className="w-[200px]">{t("common.user")}</TableHead>
          <TableHead>{t("instance.grants")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {instanceRoleList.length === 0 ? (
          <TableRow>
            <TableCell
              colSpan={2}
              className="py-8 text-center text-control-light"
            >
              {t("common.no-data")}
            </TableCell>
          </TableRow>
        ) : (
          instanceRoleList.map((role) => (
            <TableRow key={role.roleName}>
              <TableCell>{role.roleName}</TableCell>
              <TableCell className="whitespace-pre-wrap break-all">
                {(role.attribute ?? "").replaceAll("\n", "\n\n")}
              </TableCell>
            </TableRow>
          ))
        )}
      </TableBody>
    </Table>
  );
}
