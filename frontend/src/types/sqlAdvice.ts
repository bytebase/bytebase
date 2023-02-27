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
