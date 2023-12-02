/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { Timestamp } from "../google/protobuf/timestamp";
import { Engine, engineFromJSON, engineToJSON } from "./common";
import { DatabaseMetadata } from "./database_service";

export const protobufPackage = "bytebase.v1";

export enum BranchView {
  /**
   * BRANCH_VIEW_UNSPECIFIED - The default / unset value.
   * The API will default to the BASIC view.
   */
  BRANCH_VIEW_UNSPECIFIED = 0,
  /** BRANCH_VIEW_BASIC - Exclude schema, baseline_schema. */
  BRANCH_VIEW_BASIC = 1,
  /** BRANCH_VIEW_FULL - Include everything. */
  BRANCH_VIEW_FULL = 2,
  UNRECOGNIZED = -1,
}

export function branchViewFromJSON(object: any): BranchView {
  switch (object) {
    case 0:
    case "BRANCH_VIEW_UNSPECIFIED":
      return BranchView.BRANCH_VIEW_UNSPECIFIED;
    case 1:
    case "BRANCH_VIEW_BASIC":
      return BranchView.BRANCH_VIEW_BASIC;
    case 2:
    case "BRANCH_VIEW_FULL":
      return BranchView.BRANCH_VIEW_FULL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return BranchView.UNRECOGNIZED;
  }
}

export function branchViewToJSON(object: BranchView): string {
  switch (object) {
    case BranchView.BRANCH_VIEW_UNSPECIFIED:
      return "BRANCH_VIEW_UNSPECIFIED";
    case BranchView.BRANCH_VIEW_BASIC:
      return "BRANCH_VIEW_BASIC";
    case BranchView.BRANCH_VIEW_FULL:
      return "BRANCH_VIEW_FULL";
    case BranchView.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface Branch {
  /**
   * The name of the branch.
   * Format: projects/{project}/branches/{branch}
   * {branch} should be the id of a sheet.
   */
  name: string;
  /** The title of branch. AKA sheet's name. */
  title: string;
  /** The schema of branch. AKA sheet's statement. */
  schema: string;
  /** The metadata of the current editing schema. */
  schemaMetadata:
    | DatabaseMetadata
    | undefined;
  /** The baseline schema. */
  baselineSchema: string;
  /** The metadata of the baseline schema. */
  baselineSchemaMetadata:
    | DatabaseMetadata
    | undefined;
  /** The database engine of the branch. */
  engine: Engine;
  /**
   * The name of the baseline database.
   * Format: instances/{instance}/databases/{database}
   */
  baselineDatabase: string;
  /**
   * The name of the parent branch.
   * For main branch, it's empty.
   * For child branch, its format will be: projects/{project}/branches/{branch}
   */
  parentBranch: string;
  /** The etag of the branch. */
  etag: string;
  /**
   * The creator of the branch.
   * Format: users/{email}
   */
  creator: string;
  /**
   * The updater of the branch.
   * Format: users/{email}
   */
  updater: string;
  /** The timestamp when the branch was created. */
  createTime:
    | Date
    | undefined;
  /** The timestamp when the branch was last updated. */
  updateTime: Date | undefined;
}

export interface GetBranchRequest {
  /**
   * The name of the branch to retrieve.
   * Format: projects/{project}/branches/{branch}
   */
  name: string;
}

export interface ListBranchesRequest {
  /**
   * The parent resource of the branch.
   * Foramt: projects/{project}
   */
  parent: string;
  /** To filter the search result. */
  filter: string;
  /**
   * The maximum number of branches to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 branches will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListBranches` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListBranches` must match
   * the call that provided the page token.
   */
  pageToken: string;
  view: BranchView;
}

export interface ListBranchesResponse {
  /** The branches from the specified request. */
  branches: Branch[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreateBranchRequest {
  /**
   * The parent, which owns this collection of branches.
   * Format: project/{project}
   */
  parent: string;
  branch:
    | Branch
    | undefined;
  /**
   * The ID to use for the branch, which will become the final component of
   * the branch's resource name.
   * Format: [a-zA-Z][a-zA-Z0-9-_/]+.
   */
  branchId: string;
}

export interface UpdateBranchRequest {
  /**
   * The branch to update.
   *
   * The branch's `name` field is used to identify the branch to update.
   * Format: projects/{project}/branches/{branch}
   */
  branch:
    | Branch
    | undefined;
  /** The list of fields to update. */
  updateMask: string[] | undefined;
}

export interface MergeBranchRequest {
  /**
   * The name of the base branch to merge to.
   * Format: projects/{project}/branches/{branch}
   */
  name: string;
  /**
   * The head branch to merge from.
   * Format: projects/{project}/branches/{branch}
   */
  headBranch: string;
}

export interface DeleteBranchRequest {
  /**
   * The name of the branch to delete.
   * Format: projects/{project}/branches/{branch}
   */
  name: string;
}

export interface DiffMetadataRequest {
  /** The metadata of the source schema. */
  sourceMetadata:
    | DatabaseMetadata
    | undefined;
  /** The metadata of the target schema. */
  targetMetadata:
    | DatabaseMetadata
    | undefined;
  /** The database engine of the schema. */
  engine: Engine;
}

export interface DiffMetadataResponse {
  /** The diff of the metadata. */
  diff: string;
}

function createBaseBranch(): Branch {
  return {
    name: "",
    title: "",
    schema: "",
    schemaMetadata: undefined,
    baselineSchema: "",
    baselineSchemaMetadata: undefined,
    engine: 0,
    baselineDatabase: "",
    parentBranch: "",
    etag: "",
    creator: "",
    updater: "",
    createTime: undefined,
    updateTime: undefined,
  };
}

export const Branch = {
  encode(message: Branch, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.schema !== "") {
      writer.uint32(26).string(message.schema);
    }
    if (message.schemaMetadata !== undefined) {
      DatabaseMetadata.encode(message.schemaMetadata, writer.uint32(34).fork()).ldelim();
    }
    if (message.baselineSchema !== "") {
      writer.uint32(42).string(message.baselineSchema);
    }
    if (message.baselineSchemaMetadata !== undefined) {
      DatabaseMetadata.encode(message.baselineSchemaMetadata, writer.uint32(50).fork()).ldelim();
    }
    if (message.engine !== 0) {
      writer.uint32(56).int32(message.engine);
    }
    if (message.baselineDatabase !== "") {
      writer.uint32(66).string(message.baselineDatabase);
    }
    if (message.parentBranch !== "") {
      writer.uint32(74).string(message.parentBranch);
    }
    if (message.etag !== "") {
      writer.uint32(82).string(message.etag);
    }
    if (message.creator !== "") {
      writer.uint32(90).string(message.creator);
    }
    if (message.updater !== "") {
      writer.uint32(98).string(message.updater);
    }
    if (message.createTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createTime), writer.uint32(106).fork()).ldelim();
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(114).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Branch {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranch();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.title = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.schema = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.schemaMetadata = DatabaseMetadata.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.baselineSchema = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.baselineSchemaMetadata = DatabaseMetadata.decode(reader, reader.uint32());
          continue;
        case 7:
          if (tag !== 56) {
            break;
          }

          message.engine = reader.int32() as any;
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.baselineDatabase = reader.string();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.parentBranch = reader.string();
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.etag = reader.string();
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.updater = reader.string();
          continue;
        case 13:
          if (tag !== 106) {
            break;
          }

          message.createTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 14:
          if (tag !== 114) {
            break;
          }

          message.updateTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Branch {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      schema: isSet(object.schema) ? globalThis.String(object.schema) : "",
      schemaMetadata: isSet(object.schemaMetadata) ? DatabaseMetadata.fromJSON(object.schemaMetadata) : undefined,
      baselineSchema: isSet(object.baselineSchema) ? globalThis.String(object.baselineSchema) : "",
      baselineSchemaMetadata: isSet(object.baselineSchemaMetadata)
        ? DatabaseMetadata.fromJSON(object.baselineSchemaMetadata)
        : undefined,
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
      baselineDatabase: isSet(object.baselineDatabase) ? globalThis.String(object.baselineDatabase) : "",
      parentBranch: isSet(object.parentBranch) ? globalThis.String(object.parentBranch) : "",
      etag: isSet(object.etag) ? globalThis.String(object.etag) : "",
      creator: isSet(object.creator) ? globalThis.String(object.creator) : "",
      updater: isSet(object.updater) ? globalThis.String(object.updater) : "",
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
    };
  },

  toJSON(message: Branch): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.schema !== "") {
      obj.schema = message.schema;
    }
    if (message.schemaMetadata !== undefined) {
      obj.schemaMetadata = DatabaseMetadata.toJSON(message.schemaMetadata);
    }
    if (message.baselineSchema !== "") {
      obj.baselineSchema = message.baselineSchema;
    }
    if (message.baselineSchemaMetadata !== undefined) {
      obj.baselineSchemaMetadata = DatabaseMetadata.toJSON(message.baselineSchemaMetadata);
    }
    if (message.engine !== 0) {
      obj.engine = engineToJSON(message.engine);
    }
    if (message.baselineDatabase !== "") {
      obj.baselineDatabase = message.baselineDatabase;
    }
    if (message.parentBranch !== "") {
      obj.parentBranch = message.parentBranch;
    }
    if (message.etag !== "") {
      obj.etag = message.etag;
    }
    if (message.creator !== "") {
      obj.creator = message.creator;
    }
    if (message.updater !== "") {
      obj.updater = message.updater;
    }
    if (message.createTime !== undefined) {
      obj.createTime = message.createTime.toISOString();
    }
    if (message.updateTime !== undefined) {
      obj.updateTime = message.updateTime.toISOString();
    }
    return obj;
  },

  create(base?: DeepPartial<Branch>): Branch {
    return Branch.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Branch>): Branch {
    const message = createBaseBranch();
    message.name = object.name ?? "";
    message.title = object.title ?? "";
    message.schema = object.schema ?? "";
    message.schemaMetadata = (object.schemaMetadata !== undefined && object.schemaMetadata !== null)
      ? DatabaseMetadata.fromPartial(object.schemaMetadata)
      : undefined;
    message.baselineSchema = object.baselineSchema ?? "";
    message.baselineSchemaMetadata =
      (object.baselineSchemaMetadata !== undefined && object.baselineSchemaMetadata !== null)
        ? DatabaseMetadata.fromPartial(object.baselineSchemaMetadata)
        : undefined;
    message.engine = object.engine ?? 0;
    message.baselineDatabase = object.baselineDatabase ?? "";
    message.parentBranch = object.parentBranch ?? "";
    message.etag = object.etag ?? "";
    message.creator = object.creator ?? "";
    message.updater = object.updater ?? "";
    message.createTime = object.createTime ?? undefined;
    message.updateTime = object.updateTime ?? undefined;
    return message;
  },
};

function createBaseGetBranchRequest(): GetBranchRequest {
  return { name: "" };
}

export const GetBranchRequest = {
  encode(message: GetBranchRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetBranchRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetBranchRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetBranchRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetBranchRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<GetBranchRequest>): GetBranchRequest {
    return GetBranchRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GetBranchRequest>): GetBranchRequest {
    const message = createBaseGetBranchRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListBranchesRequest(): ListBranchesRequest {
  return { parent: "", filter: "", pageSize: 0, pageToken: "", view: 0 };
}

export const ListBranchesRequest = {
  encode(message: ListBranchesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.filter !== "") {
      writer.uint32(18).string(message.filter);
    }
    if (message.pageSize !== 0) {
      writer.uint32(24).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(34).string(message.pageToken);
    }
    if (message.view !== 0) {
      writer.uint32(40).int32(message.view);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListBranchesRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListBranchesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.filter = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.pageToken = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.view = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListBranchesRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      filter: isSet(object.filter) ? globalThis.String(object.filter) : "",
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
      view: isSet(object.view) ? branchViewFromJSON(object.view) : 0,
    };
  },

  toJSON(message: ListBranchesRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.filter !== "") {
      obj.filter = message.filter;
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    if (message.view !== 0) {
      obj.view = branchViewToJSON(message.view);
    }
    return obj;
  },

  create(base?: DeepPartial<ListBranchesRequest>): ListBranchesRequest {
    return ListBranchesRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListBranchesRequest>): ListBranchesRequest {
    const message = createBaseListBranchesRequest();
    message.parent = object.parent ?? "";
    message.filter = object.filter ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.view = object.view ?? 0;
    return message;
  },
};

function createBaseListBranchesResponse(): ListBranchesResponse {
  return { branches: [], nextPageToken: "" };
}

export const ListBranchesResponse = {
  encode(message: ListBranchesResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.branches) {
      Branch.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListBranchesResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListBranchesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.branches.push(Branch.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListBranchesResponse {
    return {
      branches: globalThis.Array.isArray(object?.branches) ? object.branches.map((e: any) => Branch.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListBranchesResponse): unknown {
    const obj: any = {};
    if (message.branches?.length) {
      obj.branches = message.branches.map((e) => Branch.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListBranchesResponse>): ListBranchesResponse {
    return ListBranchesResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListBranchesResponse>): ListBranchesResponse {
    const message = createBaseListBranchesResponse();
    message.branches = object.branches?.map((e) => Branch.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateBranchRequest(): CreateBranchRequest {
  return { parent: "", branch: undefined, branchId: "" };
}

export const CreateBranchRequest = {
  encode(message: CreateBranchRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.branch !== undefined) {
      Branch.encode(message.branch, writer.uint32(18).fork()).ldelim();
    }
    if (message.branchId !== "") {
      writer.uint32(26).string(message.branchId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateBranchRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateBranchRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.branch = Branch.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.branchId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateBranchRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      branch: isSet(object.branch) ? Branch.fromJSON(object.branch) : undefined,
      branchId: isSet(object.branchId) ? globalThis.String(object.branchId) : "",
    };
  },

  toJSON(message: CreateBranchRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.branch !== undefined) {
      obj.branch = Branch.toJSON(message.branch);
    }
    if (message.branchId !== "") {
      obj.branchId = message.branchId;
    }
    return obj;
  },

  create(base?: DeepPartial<CreateBranchRequest>): CreateBranchRequest {
    return CreateBranchRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateBranchRequest>): CreateBranchRequest {
    const message = createBaseCreateBranchRequest();
    message.parent = object.parent ?? "";
    message.branch = (object.branch !== undefined && object.branch !== null)
      ? Branch.fromPartial(object.branch)
      : undefined;
    message.branchId = object.branchId ?? "";
    return message;
  },
};

function createBaseUpdateBranchRequest(): UpdateBranchRequest {
  return { branch: undefined, updateMask: undefined };
}

export const UpdateBranchRequest = {
  encode(message: UpdateBranchRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.branch !== undefined) {
      Branch.encode(message.branch, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateBranchRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateBranchRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.branch = Branch.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UpdateBranchRequest {
    return {
      branch: isSet(object.branch) ? Branch.fromJSON(object.branch) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateBranchRequest): unknown {
    const obj: any = {};
    if (message.branch !== undefined) {
      obj.branch = Branch.toJSON(message.branch);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateBranchRequest>): UpdateBranchRequest {
    return UpdateBranchRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UpdateBranchRequest>): UpdateBranchRequest {
    const message = createBaseUpdateBranchRequest();
    message.branch = (object.branch !== undefined && object.branch !== null)
      ? Branch.fromPartial(object.branch)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseMergeBranchRequest(): MergeBranchRequest {
  return { name: "", headBranch: "" };
}

export const MergeBranchRequest = {
  encode(message: MergeBranchRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.headBranch !== "") {
      writer.uint32(18).string(message.headBranch);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MergeBranchRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMergeBranchRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.headBranch = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MergeBranchRequest {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      headBranch: isSet(object.headBranch) ? globalThis.String(object.headBranch) : "",
    };
  },

  toJSON(message: MergeBranchRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.headBranch !== "") {
      obj.headBranch = message.headBranch;
    }
    return obj;
  },

  create(base?: DeepPartial<MergeBranchRequest>): MergeBranchRequest {
    return MergeBranchRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<MergeBranchRequest>): MergeBranchRequest {
    const message = createBaseMergeBranchRequest();
    message.name = object.name ?? "";
    message.headBranch = object.headBranch ?? "";
    return message;
  },
};

function createBaseDeleteBranchRequest(): DeleteBranchRequest {
  return { name: "" };
}

export const DeleteBranchRequest = {
  encode(message: DeleteBranchRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteBranchRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteBranchRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DeleteBranchRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: DeleteBranchRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<DeleteBranchRequest>): DeleteBranchRequest {
    return DeleteBranchRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DeleteBranchRequest>): DeleteBranchRequest {
    const message = createBaseDeleteBranchRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseDiffMetadataRequest(): DiffMetadataRequest {
  return { sourceMetadata: undefined, targetMetadata: undefined, engine: 0 };
}

export const DiffMetadataRequest = {
  encode(message: DiffMetadataRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sourceMetadata !== undefined) {
      DatabaseMetadata.encode(message.sourceMetadata, writer.uint32(10).fork()).ldelim();
    }
    if (message.targetMetadata !== undefined) {
      DatabaseMetadata.encode(message.targetMetadata, writer.uint32(18).fork()).ldelim();
    }
    if (message.engine !== 0) {
      writer.uint32(24).int32(message.engine);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DiffMetadataRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDiffMetadataRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sourceMetadata = DatabaseMetadata.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.targetMetadata = DatabaseMetadata.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.engine = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DiffMetadataRequest {
    return {
      sourceMetadata: isSet(object.sourceMetadata) ? DatabaseMetadata.fromJSON(object.sourceMetadata) : undefined,
      targetMetadata: isSet(object.targetMetadata) ? DatabaseMetadata.fromJSON(object.targetMetadata) : undefined,
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
    };
  },

  toJSON(message: DiffMetadataRequest): unknown {
    const obj: any = {};
    if (message.sourceMetadata !== undefined) {
      obj.sourceMetadata = DatabaseMetadata.toJSON(message.sourceMetadata);
    }
    if (message.targetMetadata !== undefined) {
      obj.targetMetadata = DatabaseMetadata.toJSON(message.targetMetadata);
    }
    if (message.engine !== 0) {
      obj.engine = engineToJSON(message.engine);
    }
    return obj;
  },

  create(base?: DeepPartial<DiffMetadataRequest>): DiffMetadataRequest {
    return DiffMetadataRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DiffMetadataRequest>): DiffMetadataRequest {
    const message = createBaseDiffMetadataRequest();
    message.sourceMetadata = (object.sourceMetadata !== undefined && object.sourceMetadata !== null)
      ? DatabaseMetadata.fromPartial(object.sourceMetadata)
      : undefined;
    message.targetMetadata = (object.targetMetadata !== undefined && object.targetMetadata !== null)
      ? DatabaseMetadata.fromPartial(object.targetMetadata)
      : undefined;
    message.engine = object.engine ?? 0;
    return message;
  },
};

function createBaseDiffMetadataResponse(): DiffMetadataResponse {
  return { diff: "" };
}

export const DiffMetadataResponse = {
  encode(message: DiffMetadataResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.diff !== "") {
      writer.uint32(10).string(message.diff);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DiffMetadataResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDiffMetadataResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.diff = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DiffMetadataResponse {
    return { diff: isSet(object.diff) ? globalThis.String(object.diff) : "" };
  },

  toJSON(message: DiffMetadataResponse): unknown {
    const obj: any = {};
    if (message.diff !== "") {
      obj.diff = message.diff;
    }
    return obj;
  },

  create(base?: DeepPartial<DiffMetadataResponse>): DiffMetadataResponse {
    return DiffMetadataResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DiffMetadataResponse>): DiffMetadataResponse {
    const message = createBaseDiffMetadataResponse();
    message.diff = object.diff ?? "";
    return message;
  },
};

export type BranchServiceDefinition = typeof BranchServiceDefinition;
export const BranchServiceDefinition = {
  name: "BranchService",
  fullName: "bytebase.v1.BranchService",
  methods: {
    getBranch: {
      name: "GetBranch",
      requestType: GetBranchRequest,
      requestStream: false,
      responseType: Branch,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              34,
              18,
              32,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              98,
              114,
              97,
              110,
              99,
              104,
              101,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    listBranches: {
      name: "ListBranches",
      requestType: ListBranchesRequest,
      requestStream: false,
      responseType: ListBranchesResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [
            new Uint8Array([
              34,
              18,
              32,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              125,
              47,
              98,
              114,
              97,
              110,
              99,
              104,
              101,
              115,
            ]),
          ],
        },
      },
    },
    createBranch: {
      name: "CreateBranch",
      requestType: CreateBranchRequest,
      requestStream: false,
      responseType: Branch,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([13, 112, 97, 114, 101, 110, 116, 44, 98, 114, 97, 110, 99, 104])],
          578365826: [
            new Uint8Array([
              40,
              58,
              6,
              98,
              114,
              97,
              110,
              99,
              104,
              34,
              30,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              125,
              47,
              98,
              114,
              97,
              110,
              99,
              104,
            ]),
          ],
        },
      },
    },
    updateBranch: {
      name: "UpdateBranch",
      requestType: UpdateBranchRequest,
      requestStream: false,
      responseType: Branch,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([18, 98, 114, 97, 110, 99, 104, 44, 117, 112, 100, 97, 116, 101, 95, 109, 97, 115, 107]),
          ],
          578365826: [
            new Uint8Array([
              49,
              58,
              6,
              98,
              114,
              97,
              110,
              99,
              104,
              50,
              39,
              47,
              118,
              49,
              47,
              123,
              98,
              114,
              97,
              110,
              99,
              104,
              46,
              110,
              97,
              109,
              101,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              98,
              114,
              97,
              110,
              99,
              104,
              101,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    mergeBranch: {
      name: "MergeBranch",
      requestType: MergeBranchRequest,
      requestStream: false,
      responseType: Branch,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([16, 110, 97, 109, 101, 44, 116, 97, 114, 103, 101, 116, 95, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              40,
              34,
              38,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              98,
              114,
              97,
              110,
              99,
              104,
              101,
              115,
              47,
              42,
              125,
              58,
              109,
              101,
              114,
              103,
              101,
            ]),
          ],
        },
      },
    },
    deleteBranch: {
      name: "DeleteBranch",
      requestType: DeleteBranchRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              34,
              42,
              32,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              98,
              114,
              97,
              110,
              99,
              104,
              101,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    diffMetadata: {
      name: "DiffMetadata",
      requestType: DiffMetadataRequest,
      requestStream: false,
      responseType: DiffMetadataResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              34,
              58,
              1,
              42,
              34,
              29,
              47,
              118,
              49,
              47,
              115,
              99,
              104,
              101,
              109,
              97,
              68,
              101,
              115,
              105,
              103,
              110,
              58,
              100,
              105,
              102,
              102,
              77,
              101,
              116,
              97,
              100,
              97,
              116,
              97,
            ]),
          ],
        },
      },
    },
  },
} as const;

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Long ? string | number | Long : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function toTimestamp(date: Date): Timestamp {
  const seconds = numberToLong(date.getTime() / 1_000);
  const nanos = (date.getTime() % 1_000) * 1_000_000;
  return { seconds, nanos };
}

function fromTimestamp(t: Timestamp): Date {
  let millis = (t.seconds.toNumber() || 0) * 1_000;
  millis += (t.nanos || 0) / 1_000_000;
  return new globalThis.Date(millis);
}

function fromJsonTimestamp(o: any): Date {
  if (o instanceof globalThis.Date) {
    return o;
  } else if (typeof o === "string") {
    return new globalThis.Date(o);
  } else {
    return fromTimestamp(Timestamp.fromJSON(o));
  }
}

function numberToLong(number: number) {
  return Long.fromNumber(number);
}

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
