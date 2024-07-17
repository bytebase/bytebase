/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { GetIamPolicyRequest, IamPolicy, SetIamPolicyRequest } from "./iam_policy";

export const protobufPackage = "bytebase.v1";

export interface PatchIamPolicyRequest {
  /**
   * The name of the resource to get the IAM policy.
   * Format: workspaces/{workspace}
   */
  resource: string;
  /**
   * Specifies the principals requesting access for a Bytebase resource.
   * Format: user:{email}
   */
  member: string;
  /**
   * The roles that is assigned to the member.
   * Format: roles/{role}
   */
  roles: string[];
}

function createBasePatchIamPolicyRequest(): PatchIamPolicyRequest {
  return { resource: "", member: "", roles: [] };
}

export const PatchIamPolicyRequest = {
  encode(message: PatchIamPolicyRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.resource !== "") {
      writer.uint32(10).string(message.resource);
    }
    if (message.member !== "") {
      writer.uint32(18).string(message.member);
    }
    for (const v of message.roles) {
      writer.uint32(26).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PatchIamPolicyRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePatchIamPolicyRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.resource = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.member = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.roles.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PatchIamPolicyRequest {
    return {
      resource: isSet(object.resource) ? globalThis.String(object.resource) : "",
      member: isSet(object.member) ? globalThis.String(object.member) : "",
      roles: globalThis.Array.isArray(object?.roles) ? object.roles.map((e: any) => globalThis.String(e)) : [],
    };
  },

  toJSON(message: PatchIamPolicyRequest): unknown {
    const obj: any = {};
    if (message.resource !== "") {
      obj.resource = message.resource;
    }
    if (message.member !== "") {
      obj.member = message.member;
    }
    if (message.roles?.length) {
      obj.roles = message.roles;
    }
    return obj;
  },

  create(base?: DeepPartial<PatchIamPolicyRequest>): PatchIamPolicyRequest {
    return PatchIamPolicyRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<PatchIamPolicyRequest>): PatchIamPolicyRequest {
    const message = createBasePatchIamPolicyRequest();
    message.resource = object.resource ?? "";
    message.member = object.member ?? "";
    message.roles = object.roles?.map((e) => e) || [];
    return message;
  },
};

export type WorkspaceServiceDefinition = typeof WorkspaceServiceDefinition;
export const WorkspaceServiceDefinition = {
  name: "WorkspaceService",
  fullName: "bytebase.v1.WorkspaceService",
  methods: {
    getIamPolicy: {
      name: "GetIamPolicy",
      requestType: GetIamPolicyRequest,
      requestStream: false,
      responseType: IamPolicy,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              42,
              18,
              40,
              47,
              118,
              49,
              47,
              123,
              114,
              101,
              115,
              111,
              117,
              114,
              99,
              101,
              61,
              119,
              111,
              114,
              107,
              115,
              112,
              97,
              99,
              101,
              115,
              47,
              42,
              125,
              58,
              103,
              101,
              116,
              73,
              97,
              109,
              80,
              111,
              108,
              105,
              99,
              121,
            ]),
          ],
        },
      },
    },
    setIamPolicy: {
      name: "SetIamPolicy",
      requestType: SetIamPolicyRequest,
      requestStream: false,
      responseType: IamPolicy,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              45,
              58,
              1,
              42,
              34,
              40,
              47,
              118,
              49,
              47,
              123,
              114,
              101,
              115,
              111,
              117,
              114,
              99,
              101,
              61,
              119,
              111,
              114,
              107,
              115,
              112,
              97,
              99,
              101,
              115,
              47,
              42,
              125,
              58,
              115,
              101,
              116,
              73,
              97,
              109,
              80,
              111,
              108,
              105,
              99,
              121,
            ]),
          ],
        },
      },
    },
    patchIamPolicy: {
      name: "PatchIamPolicy",
      requestType: PatchIamPolicyRequest,
      requestStream: false,
      responseType: IamPolicy,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              47,
              58,
              1,
              42,
              50,
              42,
              47,
              118,
              49,
              47,
              123,
              114,
              101,
              115,
              111,
              117,
              114,
              99,
              101,
              61,
              119,
              111,
              114,
              107,
              115,
              112,
              97,
              99,
              101,
              115,
              47,
              42,
              125,
              58,
              112,
              97,
              116,
              99,
              104,
              73,
              97,
              109,
              80,
              111,
              108,
              105,
              99,
              121,
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

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
