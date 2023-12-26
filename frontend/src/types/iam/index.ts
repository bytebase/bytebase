import IAM_ACL_DATA from "./acl.yaml";

interface Role {
  name: string;
  permissions: string[];
}

export const SYSTEM_ROLES: Role[] = IAM_ACL_DATA.roles;
