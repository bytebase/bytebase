/* eslint-disable */

export const protobufPackage = "bytebase.v1";

export enum VcsType {
  VCS_TYPE_UNSPECIFIED = 0,
  GITLAB = 1,
  GITHUB = 2,
  BITBUCKET = 3,
  UNRECOGNIZED = -1,
}

export function vcsTypeFromJSON(object: any): VcsType {
  switch (object) {
    case 0:
    case "VCS_TYPE_UNSPECIFIED":
      return VcsType.VCS_TYPE_UNSPECIFIED;
    case 1:
    case "GITLAB":
      return VcsType.GITLAB;
    case 2:
    case "GITHUB":
      return VcsType.GITHUB;
    case 3:
    case "BITBUCKET":
      return VcsType.BITBUCKET;
    case -1:
    case "UNRECOGNIZED":
    default:
      return VcsType.UNRECOGNIZED;
  }
}

export function vcsTypeToJSON(object: VcsType): string {
  switch (object) {
    case VcsType.VCS_TYPE_UNSPECIFIED:
      return "VCS_TYPE_UNSPECIFIED";
    case VcsType.GITLAB:
      return "GITLAB";
    case VcsType.GITHUB:
      return "GITHUB";
    case VcsType.BITBUCKET:
      return "BITBUCKET";
    case VcsType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
