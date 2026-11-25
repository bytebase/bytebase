import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { VCSType } from "@/types/proto-es/v1/common_pb";
import { Release_VCSSourceSchema } from "@/types/proto-es/v1/release_service_pb";
import { parseRepoCommit } from "./vcsSource";

const parse = (vcsType: VCSType, url: string) =>
  parseRepoCommit(create(Release_VCSSourceSchema, { vcsType, url }));

describe("parseRepoCommit", () => {
  test.each([
    [
      "GitHub",
      VCSType.GITHUB,
      "https://github.com/example/app",
      "/commit/",
      "example/app",
    ],
    [
      "GitLab",
      VCSType.GITLAB,
      "https://gitlab.example.com/group/app",
      "/-/commit/",
      "group/app",
    ],
    [
      "a GitLab subgroup",
      VCSType.GITLAB,
      "https://gitlab.example.com/acme/payments/api",
      "/-/commit/",
      "acme/payments/api",
    ],
    [
      "Bitbucket",
      VCSType.BITBUCKET,
      "https://bitbucket.org/example/app",
      "/commits/",
      "example/app",
    ],
    [
      "Azure DevOps",
      VCSType.AZURE_DEVOPS,
      "https://dev.azure.com/example/Payments/_git/api",
      "/commit/",
      "Payments/api",
    ],
    [
      "an Azure DevOps visualstudio.com host",
      VCSType.AZURE_DEVOPS,
      "https://example.visualstudio.com/Payments/_git/api",
      "/commit/",
      "Payments/api",
    ],
    [
      "an Azure DevOps project with a space",
      VCSType.AZURE_DEVOPS,
      "https://dev.azure.com/example/Shop%20Team/_git/api",
      "/commit/",
      "Shop Team/api",
    ],
  ])(
    "names %s's repository as its platform does",
    (_, vcsType, repoUrl, commitPath, repo) => {
      expect(parse(vcsType, `${repoUrl}${commitPath}0a1b2c3d4e5f`)).toEqual({
        repoUrl,
        repo,
        sha: "0a1b2c3d4e5f",
      });
    }
  );

  test("gives nothing for a commit URL with no repository path", () => {
    expect(
      parse(VCSType.GITHUB, "https://github.com/commit/0a1b2c3d4e5f")
    ).toBeUndefined();
  });
});
