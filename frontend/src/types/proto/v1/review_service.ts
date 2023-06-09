/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { FieldMask } from "../google/protobuf/field_mask";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.v1";

export enum ReviewStatus {
  REVIEW_STATUS_UNSPECIFIED = 0,
  OPEN = 1,
  DONE = 2,
  CANCELED = 3,
  UNRECOGNIZED = -1,
}

export function reviewStatusFromJSON(object: any): ReviewStatus {
  switch (object) {
    case 0:
    case "REVIEW_STATUS_UNSPECIFIED":
      return ReviewStatus.REVIEW_STATUS_UNSPECIFIED;
    case 1:
    case "OPEN":
      return ReviewStatus.OPEN;
    case 2:
    case "DONE":
      return ReviewStatus.DONE;
    case 3:
    case "CANCELED":
      return ReviewStatus.CANCELED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ReviewStatus.UNRECOGNIZED;
  }
}

export function reviewStatusToJSON(object: ReviewStatus): string {
  switch (object) {
    case ReviewStatus.REVIEW_STATUS_UNSPECIFIED:
      return "REVIEW_STATUS_UNSPECIFIED";
    case ReviewStatus.OPEN:
      return "OPEN";
    case ReviewStatus.DONE:
      return "DONE";
    case ReviewStatus.CANCELED:
      return "CANCELED";
    case ReviewStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface GetReviewRequest {
  /**
   * The name of the review to retrieve.
   * Format: projects/{project}/reviews/{review}
   */
  name: string;
}

export interface CreateReviewRequest {
  /**
   * The parent, which owns this collection of reviews.
   * Format: projects/{project}
   */
  parent: string;
  /** The review to create. */
  review?: Review;
}

export interface ListReviewsRequest {
  /**
   * The parent, which owns this collection of reviews.
   * Format: projects/{project}
   * Use "projects/-" to list all reviews from all projects.
   */
  parent: string;
  /**
   * The maximum number of reviews to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 reviews will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListReviews` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListReviews` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListReviewsResponse {
  /** The reviews from the specified request. */
  reviews: Review[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface UpdateReviewRequest {
  /**
   * The review to update.
   *
   * The review's `name` field is used to identify the review to update.
   * Format: projects/{project}/reviews/{review}
   */
  review?: Review;
  /** The list of fields to update. */
  updateMask?: string[];
}

export interface BatchUpdateReviewsRequest {
  /**
   * The parent resource shared by all reviews being updated.
   * Format: projects/{project}
   * If the operation spans parents, a dash (-) may be accepted as a wildcard.
   * We only support updating the status of databases for now.
   */
  parent: string;
  /**
   * The request message specifying the resources to update.
   * A maximum of 1000 databases can be modified in a batch.
   */
  requests: UpdateReviewRequest[];
}

export interface BatchUpdateReviewsResponse {
  /** Reviews updated. */
  reviews: Review[];
}

export interface ApproveReviewRequest {
  /**
   * The name of the review to add an approver.
   * Format: projects/{project}/reviews/{review}
   */
  name: string;
}

export interface RejectReviewRequest {
  /**
   * The name of the review to add an approver.
   * Format: projects/{project}/reviews/{review}
   */
  name: string;
}

export interface Review {
  /**
   * The name of the review.
   * `review` is a system generated ID.
   * Format: projects/{project}/reviews/{review}
   */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  title: string;
  /**
   * The plan associated with the review.
   * Can be empty.
   * Format: projects/{project}/plans/{plan}
   */
  plan: string;
  /**
   * The rollout associated with the review.
   * Can be empty.
   * Format: projects/{project}/rollouts/{rollout}
   */
  rollout: string;
  description: string;
  status: ReviewStatus;
  /** Format: users/hello@world.com */
  assignee: string;
  assigneeAttention: boolean;
  approvalTemplates: ApprovalTemplate[];
  approvers: Review_Approver[];
  /**
   * If the value is `false`, it means that the backend is still finding matching approval templates.
   * If `true`, approval_templates & approvers & approval_finding_error are available.
   */
  approvalFindingDone: boolean;
  approvalFindingError: string;
  /**
   * The subscribers.
   * Format: users/hello@world.com
   */
  subscribers: string[];
  /** Format: users/hello@world.com */
  creator: string;
  createTime?: Date;
  updateTime?: Date;
}

export interface Review_Approver {
  /** The new status. */
  status: Review_Approver_Status;
  /** Format: users/hello@world.com */
  principal: string;
}

export enum Review_Approver_Status {
  STATUS_UNSPECIFIED = 0,
  PENDING = 1,
  APPROVED = 2,
  REJECTED = 3,
  UNRECOGNIZED = -1,
}

export function review_Approver_StatusFromJSON(object: any): Review_Approver_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return Review_Approver_Status.STATUS_UNSPECIFIED;
    case 1:
    case "PENDING":
      return Review_Approver_Status.PENDING;
    case 2:
    case "APPROVED":
      return Review_Approver_Status.APPROVED;
    case 3:
    case "REJECTED":
      return Review_Approver_Status.REJECTED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Review_Approver_Status.UNRECOGNIZED;
  }
}

export function review_Approver_StatusToJSON(object: Review_Approver_Status): string {
  switch (object) {
    case Review_Approver_Status.STATUS_UNSPECIFIED:
      return "STATUS_UNSPECIFIED";
    case Review_Approver_Status.PENDING:
      return "PENDING";
    case Review_Approver_Status.APPROVED:
      return "APPROVED";
    case Review_Approver_Status.REJECTED:
      return "REJECTED";
    case Review_Approver_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface ApprovalTemplate {
  flow?: ApprovalFlow;
  title: string;
  description: string;
  /**
   * The name of the creator in users/{email} format.
   * TODO: we should mark it as OUTPUT_ONLY, but currently the frontend will post the approval setting with creator.
   */
  creator: string;
}

export interface ApprovalFlow {
  steps: ApprovalStep[];
}

export interface ApprovalStep {
  type: ApprovalStep_Type;
  nodes: ApprovalNode[];
}

/**
 * Type of the ApprovalStep
 * ALL means every node must be approved to proceed.
 * ANY means approving any node will proceed.
 */
export enum ApprovalStep_Type {
  TYPE_UNSPECIFIED = 0,
  ALL = 1,
  ANY = 2,
  UNRECOGNIZED = -1,
}

export function approvalStep_TypeFromJSON(object: any): ApprovalStep_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return ApprovalStep_Type.TYPE_UNSPECIFIED;
    case 1:
    case "ALL":
      return ApprovalStep_Type.ALL;
    case 2:
    case "ANY":
      return ApprovalStep_Type.ANY;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ApprovalStep_Type.UNRECOGNIZED;
  }
}

export function approvalStep_TypeToJSON(object: ApprovalStep_Type): string {
  switch (object) {
    case ApprovalStep_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case ApprovalStep_Type.ALL:
      return "ALL";
    case ApprovalStep_Type.ANY:
      return "ANY";
    case ApprovalStep_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface ApprovalNode {
  type: ApprovalNode_Type;
  groupValue?:
    | ApprovalNode_GroupValue
    | undefined;
  /** Format: roles/{role} */
  role?: string | undefined;
}

/**
 * Type of the ApprovalNode.
 * type determines who should approve this node.
 * ANY_IN_GROUP means the ApprovalNode can be approved by an user from our predefined user group.
 * See GroupValue below for the predefined user groups.
 */
export enum ApprovalNode_Type {
  TYPE_UNSPECIFIED = 0,
  ANY_IN_GROUP = 1,
  UNRECOGNIZED = -1,
}

export function approvalNode_TypeFromJSON(object: any): ApprovalNode_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return ApprovalNode_Type.TYPE_UNSPECIFIED;
    case 1:
    case "ANY_IN_GROUP":
      return ApprovalNode_Type.ANY_IN_GROUP;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ApprovalNode_Type.UNRECOGNIZED;
  }
}

export function approvalNode_TypeToJSON(object: ApprovalNode_Type): string {
  switch (object) {
    case ApprovalNode_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case ApprovalNode_Type.ANY_IN_GROUP:
      return "ANY_IN_GROUP";
    case ApprovalNode_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/**
 * The predefined user groups are:
 * - WORKSPACE_OWNER
 * - WORKSPACE_DBA
 * - PROJECT_OWNER
 * - PROJECT_MEMBER
 */
export enum ApprovalNode_GroupValue {
  GROUP_VALUE_UNSPECIFILED = 0,
  WORKSPACE_OWNER = 1,
  WORKSPACE_DBA = 2,
  PROJECT_OWNER = 3,
  PROJECT_MEMBER = 4,
  UNRECOGNIZED = -1,
}

export function approvalNode_GroupValueFromJSON(object: any): ApprovalNode_GroupValue {
  switch (object) {
    case 0:
    case "GROUP_VALUE_UNSPECIFILED":
      return ApprovalNode_GroupValue.GROUP_VALUE_UNSPECIFILED;
    case 1:
    case "WORKSPACE_OWNER":
      return ApprovalNode_GroupValue.WORKSPACE_OWNER;
    case 2:
    case "WORKSPACE_DBA":
      return ApprovalNode_GroupValue.WORKSPACE_DBA;
    case 3:
    case "PROJECT_OWNER":
      return ApprovalNode_GroupValue.PROJECT_OWNER;
    case 4:
    case "PROJECT_MEMBER":
      return ApprovalNode_GroupValue.PROJECT_MEMBER;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ApprovalNode_GroupValue.UNRECOGNIZED;
  }
}

export function approvalNode_GroupValueToJSON(object: ApprovalNode_GroupValue): string {
  switch (object) {
    case ApprovalNode_GroupValue.GROUP_VALUE_UNSPECIFILED:
      return "GROUP_VALUE_UNSPECIFILED";
    case ApprovalNode_GroupValue.WORKSPACE_OWNER:
      return "WORKSPACE_OWNER";
    case ApprovalNode_GroupValue.WORKSPACE_DBA:
      return "WORKSPACE_DBA";
    case ApprovalNode_GroupValue.PROJECT_OWNER:
      return "PROJECT_OWNER";
    case ApprovalNode_GroupValue.PROJECT_MEMBER:
      return "PROJECT_MEMBER";
    case ApprovalNode_GroupValue.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

function createBaseGetReviewRequest(): GetReviewRequest {
  return { name: "" };
}

export const GetReviewRequest = {
  encode(message: GetReviewRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetReviewRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetReviewRequest();
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

  fromJSON(object: any): GetReviewRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetReviewRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<GetReviewRequest>): GetReviewRequest {
    return GetReviewRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetReviewRequest>): GetReviewRequest {
    const message = createBaseGetReviewRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseCreateReviewRequest(): CreateReviewRequest {
  return { parent: "", review: undefined };
}

export const CreateReviewRequest = {
  encode(message: CreateReviewRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.review !== undefined) {
      Review.encode(message.review, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateReviewRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateReviewRequest();
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

          message.review = Review.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateReviewRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      review: isSet(object.review) ? Review.fromJSON(object.review) : undefined,
    };
  },

  toJSON(message: CreateReviewRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.review !== undefined && (obj.review = message.review ? Review.toJSON(message.review) : undefined);
    return obj;
  },

  create(base?: DeepPartial<CreateReviewRequest>): CreateReviewRequest {
    return CreateReviewRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<CreateReviewRequest>): CreateReviewRequest {
    const message = createBaseCreateReviewRequest();
    message.parent = object.parent ?? "";
    message.review = (object.review !== undefined && object.review !== null)
      ? Review.fromPartial(object.review)
      : undefined;
    return message;
  },
};

function createBaseListReviewsRequest(): ListReviewsRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListReviewsRequest = {
  encode(message: ListReviewsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.pageSize !== 0) {
      writer.uint32(16).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(26).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListReviewsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListReviewsRequest();
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
          if (tag !== 16) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.pageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListReviewsRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListReviewsRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  create(base?: DeepPartial<ListReviewsRequest>): ListReviewsRequest {
    return ListReviewsRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListReviewsRequest>): ListReviewsRequest {
    const message = createBaseListReviewsRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListReviewsResponse(): ListReviewsResponse {
  return { reviews: [], nextPageToken: "" };
}

export const ListReviewsResponse = {
  encode(message: ListReviewsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.reviews) {
      Review.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListReviewsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListReviewsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.reviews.push(Review.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListReviewsResponse {
    return {
      reviews: Array.isArray(object?.reviews) ? object.reviews.map((e: any) => Review.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListReviewsResponse): unknown {
    const obj: any = {};
    if (message.reviews) {
      obj.reviews = message.reviews.map((e) => e ? Review.toJSON(e) : undefined);
    } else {
      obj.reviews = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  create(base?: DeepPartial<ListReviewsResponse>): ListReviewsResponse {
    return ListReviewsResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListReviewsResponse>): ListReviewsResponse {
    const message = createBaseListReviewsResponse();
    message.reviews = object.reviews?.map((e) => Review.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseUpdateReviewRequest(): UpdateReviewRequest {
  return { review: undefined, updateMask: undefined };
}

export const UpdateReviewRequest = {
  encode(message: UpdateReviewRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.review !== undefined) {
      Review.encode(message.review, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateReviewRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateReviewRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.review = Review.decode(reader, reader.uint32());
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

  fromJSON(object: any): UpdateReviewRequest {
    return {
      review: isSet(object.review) ? Review.fromJSON(object.review) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateReviewRequest): unknown {
    const obj: any = {};
    message.review !== undefined && (obj.review = message.review ? Review.toJSON(message.review) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  create(base?: DeepPartial<UpdateReviewRequest>): UpdateReviewRequest {
    return UpdateReviewRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UpdateReviewRequest>): UpdateReviewRequest {
    const message = createBaseUpdateReviewRequest();
    message.review = (object.review !== undefined && object.review !== null)
      ? Review.fromPartial(object.review)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseBatchUpdateReviewsRequest(): BatchUpdateReviewsRequest {
  return { parent: "", requests: [] };
}

export const BatchUpdateReviewsRequest = {
  encode(message: BatchUpdateReviewsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    for (const v of message.requests) {
      UpdateReviewRequest.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchUpdateReviewsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchUpdateReviewsRequest();
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

          message.requests.push(UpdateReviewRequest.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchUpdateReviewsRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      requests: Array.isArray(object?.requests) ? object.requests.map((e: any) => UpdateReviewRequest.fromJSON(e)) : [],
    };
  },

  toJSON(message: BatchUpdateReviewsRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    if (message.requests) {
      obj.requests = message.requests.map((e) => e ? UpdateReviewRequest.toJSON(e) : undefined);
    } else {
      obj.requests = [];
    }
    return obj;
  },

  create(base?: DeepPartial<BatchUpdateReviewsRequest>): BatchUpdateReviewsRequest {
    return BatchUpdateReviewsRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<BatchUpdateReviewsRequest>): BatchUpdateReviewsRequest {
    const message = createBaseBatchUpdateReviewsRequest();
    message.parent = object.parent ?? "";
    message.requests = object.requests?.map((e) => UpdateReviewRequest.fromPartial(e)) || [];
    return message;
  },
};

function createBaseBatchUpdateReviewsResponse(): BatchUpdateReviewsResponse {
  return { reviews: [] };
}

export const BatchUpdateReviewsResponse = {
  encode(message: BatchUpdateReviewsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.reviews) {
      Review.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchUpdateReviewsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchUpdateReviewsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.reviews.push(Review.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchUpdateReviewsResponse {
    return { reviews: Array.isArray(object?.reviews) ? object.reviews.map((e: any) => Review.fromJSON(e)) : [] };
  },

  toJSON(message: BatchUpdateReviewsResponse): unknown {
    const obj: any = {};
    if (message.reviews) {
      obj.reviews = message.reviews.map((e) => e ? Review.toJSON(e) : undefined);
    } else {
      obj.reviews = [];
    }
    return obj;
  },

  create(base?: DeepPartial<BatchUpdateReviewsResponse>): BatchUpdateReviewsResponse {
    return BatchUpdateReviewsResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<BatchUpdateReviewsResponse>): BatchUpdateReviewsResponse {
    const message = createBaseBatchUpdateReviewsResponse();
    message.reviews = object.reviews?.map((e) => Review.fromPartial(e)) || [];
    return message;
  },
};

function createBaseApproveReviewRequest(): ApproveReviewRequest {
  return { name: "" };
}

export const ApproveReviewRequest = {
  encode(message: ApproveReviewRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApproveReviewRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApproveReviewRequest();
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

  fromJSON(object: any): ApproveReviewRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: ApproveReviewRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<ApproveReviewRequest>): ApproveReviewRequest {
    return ApproveReviewRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ApproveReviewRequest>): ApproveReviewRequest {
    const message = createBaseApproveReviewRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseRejectReviewRequest(): RejectReviewRequest {
  return { name: "" };
}

export const RejectReviewRequest = {
  encode(message: RejectReviewRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RejectReviewRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRejectReviewRequest();
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

  fromJSON(object: any): RejectReviewRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: RejectReviewRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<RejectReviewRequest>): RejectReviewRequest {
    return RejectReviewRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<RejectReviewRequest>): RejectReviewRequest {
    const message = createBaseRejectReviewRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseReview(): Review {
  return {
    name: "",
    uid: "",
    title: "",
    plan: "",
    rollout: "",
    description: "",
    status: 0,
    assignee: "",
    assigneeAttention: false,
    approvalTemplates: [],
    approvers: [],
    approvalFindingDone: false,
    approvalFindingError: "",
    subscribers: [],
    creator: "",
    createTime: undefined,
    updateTime: undefined,
  };
}

export const Review = {
  encode(message: Review, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.title !== "") {
      writer.uint32(26).string(message.title);
    }
    if (message.plan !== "") {
      writer.uint32(130).string(message.plan);
    }
    if (message.rollout !== "") {
      writer.uint32(138).string(message.rollout);
    }
    if (message.description !== "") {
      writer.uint32(34).string(message.description);
    }
    if (message.status !== 0) {
      writer.uint32(40).int32(message.status);
    }
    if (message.assignee !== "") {
      writer.uint32(50).string(message.assignee);
    }
    if (message.assigneeAttention === true) {
      writer.uint32(56).bool(message.assigneeAttention);
    }
    for (const v of message.approvalTemplates) {
      ApprovalTemplate.encode(v!, writer.uint32(66).fork()).ldelim();
    }
    for (const v of message.approvers) {
      Review_Approver.encode(v!, writer.uint32(74).fork()).ldelim();
    }
    if (message.approvalFindingDone === true) {
      writer.uint32(80).bool(message.approvalFindingDone);
    }
    if (message.approvalFindingError !== "") {
      writer.uint32(90).string(message.approvalFindingError);
    }
    for (const v of message.subscribers) {
      writer.uint32(98).string(v!);
    }
    if (message.creator !== "") {
      writer.uint32(106).string(message.creator);
    }
    if (message.createTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createTime), writer.uint32(114).fork()).ldelim();
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(122).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Review {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseReview();
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

          message.uid = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.title = reader.string();
          continue;
        case 16:
          if (tag !== 130) {
            break;
          }

          message.plan = reader.string();
          continue;
        case 17:
          if (tag !== 138) {
            break;
          }

          message.rollout = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.description = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.status = reader.int32() as any;
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.assignee = reader.string();
          continue;
        case 7:
          if (tag !== 56) {
            break;
          }

          message.assigneeAttention = reader.bool();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.approvalTemplates.push(ApprovalTemplate.decode(reader, reader.uint32()));
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.approvers.push(Review_Approver.decode(reader, reader.uint32()));
          continue;
        case 10:
          if (tag !== 80) {
            break;
          }

          message.approvalFindingDone = reader.bool();
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.approvalFindingError = reader.string();
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.subscribers.push(reader.string());
          continue;
        case 13:
          if (tag !== 106) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 14:
          if (tag !== 114) {
            break;
          }

          message.createTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 15:
          if (tag !== 122) {
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

  fromJSON(object: any): Review {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      title: isSet(object.title) ? String(object.title) : "",
      plan: isSet(object.plan) ? String(object.plan) : "",
      rollout: isSet(object.rollout) ? String(object.rollout) : "",
      description: isSet(object.description) ? String(object.description) : "",
      status: isSet(object.status) ? reviewStatusFromJSON(object.status) : 0,
      assignee: isSet(object.assignee) ? String(object.assignee) : "",
      assigneeAttention: isSet(object.assigneeAttention) ? Boolean(object.assigneeAttention) : false,
      approvalTemplates: Array.isArray(object?.approvalTemplates)
        ? object.approvalTemplates.map((e: any) => ApprovalTemplate.fromJSON(e))
        : [],
      approvers: Array.isArray(object?.approvers) ? object.approvers.map((e: any) => Review_Approver.fromJSON(e)) : [],
      approvalFindingDone: isSet(object.approvalFindingDone) ? Boolean(object.approvalFindingDone) : false,
      approvalFindingError: isSet(object.approvalFindingError) ? String(object.approvalFindingError) : "",
      subscribers: Array.isArray(object?.subscribers) ? object.subscribers.map((e: any) => String(e)) : [],
      creator: isSet(object.creator) ? String(object.creator) : "",
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
    };
  },

  toJSON(message: Review): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.title !== undefined && (obj.title = message.title);
    message.plan !== undefined && (obj.plan = message.plan);
    message.rollout !== undefined && (obj.rollout = message.rollout);
    message.description !== undefined && (obj.description = message.description);
    message.status !== undefined && (obj.status = reviewStatusToJSON(message.status));
    message.assignee !== undefined && (obj.assignee = message.assignee);
    message.assigneeAttention !== undefined && (obj.assigneeAttention = message.assigneeAttention);
    if (message.approvalTemplates) {
      obj.approvalTemplates = message.approvalTemplates.map((e) => e ? ApprovalTemplate.toJSON(e) : undefined);
    } else {
      obj.approvalTemplates = [];
    }
    if (message.approvers) {
      obj.approvers = message.approvers.map((e) => e ? Review_Approver.toJSON(e) : undefined);
    } else {
      obj.approvers = [];
    }
    message.approvalFindingDone !== undefined && (obj.approvalFindingDone = message.approvalFindingDone);
    message.approvalFindingError !== undefined && (obj.approvalFindingError = message.approvalFindingError);
    if (message.subscribers) {
      obj.subscribers = message.subscribers.map((e) => e);
    } else {
      obj.subscribers = [];
    }
    message.creator !== undefined && (obj.creator = message.creator);
    message.createTime !== undefined && (obj.createTime = message.createTime.toISOString());
    message.updateTime !== undefined && (obj.updateTime = message.updateTime.toISOString());
    return obj;
  },

  create(base?: DeepPartial<Review>): Review {
    return Review.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Review>): Review {
    const message = createBaseReview();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.title = object.title ?? "";
    message.plan = object.plan ?? "";
    message.rollout = object.rollout ?? "";
    message.description = object.description ?? "";
    message.status = object.status ?? 0;
    message.assignee = object.assignee ?? "";
    message.assigneeAttention = object.assigneeAttention ?? false;
    message.approvalTemplates = object.approvalTemplates?.map((e) => ApprovalTemplate.fromPartial(e)) || [];
    message.approvers = object.approvers?.map((e) => Review_Approver.fromPartial(e)) || [];
    message.approvalFindingDone = object.approvalFindingDone ?? false;
    message.approvalFindingError = object.approvalFindingError ?? "";
    message.subscribers = object.subscribers?.map((e) => e) || [];
    message.creator = object.creator ?? "";
    message.createTime = object.createTime ?? undefined;
    message.updateTime = object.updateTime ?? undefined;
    return message;
  },
};

function createBaseReview_Approver(): Review_Approver {
  return { status: 0, principal: "" };
}

export const Review_Approver = {
  encode(message: Review_Approver, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.status !== 0) {
      writer.uint32(8).int32(message.status);
    }
    if (message.principal !== "") {
      writer.uint32(18).string(message.principal);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Review_Approver {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseReview_Approver();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.status = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.principal = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Review_Approver {
    return {
      status: isSet(object.status) ? review_Approver_StatusFromJSON(object.status) : 0,
      principal: isSet(object.principal) ? String(object.principal) : "",
    };
  },

  toJSON(message: Review_Approver): unknown {
    const obj: any = {};
    message.status !== undefined && (obj.status = review_Approver_StatusToJSON(message.status));
    message.principal !== undefined && (obj.principal = message.principal);
    return obj;
  },

  create(base?: DeepPartial<Review_Approver>): Review_Approver {
    return Review_Approver.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Review_Approver>): Review_Approver {
    const message = createBaseReview_Approver();
    message.status = object.status ?? 0;
    message.principal = object.principal ?? "";
    return message;
  },
};

function createBaseApprovalTemplate(): ApprovalTemplate {
  return { flow: undefined, title: "", description: "", creator: "" };
}

export const ApprovalTemplate = {
  encode(message: ApprovalTemplate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.flow !== undefined) {
      ApprovalFlow.encode(message.flow, writer.uint32(10).fork()).ldelim();
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.creator !== "") {
      writer.uint32(34).string(message.creator);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalTemplate {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalTemplate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.flow = ApprovalFlow.decode(reader, reader.uint32());
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

          message.description = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.creator = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ApprovalTemplate {
    return {
      flow: isSet(object.flow) ? ApprovalFlow.fromJSON(object.flow) : undefined,
      title: isSet(object.title) ? String(object.title) : "",
      description: isSet(object.description) ? String(object.description) : "",
      creator: isSet(object.creator) ? String(object.creator) : "",
    };
  },

  toJSON(message: ApprovalTemplate): unknown {
    const obj: any = {};
    message.flow !== undefined && (obj.flow = message.flow ? ApprovalFlow.toJSON(message.flow) : undefined);
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined && (obj.description = message.description);
    message.creator !== undefined && (obj.creator = message.creator);
    return obj;
  },

  create(base?: DeepPartial<ApprovalTemplate>): ApprovalTemplate {
    return ApprovalTemplate.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ApprovalTemplate>): ApprovalTemplate {
    const message = createBaseApprovalTemplate();
    message.flow = (object.flow !== undefined && object.flow !== null)
      ? ApprovalFlow.fromPartial(object.flow)
      : undefined;
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.creator = object.creator ?? "";
    return message;
  },
};

function createBaseApprovalFlow(): ApprovalFlow {
  return { steps: [] };
}

export const ApprovalFlow = {
  encode(message: ApprovalFlow, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.steps) {
      ApprovalStep.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalFlow {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalFlow();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.steps.push(ApprovalStep.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ApprovalFlow {
    return { steps: Array.isArray(object?.steps) ? object.steps.map((e: any) => ApprovalStep.fromJSON(e)) : [] };
  },

  toJSON(message: ApprovalFlow): unknown {
    const obj: any = {};
    if (message.steps) {
      obj.steps = message.steps.map((e) => e ? ApprovalStep.toJSON(e) : undefined);
    } else {
      obj.steps = [];
    }
    return obj;
  },

  create(base?: DeepPartial<ApprovalFlow>): ApprovalFlow {
    return ApprovalFlow.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ApprovalFlow>): ApprovalFlow {
    const message = createBaseApprovalFlow();
    message.steps = object.steps?.map((e) => ApprovalStep.fromPartial(e)) || [];
    return message;
  },
};

function createBaseApprovalStep(): ApprovalStep {
  return { type: 0, nodes: [] };
}

export const ApprovalStep = {
  encode(message: ApprovalStep, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    for (const v of message.nodes) {
      ApprovalNode.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalStep {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalStep();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nodes.push(ApprovalNode.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ApprovalStep {
    return {
      type: isSet(object.type) ? approvalStep_TypeFromJSON(object.type) : 0,
      nodes: Array.isArray(object?.nodes) ? object.nodes.map((e: any) => ApprovalNode.fromJSON(e)) : [],
    };
  },

  toJSON(message: ApprovalStep): unknown {
    const obj: any = {};
    message.type !== undefined && (obj.type = approvalStep_TypeToJSON(message.type));
    if (message.nodes) {
      obj.nodes = message.nodes.map((e) => e ? ApprovalNode.toJSON(e) : undefined);
    } else {
      obj.nodes = [];
    }
    return obj;
  },

  create(base?: DeepPartial<ApprovalStep>): ApprovalStep {
    return ApprovalStep.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ApprovalStep>): ApprovalStep {
    const message = createBaseApprovalStep();
    message.type = object.type ?? 0;
    message.nodes = object.nodes?.map((e) => ApprovalNode.fromPartial(e)) || [];
    return message;
  },
};

function createBaseApprovalNode(): ApprovalNode {
  return { type: 0, groupValue: undefined, role: undefined };
}

export const ApprovalNode = {
  encode(message: ApprovalNode, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    if (message.groupValue !== undefined) {
      writer.uint32(16).int32(message.groupValue);
    }
    if (message.role !== undefined) {
      writer.uint32(26).string(message.role);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalNode {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalNode();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.groupValue = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.role = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ApprovalNode {
    return {
      type: isSet(object.type) ? approvalNode_TypeFromJSON(object.type) : 0,
      groupValue: isSet(object.groupValue) ? approvalNode_GroupValueFromJSON(object.groupValue) : undefined,
      role: isSet(object.role) ? String(object.role) : undefined,
    };
  },

  toJSON(message: ApprovalNode): unknown {
    const obj: any = {};
    message.type !== undefined && (obj.type = approvalNode_TypeToJSON(message.type));
    message.groupValue !== undefined &&
      (obj.groupValue = message.groupValue !== undefined
        ? approvalNode_GroupValueToJSON(message.groupValue)
        : undefined);
    message.role !== undefined && (obj.role = message.role);
    return obj;
  },

  create(base?: DeepPartial<ApprovalNode>): ApprovalNode {
    return ApprovalNode.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ApprovalNode>): ApprovalNode {
    const message = createBaseApprovalNode();
    message.type = object.type ?? 0;
    message.groupValue = object.groupValue ?? undefined;
    message.role = object.role ?? undefined;
    return message;
  },
};

export type ReviewServiceDefinition = typeof ReviewServiceDefinition;
export const ReviewServiceDefinition = {
  name: "ReviewService",
  fullName: "bytebase.v1.ReviewService",
  methods: {
    getReview: {
      name: "GetReview",
      requestType: GetReviewRequest,
      requestStream: false,
      responseType: Review,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              33,
              18,
              31,
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
              114,
              101,
              118,
              105,
              101,
              119,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    createReview: {
      name: "CreateReview",
      requestType: CreateReviewRequest,
      requestStream: false,
      responseType: Review,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([13, 112, 97, 114, 101, 110, 116, 44, 114, 101, 118, 105, 101, 119])],
          578365826: [
            new Uint8Array([
              41,
              58,
              6,
              114,
              101,
              118,
              105,
              101,
              119,
              34,
              31,
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
              114,
              101,
              118,
              105,
              101,
              119,
              115,
            ]),
          ],
        },
      },
    },
    listReviews: {
      name: "ListReviews",
      requestType: ListReviewsRequest,
      requestStream: false,
      responseType: ListReviewsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          578365826: [
            new Uint8Array([
              33,
              18,
              31,
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
              114,
              101,
              118,
              105,
              101,
              119,
              115,
            ]),
          ],
        },
      },
    },
    updateReview: {
      name: "UpdateReview",
      requestType: UpdateReviewRequest,
      requestStream: false,
      responseType: Review,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([18, 114, 101, 118, 105, 101, 119, 44, 117, 112, 100, 97, 116, 101, 95, 109, 97, 115, 107]),
          ],
          578365826: [
            new Uint8Array([
              48,
              58,
              6,
              114,
              101,
              118,
              105,
              101,
              119,
              50,
              38,
              47,
              118,
              49,
              47,
              123,
              114,
              101,
              118,
              105,
              101,
              119,
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
              114,
              101,
              118,
              105,
              101,
              119,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    batchUpdateReviews: {
      name: "BatchUpdateReviews",
      requestType: BatchUpdateReviewsRequest,
      requestStream: false,
      responseType: BatchUpdateReviewsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              48,
              58,
              1,
              42,
              34,
              43,
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
              114,
              101,
              118,
              105,
              101,
              119,
              115,
              58,
              98,
              97,
              116,
              99,
              104,
              85,
              112,
              100,
              97,
              116,
              101,
            ]),
          ],
        },
      },
    },
    approveReview: {
      name: "ApproveReview",
      requestType: ApproveReviewRequest,
      requestStream: false,
      responseType: Review,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              44,
              58,
              1,
              42,
              34,
              39,
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
              114,
              101,
              118,
              105,
              101,
              119,
              115,
              47,
              42,
              125,
              58,
              97,
              112,
              112,
              114,
              111,
              118,
              101,
            ]),
          ],
        },
      },
    },
    rejectReview: {
      name: "RejectReview",
      requestType: RejectReviewRequest,
      requestStream: false,
      responseType: Review,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              43,
              58,
              1,
              42,
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
              114,
              101,
              118,
              105,
              101,
              119,
              115,
              47,
              42,
              125,
              58,
              114,
              101,
              106,
              101,
              99,
              116,
            ]),
          ],
        },
      },
    },
  },
} as const;

export interface ReviewServiceImplementation<CallContextExt = {}> {
  getReview(request: GetReviewRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Review>>;
  createReview(request: CreateReviewRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Review>>;
  listReviews(
    request: ListReviewsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListReviewsResponse>>;
  updateReview(request: UpdateReviewRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Review>>;
  batchUpdateReviews(
    request: BatchUpdateReviewsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<BatchUpdateReviewsResponse>>;
  approveReview(request: ApproveReviewRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Review>>;
  rejectReview(request: RejectReviewRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Review>>;
}

export interface ReviewServiceClient<CallOptionsExt = {}> {
  getReview(request: DeepPartial<GetReviewRequest>, options?: CallOptions & CallOptionsExt): Promise<Review>;
  createReview(request: DeepPartial<CreateReviewRequest>, options?: CallOptions & CallOptionsExt): Promise<Review>;
  listReviews(
    request: DeepPartial<ListReviewsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListReviewsResponse>;
  updateReview(request: DeepPartial<UpdateReviewRequest>, options?: CallOptions & CallOptionsExt): Promise<Review>;
  batchUpdateReviews(
    request: DeepPartial<BatchUpdateReviewsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<BatchUpdateReviewsResponse>;
  approveReview(request: DeepPartial<ApproveReviewRequest>, options?: CallOptions & CallOptionsExt): Promise<Review>;
  rejectReview(request: DeepPartial<RejectReviewRequest>, options?: CallOptions & CallOptionsExt): Promise<Review>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function toTimestamp(date: Date): Timestamp {
  const seconds = date.getTime() / 1_000;
  const nanos = (date.getTime() % 1_000) * 1_000_000;
  return { seconds, nanos };
}

function fromTimestamp(t: Timestamp): Date {
  let millis = (t.seconds || 0) * 1_000;
  millis += (t.nanos || 0) / 1_000_000;
  return new Date(millis);
}

function fromJsonTimestamp(o: any): Date {
  if (o instanceof Date) {
    return o;
  } else if (typeof o === "string") {
    return new Date(o);
  } else {
    return fromTimestamp(Timestamp.fromJSON(o));
  }
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
