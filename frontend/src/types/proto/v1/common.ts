/* eslint-disable */

export const protobufPackage = "bytebase.v1";

export enum State {
  STATE_UNSPECIFIED = 0,
  STATE_ACTIVE = 1,
  STATE_DELETED = 2,
  UNRECOGNIZED = -1,
}

export function stateFromJSON(object: any): State {
  switch (object) {
    case 0:
    case "STATE_UNSPECIFIED":
      return State.STATE_UNSPECIFIED;
    case 1:
    case "STATE_ACTIVE":
      return State.STATE_ACTIVE;
    case 2:
    case "STATE_DELETED":
      return State.STATE_DELETED;
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
    case State.STATE_ACTIVE:
      return "STATE_ACTIVE";
    case State.STATE_DELETED:
      return "STATE_DELETED";
    case State.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
