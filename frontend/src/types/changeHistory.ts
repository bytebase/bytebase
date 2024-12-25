import type { Table } from "./changelog";

export interface SearchChangeHistoriesParams {
  tables?: Table[];
  types?: string[];
}
