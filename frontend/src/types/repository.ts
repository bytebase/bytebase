import isEmpty from "lodash-es/isEmpty";
import { RepositoryId, VCSId } from "./id";
import { Principal } from "./principal";
import { Project } from "./project";
import { VCS } from "./vcs";

export type Repository = {
  id: RepositoryId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Related fields
  vcs: VCS;
  project: Project;

  // Domain specific fields
  // e.g. sample-project
  name: string;
  // e.g. bytebase/sample-project
  fullPath: string;
  // e.g. http://gitlab.bytebase.com/bytebase/sample-project
  webUrl: string;
  baseDirectory: string;
  branchFilter: string;
  filePathTemplate: string;
  schemaPathTemplate: string;
  sheetPathTemplate: string;
  // e.g. In GitLab, this is the corresponding project id.
  externalId: string;
};

export type RepositoryCreate = {
  // Related fields
  vcsId: VCSId;

  // Domain specific fields
  name: string;
  fullPath: string;
  webUrl: string;
  branchFilter: string;
  baseDirectory: string;
  filePathTemplate: string;
  schemaPathTemplate: string;
  sheetPathTemplate: string;
  externalId: string;
  accessToken: string;
  expiresTs: number;
  refreshToken: string;
};

export type RepositoryPatch = {
  baseDirectory?: string;
  branchFilter?: string;
  filePathTemplate?: string;
  schemaPathTemplate?: string;
  sheetPathTemplate?: string;
};

export type RepositoryConfig = {
  baseDirectory: string;
  branchFilter: string;
  filePathTemplate: string;
  schemaPathTemplate: string;
  sheetPathTemplate: string;
};

export type ExternalRepositoryInfo = {
  // e.g. In GitLab, this is the corresponding project id. e.g. 123
  externalId: string;
  // e.g. sample-project
  name: string;
  // e.g. bytebase/sample-project
  fullPath: string;
  // e.g. http://gitlab.bytebase.com/bytebase/sample-project
  webUrl: string;
};

type WebUrlReplaceParams = {
  DB_NAME?: string;
  VERSION?: string;
  TYPE?: "migrate" | "data";
  ENV_NAME?: string;
};

export function baseDirectoryWebUrl(
  repository: Repository,
  params: WebUrlReplaceParams = {}
): string {
  let url = "";
  if (repository.vcs.type == "GITLAB_SELF_HOST") {
    url = `${repository.webUrl}/-/tree/${repository.branchFilter}`;
    if (!isEmpty(repository.baseDirectory)) {
      url += `/${repository.baseDirectory}`;
    }
  } else if (repository.vcs.type == "GITHUB_COM") {
    url = `${repository.webUrl}/tree/${repository.branchFilter}`;
    if (!isEmpty(repository.baseDirectory)) {
      url += `/${repository.baseDirectory}`;
    }
  }
  if (url) {
    // Replace the patterns in the filePathTemplate if possible.
    const segments = repository.filePathTemplate.split("/");
    segments.pop(); // exclude the last one, it's the filename.
    // Try to replace the segments from left to right.
    // Once we meet a "dynamic" segment which has a pattern that cannot be replaced
    // we won't push it, either the segments behind it.
    // E.g., the filePathTemplate is
    // configure/{{ENV_NAME}}/20220707-wechat/{{TYPE}}/**/**/**/{{DB_NAME}}__{{VERSION}}__{{DESCRIPTION}}.sql
    /**
      The segments are
        - configure
        - {{ENV_NAME}}
        - 20220707-wechat
        - {{TYPE}}
        - **
        - **
        - **
      When
        - ENV_NAME=dev
        - TYPE=migrate
      we are confident enough that the path will be started with
      "/configure/dev/20220707-wechat/migrate"
      That's our best effort.
     */
    for (let i = 0; i < segments.length; i++) {
      const segment = segments[i];
      const replaced = replaceParams(segment, params);
      if (replaced.match(/[*{}]/)) {
        // Still remained some patterns cannot be replaced in the value.
        break;
      }
      url += `/${replaced}`;
    }

    return url;
  }

  // Fallback for other types of VCS.
  // Shouldn't reach this line.
  return repository.webUrl;
}

const replaceParams = (
  template: string,
  params: WebUrlReplaceParams = {}
): string => {
  let replaced = template;
  Object.keys(params).forEach((key) => {
    const pattern = `{{${key}}}`;
    const value = params[key as keyof WebUrlReplaceParams];
    if (value) {
      replaced = replaced.replaceAll(pattern, value);
    }
  });
  return replaced;
};
