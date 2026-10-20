import { displayRoleDescriptionFromList } from "@/lib/role";
import type { Role } from "@/types/proto-es/v1/role_service_pb";

/** The selected role's description, under a role select or a role title. */
export function RoleDescription({
  role,
  roleList,
}: {
  role: string;
  roleList: Role[];
}) {
  const description = role
    ? displayRoleDescriptionFromList(role, roleList)
    : undefined;
  if (!description) {
    return null;
  }
  return <p className="text-xs leading-4 text-control-light">{description}</p>;
}
