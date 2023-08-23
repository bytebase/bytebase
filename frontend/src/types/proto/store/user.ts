/* eslint-disable */
import * as _m0 from "protobufjs/minimal";

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
      otpSecret: isSet(object.otpSecret) ? String(object.otpSecret) : "",
      tempOtpSecret: isSet(object.tempOtpSecret) ? String(object.tempOtpSecret) : "",
      recoveryCodes: Array.isArray(object?.recoveryCodes) ? object.recoveryCodes.map((e: any) => String(e)) : [],
      tempRecoveryCodes: Array.isArray(object?.tempRecoveryCodes)
        ? object.tempRecoveryCodes.map((e: any) => String(e))
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

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
