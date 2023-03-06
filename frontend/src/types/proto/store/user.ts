/* eslint-disable */
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

/** MFAConfig is the MFA configuration for a user. */
export interface MFAConfig {
  /** The otp_secret is the secret key used to validate the OTP code. */
  otpSecret: string;
  /** The recovery_codes are the codes that can be used to recover the account. */
  recoveryCodes: string[];
}

function createBaseMFAConfig(): MFAConfig {
  return { otpSecret: "", recoveryCodes: [] };
}

export const MFAConfig = {
  encode(message: MFAConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.otpSecret !== "") {
      writer.uint32(10).string(message.otpSecret);
    }
    for (const v of message.recoveryCodes) {
      writer.uint32(18).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MFAConfig {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMFAConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.otpSecret = reader.string();
          break;
        case 2:
          message.recoveryCodes.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MFAConfig {
    return {
      otpSecret: isSet(object.otpSecret) ? String(object.otpSecret) : "",
      recoveryCodes: Array.isArray(object?.recoveryCodes) ? object.recoveryCodes.map((e: any) => String(e)) : [],
    };
  },

  toJSON(message: MFAConfig): unknown {
    const obj: any = {};
    message.otpSecret !== undefined && (obj.otpSecret = message.otpSecret);
    if (message.recoveryCodes) {
      obj.recoveryCodes = message.recoveryCodes.map((e) => e);
    } else {
      obj.recoveryCodes = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<MFAConfig>): MFAConfig {
    const message = createBaseMFAConfig();
    message.otpSecret = object.otpSecret ?? "";
    message.recoveryCodes = object.recoveryCodes?.map((e) => e) || [];
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
