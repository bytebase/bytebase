import { ChangeHistory } from "@/types/proto/v1/database_service";
import { Issue } from "@/types/proto/v1/issue_service";

export interface AffectedTable {
  schema: string;
  table: string;
  dropped: boolean;
}

export const EmptyAffectedTable: AffectedTable = {
  schema: "",
  table: "",
  dropped: false,
};

export interface ComposedChangeHistory extends ChangeHistory {
  issueEntity?: Issue;
}

export interface SearchChangeHistoriesParams {
  tables?: AffectedTable[];
  types?: string[];
}
