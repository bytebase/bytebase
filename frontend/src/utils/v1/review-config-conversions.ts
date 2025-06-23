import { fromJson, toJson } from "@bufbuild/protobuf";
import type { ReviewConfig as OldReviewConfig } from "@/types/proto/v1/review_config_service";
import { ReviewConfig as OldReviewConfigProto } from "@/types/proto/v1/review_config_service";
import type { ReviewConfig as NewReviewConfig } from "@/types/proto-es/v1/review_config_service_pb";
import { ReviewConfigSchema } from "@/types/proto-es/v1/review_config_service_pb";

// Convert old proto to proto-es
export const convertOldReviewConfigToNew = (oldReviewConfig: OldReviewConfig): NewReviewConfig => {
  // Use toJSON to convert old proto to JSON, then fromJson to convert to proto-es
  const json = OldReviewConfigProto.toJSON(oldReviewConfig) as any; // Type assertion needed due to proto type incompatibility
  return fromJson(ReviewConfigSchema, json);
};

// Convert proto-es to old proto
export const convertNewReviewConfigToOld = (newReviewConfig: NewReviewConfig): OldReviewConfig => {
  // Use toJson to convert proto-es to JSON, then fromJSON to convert to old proto
  const json = toJson(ReviewConfigSchema, newReviewConfig);
  return OldReviewConfigProto.fromJSON(json);
};