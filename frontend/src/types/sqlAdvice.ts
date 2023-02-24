import { SQLAdviceCode } from "./sqlAdviceCode";

export * from "./sqlAdviceCode";

export type AdviceStatus = "SUCCESS" | "WARN" | "ERROR";

export type Advice = {
  status: AdviceStatus;
  code: SQLAdviceCode;
  title: string;
  content: string;
  line: number;
};

export type SQLResultSet = {
  // [columnNames: string[], types: string[], data: any[][], sensitive?: boolean[]]
  data: [string[], string[], any[][], boolean[]];
  error: string;
  adviceList: Advice[];
};
