import { computed } from "vue";
import { Risk_Source } from "./proto-es/v1/risk_service_pb";

export const PresetRiskLevel = {
  HIGH: 300,
  MODERATE: 200,
  LOW: 100,
};
export const PresetRiskLevelList = [
  { name: "HIGH", level: PresetRiskLevel.HIGH },
  { name: "MODERATE", level: PresetRiskLevel.MODERATE },
  { name: "LOW", level: PresetRiskLevel.LOW },
];

export const DEFAULT_RISK_LEVEL = 0;

export const useSupportedSourceList = () => {
  return computed(() => {
    return [
      Risk_Source.DDL,
      Risk_Source.DML,
      Risk_Source.CREATE_DATABASE,
      Risk_Source.DATA_EXPORT,
      Risk_Source.REQUEST_ROLE,
    ];
  });
};
