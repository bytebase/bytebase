import { ChangeHistory } from "@/types/proto/v1/database_service";
import { Issue } from "@/types/proto/v1/issue_service";

export interface Table {
  schema: string;
  table: string;
}

export interface AffectedTable extends Table {
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

export interface SearchChangeLogParams {
  tables?: Table[];
  types?: string[];
}
