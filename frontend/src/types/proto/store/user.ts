/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.store";

/** MFAConfig is the MFA configuration for a user. */
export interface MFAConfig {
  /** The otp_secret is the secret key used to validate the OTP code. */
  otpSecret: string;
  /** The temp_otp_secret is the temporary secret key used to validate the OTP code and will replace the otp_secret in two phase commits. */
  tempOtpSecret: string;
  /** The recovery_codes are the codes that can be used to recover the account. */
  recoveryCodes: string[];
  /** The temp_recovery_codes are the temporary codes that will replace the recovery_codes in two phase commits. */
  tempRecoveryCodes: string[];
}

export interface UserProfile {
  lastLoginTime: Date | undefined;
  lastChangePasswordTime:
    | Date
    | undefined;
  /** source means where the user comes from. For now we support Entra ID SCIM sync, so the source could be Entra ID. */
  source: string;
}

function createBaseMFAConfig(): MFAConfig {
  return { otpSecret: "", tempOtpSecret: "", recoveryCodes: [], tempRecoveryCodes: [] };
}

export const MFAConfig = {
  encode(message: MFAConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.otpSecret !== "") {
      writer.uint32(10).string(message.otpSecret);
    }
    if (message.tempOtpSecret !== "") {
      writer.uint32(18).string(message.tempOtpSecret);
    }
    for (const v of message.recoveryCodes) {
      writer.uint32(26).string(v!);
    }
    for (const v of message.tempRecoveryCodes) {
      writer.uint32(34).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MFAConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMFAConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.otpSecret = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.tempOtpSecret = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.recoveryCodes.push(reader.string());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.tempRecoveryCodes.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MFAConfig {
    return {
      otpSecret: isSet(object.otpSecret) ? globalThis.String(object.otpSecret) : "",
      tempOtpSecret: isSet(object.tempOtpSecret) ? globalThis.String(object.tempOtpSecret) : "",
      recoveryCodes: globalThis.Array.isArray(object?.recoveryCodes)
        ? object.recoveryCodes.map((e: any) => globalThis.String(e))
        : [],
      tempRecoveryCodes: globalThis.Array.isArray(object?.tempRecoveryCodes)
        ? object.tempRecoveryCodes.map((e: any) => globalThis.String(e))
        : [],
    };
  },

  toJSON(message: MFAConfig): unknown {
    const obj: any = {};
    if (message.otpSecret !== "") {
      obj.otpSecret = message.otpSecret;
    }
    if (message.tempOtpSecret !== "") {
      obj.tempOtpSecret = message.tempOtpSecret;
    }
    if (message.recoveryCodes?.length) {
      obj.recoveryCodes = message.recoveryCodes;
    }
    if (message.tempRecoveryCodes?.length) {
      obj.tempRecoveryCodes = message.tempRecoveryCodes;
    }
    return obj;
  },

  create(base?: DeepPartial<MFAConfig>): MFAConfig {
    return MFAConfig.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<MFAConfig>): MFAConfig {
    const message = createBaseMFAConfig();
    message.otpSecret = object.otpSecret ?? "";
    message.tempOtpSecret = object.tempOtpSecret ?? "";
    message.recoveryCodes = object.recoveryCodes?.map((e) => e) || [];
    message.tempRecoveryCodes = object.tempRecoveryCodes?.map((e) => e) || [];
    return message;
  },
};

function createBaseUserProfile(): UserProfile {
  return { lastLoginTime: undefined, lastChangePasswordTime: undefined, source: "" };
}

export const UserProfile = {
  encode(message: UserProfile, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.lastLoginTime !== undefined) {
      Timestamp.encode(toTimestamp(message.lastLoginTime), writer.uint32(10).fork()).ldelim();
    }
    if (message.lastChangePasswordTime !== undefined) {
      Timestamp.encode(toTimestamp(message.lastChangePasswordTime), writer.uint32(18).fork()).ldelim();
    }
    if (message.source !== "") {
      writer.uint32(26).string(message.source);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UserProfile {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUserProfile();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.lastLoginTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.lastChangePasswordTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.source = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UserProfile {
    return {
      lastLoginTime: isSet(object.lastLoginTime) ? fromJsonTimestamp(object.lastLoginTime) : undefined,
      lastChangePasswordTime: isSet(object.lastChangePasswordTime)
        ? fromJsonTimestamp(object.lastChangePasswordTime)
        : undefined,
      source: isSet(object.source) ? globalThis.String(object.source) : "",
    };
  },

  toJSON(message: UserProfile): unknown {
    const obj: any = {};
    if (message.lastLoginTime !== undefined) {
      obj.lastLoginTime = message.lastLoginTime.toISOString();
    }
    if (message.lastChangePasswordTime !== undefined) {
      obj.lastChangePasswordTime = message.lastChangePasswordTime.toISOString();
    }
    if (message.source !== "") {
      obj.source = message.source;
    }
    return obj;
  },

  create(base?: DeepPartial<UserProfile>): UserProfile {
    return UserProfile.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UserProfile>): UserProfile {
    const message = createBaseUserProfile();
    message.lastLoginTime = object.lastLoginTime ?? undefined;
    message.lastChangePasswordTime = object.lastChangePasswordTime ?? undefined;
    message.source = object.source ?? "";
    return message;
  },
};

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
