// @generated by protoc-gen-es v2.5.2
// @generated from file v1/instance_role_service.proto (package bytebase.v1, syntax proto3)
/* eslint-disable */

import type { GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Message } from "@bufbuild/protobuf";

/**
 * Describes the file v1/instance_role_service.proto.
 */
export declare const file_v1_instance_role_service: GenFile;

/**
 * @generated from message bytebase.v1.GetInstanceRoleRequest
 */
export declare type GetInstanceRoleRequest = Message<"bytebase.v1.GetInstanceRoleRequest"> & {
  /**
   * The name of the role to retrieve.
   * Format: instances/{instance}/roles/{role name}
   * The role name is the unique name for the role.
   *
   * @generated from field: string name = 1;
   */
  name: string;
};

/**
 * Describes the message bytebase.v1.GetInstanceRoleRequest.
 * Use `create(GetInstanceRoleRequestSchema)` to create a new message.
 */
export declare const GetInstanceRoleRequestSchema: GenMessage<GetInstanceRoleRequest>;

/**
 * @generated from message bytebase.v1.ListInstanceRolesRequest
 */
export declare type ListInstanceRolesRequest = Message<"bytebase.v1.ListInstanceRolesRequest"> & {
  /**
   * The parent, which owns this collection of roles.
   * Format: instances/{instance}
   *
   * @generated from field: string parent = 1;
   */
  parent: string;

  /**
   * Not used.
   * The maximum number of roles to return. The service may return fewer than
   * this value.
   * If unspecified, at most 10 roles will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   *
   * @generated from field: int32 page_size = 2;
   */
  pageSize: number;

  /**
   * Not used.
   * A page token, received from a previous `ListInstanceRoles` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListInstanceRoles` must match
   * the call that provided the page token.
   *
   * @generated from field: string page_token = 3;
   */
  pageToken: string;

  /**
   * Refresh will refresh and return the latest data.
   *
   * @generated from field: bool refresh = 4;
   */
  refresh: boolean;
};

/**
 * Describes the message bytebase.v1.ListInstanceRolesRequest.
 * Use `create(ListInstanceRolesRequestSchema)` to create a new message.
 */
export declare const ListInstanceRolesRequestSchema: GenMessage<ListInstanceRolesRequest>;

/**
 * @generated from message bytebase.v1.ListInstanceRolesResponse
 */
export declare type ListInstanceRolesResponse = Message<"bytebase.v1.ListInstanceRolesResponse"> & {
  /**
   * The roles from the specified request.
   *
   * @generated from field: repeated bytebase.v1.InstanceRole roles = 1;
   */
  roles: InstanceRole[];

  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   *
   * @generated from field: string next_page_token = 2;
   */
  nextPageToken: string;
};

/**
 * Describes the message bytebase.v1.ListInstanceRolesResponse.
 * Use `create(ListInstanceRolesResponseSchema)` to create a new message.
 */
export declare const ListInstanceRolesResponseSchema: GenMessage<ListInstanceRolesResponse>;

/**
 * InstanceRole is the API message for instance role.
 *
 * @generated from message bytebase.v1.InstanceRole
 */
export declare type InstanceRole = Message<"bytebase.v1.InstanceRole"> & {
  /**
   * The name of the role.
   * Format: instances/{instance}/roles/{role}
   * The role name is the unique name for the role.
   *
   * @generated from field: string name = 1;
   */
  name: string;

  /**
   * The role name. It's unique within the instance.
   *
   * @generated from field: string role_name = 2;
   */
  roleName: string;

  /**
   * The role password.
   *
   * @generated from field: optional string password = 3;
   */
  password?: string;

  /**
   * The connection count limit for this role.
   *
   * @generated from field: optional int32 connection_limit = 4;
   */
  connectionLimit?: number;

  /**
   * The expiration for the role's password.
   *
   * @generated from field: optional string valid_until = 5;
   */
  validUntil?: string;

  /**
   * The role attribute.
   * For PostgreSQL, it containt super_user, no_inherit, create_role, create_db, can_login, replication and bypass_rls. Docs: https://www.postgresql.org/docs/current/role-attributes.html
   * For MySQL, it's the global privileges as GRANT statements, which means it only contains "GRANT ... ON *.* TO ...". Docs: https://dev.mysql.com/doc/refman/8.0/en/grant.html
   *
   * @generated from field: optional string attribute = 6;
   */
  attribute?: string;
};

/**
 * Describes the message bytebase.v1.InstanceRole.
 * Use `create(InstanceRoleSchema)` to create a new message.
 */
export declare const InstanceRoleSchema: GenMessage<InstanceRole>;

/**
 * @generated from service bytebase.v1.InstanceRoleService
 */
export declare const InstanceRoleService: GenService<{
  /**
   * Permissions required: bb.instanceRoles.get
   *
   * @generated from rpc bytebase.v1.InstanceRoleService.GetInstanceRole
   */
  getInstanceRole: {
    methodKind: "unary";
    input: typeof GetInstanceRoleRequestSchema;
    output: typeof InstanceRoleSchema;
  },
  /**
   * Permissions required: bb.instanceRoles.list
   *
   * @generated from rpc bytebase.v1.InstanceRoleService.ListInstanceRoles
   */
  listInstanceRoles: {
    methodKind: "unary";
    input: typeof ListInstanceRolesRequestSchema;
    output: typeof ListInstanceRolesResponseSchema;
  },
}>;

