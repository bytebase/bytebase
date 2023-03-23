/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { IssuePayloadApproval } from "./approval";

export const protobufPackage = "bytebase.store";

export interface IssuePayload {
  approval?: IssuePayloadApproval;
}

function createBaseIssuePayload(): IssuePayload {
  return { approval: undefined };
}

export const IssuePayload = {
  encode(message: IssuePayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.approval !== undefined) {
      IssuePayloadApproval.encode(message.approval, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IssuePayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIssuePayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.approval = IssuePayloadApproval.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): IssuePayload {
    return { approval: isSet(object.approval) ? IssuePayloadApproval.fromJSON(object.approval) : undefined };
  },

  toJSON(message: IssuePayload): unknown {
    const obj: any = {};
    message.approval !== undefined &&
      (obj.approval = message.approval ? IssuePayloadApproval.toJSON(message.approval) : undefined);
    return obj;
  },

  create(base?: DeepPartial<IssuePayload>): IssuePayload {
    return IssuePayload.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<IssuePayload>): IssuePayload {
    const message = createBaseIssuePayload();
    message.approval = (object.approval !== undefined && object.approval !== null)
      ? IssuePayloadApproval.fromPartial(object.approval)
      : undefined;
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
