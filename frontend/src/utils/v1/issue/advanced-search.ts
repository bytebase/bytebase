import { UIIssueFilterScopeId } from "./ui-filter";

export type SearchScopeId =
  | "project"
  | "instance"
  | "database"
  | "type"
  | "creator"
  | "assignee"
  | "subscriber"
  | UIIssueFilterScopeId;

export interface SearchParams {
  query: string;
  scopes: {
    id: SearchScopeId;
    value: string;
  }[];
}
