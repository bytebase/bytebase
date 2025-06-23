import { fromJson, toJson } from "@bufbuild/protobuf";
import type { Risk as OldRisk } from "@/types/proto/v1/risk_service";
import { Risk as OldRiskProto } from "@/types/proto/v1/risk_service";
import type { Risk as NewRisk } from "@/types/proto-es/v1/risk_service_pb";
import { RiskSchema } from "@/types/proto-es/v1/risk_service_pb";
import { Risk_Source as OldRiskSource } from "@/types/proto/v1/risk_service";
import { Risk_Source as NewRiskSource } from "@/types/proto-es/v1/risk_service_pb";

// Convert old proto to proto-es
export const convertOldRiskToNew = (oldRisk: OldRisk): NewRisk => {
  // Use toJSON to convert old proto to JSON, then fromJson to convert to proto-es
  const json = OldRiskProto.toJSON(oldRisk) as any; // Type assertion needed due to proto type incompatibility
  return fromJson(RiskSchema, json);
};

// Convert proto-es to old proto
export const convertNewRiskToOld = (newRisk: NewRisk): OldRisk => {
  // Use toJson to convert proto-es to JSON, then fromJSON to convert to old proto
  const json = toJson(RiskSchema, newRisk);
  return OldRiskProto.fromJSON(json);
};

// Convert old Risk_Source enum to new (string to numeric)
export const convertOldRiskSourceToNew = (oldSource: OldRiskSource): NewRiskSource => {
  const mapping: Record<OldRiskSource, NewRiskSource> = {
    [OldRiskSource.SOURCE_UNSPECIFIED]: NewRiskSource.SOURCE_UNSPECIFIED,
    [OldRiskSource.DDL]: NewRiskSource.DDL,
    [OldRiskSource.DML]: NewRiskSource.DML,
    [OldRiskSource.CREATE_DATABASE]: NewRiskSource.CREATE_DATABASE,
    [OldRiskSource.DATA_EXPORT]: NewRiskSource.DATA_EXPORT,
    [OldRiskSource.REQUEST_ROLE]: NewRiskSource.REQUEST_ROLE,
    [OldRiskSource.UNRECOGNIZED]: NewRiskSource.SOURCE_UNSPECIFIED,
  };
  return mapping[oldSource] ?? NewRiskSource.SOURCE_UNSPECIFIED;
};

// Convert new Risk_Source enum to old (numeric to string)
export const convertNewRiskSourceToOld = (newSource: NewRiskSource): OldRiskSource => {
  const mapping: Record<NewRiskSource, OldRiskSource> = {
    [NewRiskSource.SOURCE_UNSPECIFIED]: OldRiskSource.SOURCE_UNSPECIFIED,
    [NewRiskSource.DDL]: OldRiskSource.DDL,
    [NewRiskSource.DML]: OldRiskSource.DML,
    [NewRiskSource.CREATE_DATABASE]: OldRiskSource.CREATE_DATABASE,
    [NewRiskSource.DATA_EXPORT]: OldRiskSource.DATA_EXPORT,
    [NewRiskSource.REQUEST_ROLE]: OldRiskSource.REQUEST_ROLE,
  };
  return mapping[newSource] ?? OldRiskSource.UNRECOGNIZED;
};