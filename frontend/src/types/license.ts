import { PlanType } from "./plan";

export interface License {
  audience: string;
  instanceCount: number;
  expiresTs: number;
  plan: PlanType;
}
