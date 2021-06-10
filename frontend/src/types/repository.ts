import { RepositoryId } from "./id";

export type Repository = {
  id: RepositoryId;
  // e.g. sample-project
  name: string;
  // e.g. bytebase/sample-project
  fullPath: string;
  // e.g. http://gitlab.bytebase.com/bytebase/sample-project
  webURL: string;
};
