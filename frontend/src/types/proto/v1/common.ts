/* eslint-disable */

export const protobufPackage = "bytebase.v1";

export enum State {
  STATE_UNSPECIFIED = 0,
  ACTIVE = 1,
  DELETED = 2,
  UNRECOGNIZED = -1,
}

export function stateFromJSON(object: any): State {
  switch (object) {
    case 0:
    case "STATE_UNSPECIFIED":
      return State.STATE_UNSPECIFIED;
    case 1:
    case "ACTIVE":
      return State.ACTIVE;
    case 2:
    case "DELETED":
      return State.DELETED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return State.UNRECOGNIZED;
  }
}

export function stateToJSON(object: State): string {
  switch (object) {
    case State.STATE_UNSPECIFIED:
      return "STATE_UNSPECIFIED";
    case State.ACTIVE:
      return "ACTIVE";
    case State.DELETED:
      return "DELETED";
    case State.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
