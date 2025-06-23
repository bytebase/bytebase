import { fromJson, toJson } from "@bufbuild/protobuf";
import type {
  QueryRequest as OldQueryRequest,
  QueryResponse as OldQueryResponse,
  ExportRequest as OldExportRequest,
  ExportResponse as OldExportResponse,
  CheckRequest as OldCheckRequest,
  CheckResponse as OldCheckResponse,
  DiffMetadataRequest as OldDiffMetadataRequest,
  DiffMetadataResponse as OldDiffMetadataResponse,
  SearchQueryHistoriesRequest as OldSearchQueryHistoriesRequest,
  SearchQueryHistoriesResponse as OldSearchQueryHistoriesResponse,
  AICompletionRequest as OldAICompletionRequest,
  AICompletionResponse as OldAICompletionResponse,
} from "@/types/proto/v1/sql_service";
import {
  QueryRequest as OldQueryRequestProto,
  QueryResponse as OldQueryResponseProto,
  ExportRequest as OldExportRequestProto,
  ExportResponse as OldExportResponseProto,
  CheckRequest as OldCheckRequestProto,
  CheckResponse as OldCheckResponseProto,
  DiffMetadataRequest as OldDiffMetadataRequestProto,
  DiffMetadataResponse as OldDiffMetadataResponseProto,
  SearchQueryHistoriesRequest as OldSearchQueryHistoriesRequestProto,
  SearchQueryHistoriesResponse as OldSearchQueryHistoriesResponseProto,
  AICompletionRequest as OldAICompletionRequestProto,
  AICompletionResponse as OldAICompletionResponseProto,
} from "@/types/proto/v1/sql_service";
import type {
  QueryRequest as NewQueryRequest,
  QueryResponse as NewQueryResponse,
  ExportRequest as NewExportRequest,
  ExportResponse as NewExportResponse,
  CheckRequest as NewCheckRequest,
  CheckResponse as NewCheckResponse,
  DiffMetadataRequest as NewDiffMetadataRequest,
  DiffMetadataResponse as NewDiffMetadataResponse,
  SearchQueryHistoriesRequest as NewSearchQueryHistoriesRequest,
  SearchQueryHistoriesResponse as NewSearchQueryHistoriesResponse,
  AICompletionRequest as NewAICompletionRequest,
  AICompletionResponse as NewAICompletionResponse,
} from "@/types/proto-es/v1/sql_service_pb";
import {
  QueryRequestSchema,
  QueryResponseSchema,
  ExportRequestSchema,
  ExportResponseSchema,
  CheckRequestSchema,
  CheckResponseSchema,
  DiffMetadataRequestSchema,
  DiffMetadataResponseSchema,
  SearchQueryHistoriesRequestSchema,
  SearchQueryHistoriesResponseSchema,
  AICompletionRequestSchema,
  AICompletionResponseSchema,
} from "@/types/proto-es/v1/sql_service_pb";

// Convert old QueryRequest to proto-es
export const convertOldQueryRequestToNew = (oldRequest: OldQueryRequest): NewQueryRequest => {
  const json = OldQueryRequestProto.toJSON(oldRequest) as any;
  return fromJson(QueryRequestSchema, json);
};

// Convert proto-es QueryResponse to old
export const convertNewQueryResponseToOld = (newResponse: NewQueryResponse): OldQueryResponse => {
  const json = toJson(QueryResponseSchema, newResponse);
  return OldQueryResponseProto.fromJSON(json);
};

// Convert old ExportRequest to proto-es
export const convertOldExportRequestToNew = (oldRequest: OldExportRequest): NewExportRequest => {
  const json = OldExportRequestProto.toJSON(oldRequest) as any;
  return fromJson(ExportRequestSchema, json);
};

// Convert proto-es ExportResponse to old
export const convertNewExportResponseToOld = (newResponse: NewExportResponse): OldExportResponse => {
  const json = toJson(ExportResponseSchema, newResponse);
  return OldExportResponseProto.fromJSON(json);
};

// Convert old CheckRequest to proto-es
export const convertOldCheckRequestToNew = (oldRequest: OldCheckRequest): NewCheckRequest => {
  const json = OldCheckRequestProto.toJSON(oldRequest) as any;
  return fromJson(CheckRequestSchema, json);
};

// Convert proto-es CheckResponse to old
export const convertNewCheckResponseToOld = (newResponse: NewCheckResponse): OldCheckResponse => {
  const json = toJson(CheckResponseSchema, newResponse);
  return OldCheckResponseProto.fromJSON(json);
};

// Convert old DiffMetadataRequest to proto-es
export const convertOldDiffMetadataRequestToNew = (oldRequest: OldDiffMetadataRequest): NewDiffMetadataRequest => {
  const json = OldDiffMetadataRequestProto.toJSON(oldRequest) as any;
  return fromJson(DiffMetadataRequestSchema, json);
};

// Convert proto-es DiffMetadataResponse to old
export const convertNewDiffMetadataResponseToOld = (newResponse: NewDiffMetadataResponse): OldDiffMetadataResponse => {
  const json = toJson(DiffMetadataResponseSchema, newResponse);
  return OldDiffMetadataResponseProto.fromJSON(json);
};

// Convert old SearchQueryHistoriesRequest to proto-es
export const convertOldSearchQueryHistoriesRequestToNew = (oldRequest: OldSearchQueryHistoriesRequest): NewSearchQueryHistoriesRequest => {
  const json = OldSearchQueryHistoriesRequestProto.toJSON(oldRequest) as any;
  return fromJson(SearchQueryHistoriesRequestSchema, json);
};

// Convert proto-es SearchQueryHistoriesResponse to old
export const convertNewSearchQueryHistoriesResponseToOld = (newResponse: NewSearchQueryHistoriesResponse): OldSearchQueryHistoriesResponse => {
  const json = toJson(SearchQueryHistoriesResponseSchema, newResponse);
  return OldSearchQueryHistoriesResponseProto.fromJSON(json);
};

// Convert old AICompletionRequest to proto-es
export const convertOldAICompletionRequestToNew = (oldRequest: OldAICompletionRequest): NewAICompletionRequest => {
  const json = OldAICompletionRequestProto.toJSON(oldRequest) as any;
  return fromJson(AICompletionRequestSchema, json);
};

// Convert proto-es AICompletionResponse to old
export const convertNewAICompletionResponseToOld = (newResponse: NewAICompletionResponse): OldAICompletionResponse => {
  const json = toJson(AICompletionResponseSchema, newResponse);
  return OldAICompletionResponseProto.fromJSON(json);
};