import { RiskLevel } from "./proto-es/v1/common_pb";

export const PresetRiskLevel = {
  HIGH: RiskLevel.HIGH,
  MODERATE: RiskLevel.MODERATE,
  LOW: RiskLevel.LOW,
};

export const PresetRiskLevelList = [
  { name: "HIGH", level: PresetRiskLevel.HIGH },
  { name: "MODERATE", level: PresetRiskLevel.MODERATE },
  { name: "LOW", level: PresetRiskLevel.LOW },
];

export const DEFAULT_RISK_LEVEL = RiskLevel.RISK_LEVEL_UNSPECIFIED;
