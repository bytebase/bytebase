import { useTranslation } from "react-i18next";
import type { InstanceRole } from "@/types/proto-es/v1/instance_role_service_pb";

interface InstanceRoleTableProps {
  instanceRoleList: InstanceRole[];
}

export function InstanceRoleTable({
  instanceRoleList,
}: InstanceRoleTableProps) {
  const { t } = useTranslation();

  return (
    <table className="w-full text-sm border border-block-border rounded-md">
      <thead>
        <tr className="bg-gray-50 border-b border-block-border">
          <th className="text-left px-4 py-2 w-[200px] font-medium">
            {t("common.user")}
          </th>
          <th className="text-left px-4 py-2 font-medium">
            {t("instance.grants")}
          </th>
        </tr>
      </thead>
      <tbody>
        {instanceRoleList.length === 0 ? (
          <tr>
            <td
              colSpan={2}
              className="px-4 py-8 text-center text-control-light"
            >
              {t("common.no-data")}
            </td>
          </tr>
        ) : (
          instanceRoleList.map((role, index) => (
            <tr
              key={role.roleName}
              className={index % 2 === 1 ? "bg-gray-50/50" : ""}
            >
              <td className="px-4 py-2 border-b border-block-border">
                {role.roleName}
              </td>
              <td className="px-4 py-2 border-b border-block-border whitespace-pre-wrap break-all">
                {(role.attribute ?? "").replaceAll("\n", "\n\n")}
              </td>
            </tr>
          ))
        )}
      </tbody>
    </table>
  );
}
