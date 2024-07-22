/* eslint-disable */

export const protobufPackage = "bytebase.v1";

export enum AuthMethod {
  AUTH_METHOD_UNSPECIFIED = "AUTH_METHOD_UNSPECIFIED",
  /** IAM - IAM uses the standard IAM authorization check on the organizational resources. */
  IAM = "IAM",
  /** CUSTOM - Custom authorization method. */
  CUSTOM = "CUSTOM",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function authMethodFromJSON(object: any): AuthMethod {
  switch (object) {
    case 0:
    case "AUTH_METHOD_UNSPECIFIED":
      return AuthMethod.AUTH_METHOD_UNSPECIFIED;
    case 1:
    case "IAM":
      return AuthMethod.IAM;
    case 2:
    case "CUSTOM":
      return AuthMethod.CUSTOM;
    case -1:
    case "UNRECOGNIZED":
    default:
      return AuthMethod.UNRECOGNIZED;
  }
}

export function authMethodToJSON(object: AuthMethod): string {
  switch (object) {
    case AuthMethod.AUTH_METHOD_UNSPECIFIED:
      return "AUTH_METHOD_UNSPECIFIED";
    case AuthMethod.IAM:
      return "IAM";
    case AuthMethod.CUSTOM:
      return "CUSTOM";
    case AuthMethod.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function authMethodToNumber(object: AuthMethod): number {
  switch (object) {
    case AuthMethod.AUTH_METHOD_UNSPECIFIED:
      return 0;
    case AuthMethod.IAM:
      return 1;
    case AuthMethod.CUSTOM:
      return 2;
    case AuthMethod.UNRECOGNIZED:
    default:
      return -1;
  }
}
