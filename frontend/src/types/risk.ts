import { Risk_Source } from "./proto/v1/risk_service";

export const PresetRiskLevelList = [
  { name: "HIGH", level: 300 },
  { name: "MODERATE", level: 200 },
  { name: "LOW", level: 100 },
];

export const SupportedSourceList = [
  Risk_Source.DDL,
  Risk_Source.DML,
  Risk_Source.CREATE_DATABASE,
];
