// @generated by protoc-gen-es v2.5.2
// @generated from file v1/subscription_service.proto (package bytebase.v1, syntax proto3)
/* eslint-disable */

import { enumDesc, fileDesc, messageDesc, serviceDesc, tsEnum } from "@bufbuild/protobuf/codegenv2";
import { file_google_api_annotations } from "../google/api/annotations_pb";
import { file_google_api_client } from "../google/api/client_pb";
import { file_google_api_field_behavior } from "../google/api/field_behavior_pb";
import { file_google_protobuf_timestamp } from "@bufbuild/protobuf/wkt";
import { file_v1_annotation } from "./annotation_pb";

/**
 * Describes the file v1/subscription_service.proto.
 */
export const file_v1_subscription_service = /*@__PURE__*/
  fileDesc("Ch12MS9zdWJzY3JpcHRpb25fc2VydmljZS5wcm90bxILYnl0ZWJhc2UudjEiGAoWR2V0U3Vic2NyaXB0aW9uUmVxdWVzdCIsChlVcGRhdGVTdWJzY3JpcHRpb25SZXF1ZXN0Eg8KB2xpY2Vuc2UYASABKAki2QEKDFN1YnNjcmlwdGlvbhIYCgpzZWF0X2NvdW50GAEgASgFQgTiQQEDEhwKDmluc3RhbmNlX2NvdW50GAIgASgFQgTiQQEDEjYKDGV4cGlyZXNfdGltZRgDIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBCBOJBAQMSKQoEcGxhbhgEIAEoDjIVLmJ5dGViYXNlLnYxLlBsYW5UeXBlQgTiQQEDEhYKCHRyaWFsaW5nGAUgASgIQgTiQQEDEhYKCG9yZ19uYW1lGAYgASgJQgTiQQEDIm4KClBsYW5Db25maWcSKwoFcGxhbnMYASADKAsyHC5ieXRlYmFzZS52MS5QbGFuTGltaXRDb25maWcSMwoRaW5zdGFuY2VfZmVhdHVyZXMYAiADKA4yGC5ieXRlYmFzZS52MS5QbGFuRmVhdHVyZSKeAQoPUGxhbkxpbWl0Q29uZmlnEiMKBHR5cGUYASABKA4yFS5ieXRlYmFzZS52MS5QbGFuVHlwZRIeChZtYXhpbXVtX2luc3RhbmNlX2NvdW50GAIgASgFEhoKEm1heGltdW1fc2VhdF9jb3VudBgDIAEoBRIqCghmZWF0dXJlcxgEIAMoDjIYLmJ5dGViYXNlLnYxLlBsYW5GZWF0dXJlKkkKCFBsYW5UeXBlEhkKFVBMQU5fVFlQRV9VTlNQRUNJRklFRBAAEggKBEZSRUUQARIICgRURUFNEAISDgoKRU5URVJQUklTRRADKu4SCgtQbGFuRmVhdHVyZRIXChNGRUFUVVJFX1VOU1BFQ0lGSUVEEAASGwoXRkVBVFVSRV9EQVRBQkFTRV9DSEFOR0UQARIsCihGRUFUVVJFX0dJVF9CQVNFRF9TQ0hFTUFfVkVSU0lPTl9DT05UUk9MEAISKAokRkVBVFVSRV9ERUNMQVJBVElWRV9TQ0hFTUFfTUlHUkFUSU9OEAMSIwofRkVBVFVSRV9DT01QQVJFX0FORF9TWU5DX1NDSEVNQRAEEiAKHEZFQVRVUkVfT05MSU5FX1NDSEVNQV9DSEFOR0UQBRIlCiFGRUFUVVJFX1BSRV9ERVBMT1lNRU5UX1NRTF9SRVZJRVcQBhIwCixGRUFUVVJFX0FVVE9NQVRJQ19CQUNLVVBfQkVGT1JFX0RBVEFfQ0hBTkdFUxAHEiMKH0ZFQVRVUkVfT05FX0NMSUNLX0RBVEFfUk9MTEJBQ0sQCBIoCiRGRUFUVVJFX01VTFRJX0RBVEFCQVNFX0JBVENIX0NIQU5HRVMQCRIuCipGRUFUVVJFX1BST0dSRVNTSVZFX0VOVklST05NRU5UX0RFUExPWU1FTlQQChIiCh5GRUFUVVJFX1NDSEVEVUxFRF9ST0xMT1VUX1RJTUUQCxIeChpGRUFUVVJFX0RBVEFCQVNFX0NIQU5HRUxPRxAMEiIKHkZFQVRVUkVfU0NIRU1BX0RSSUZUX0RFVEVDVElPThANEhYKEkZFQVRVUkVfQ0hBTkdFTElTVBAOEhsKF0ZFQVRVUkVfU0NIRU1BX1RFTVBMQVRFEA8SGgoWRkVBVFVSRV9ST0xMT1VUX1BPTElDWRAQEiAKHEZFQVRVUkVfV0VCX0JBU0VEX1NRTF9FRElUT1IQERIhCh1GRUFUVVJFX1NRTF9FRElUT1JfQURNSU5fTU9ERRASEiMKH0ZFQVRVUkVfTkFUVVJBTF9MQU5HVUFHRV9UT19TUUwQExIgChxGRUFUVVJFX0FJX1FVRVJZX0VYUExBTkFUSU9OEBQSIAocRkVBVFVSRV9BSV9RVUVSWV9TVUdHRVNUSU9OUxAVEhkKFUZFQVRVUkVfQVVUT19DT01QTEVURRAWEhoKFkZFQVRVUkVfU0NIRU1BX0RJQUdSQU0QFxIZChVGRUFUVVJFX1NDSEVNQV9FRElUT1IQGBIXChNGRUFUVVJFX0RBVEFfRVhQT1JUEBkSGQoVRkVBVFVSRV9RVUVSWV9ISVNUT1JZEBoSKAokRkVBVFVSRV9TQVZFRF9BTkRfU0hBUkVEX1NRTF9TQ1JJUFRTEBsSFwoTRkVBVFVSRV9CQVRDSF9RVUVSWRAcEikKJUZFQVRVUkVfSU5TVEFOQ0VfUkVBRF9PTkxZX0NPTk5FQ1RJT04QHRIYChRGRUFUVVJFX1FVRVJZX1BPTElDWRAeEiEKHUZFQVRVUkVfUkVTVFJJQ1RfQ09QWUlOR19EQVRBEB8SDwoLRkVBVFVSRV9JQU0QIBIjCh9GRUFUVVJFX0lOU1RBTkNFX1NTTF9DT05ORUNUSU9OECESLworRkVBVFVSRV9JTlNUQU5DRV9DT05ORUNUSU9OX09WRVJfU1NIX1RVTk5FTBAiEjIKLkZFQVRVUkVfSU5TVEFOQ0VfQ09OTkVDVElPTl9JQU1fQVVUSEVOVElDQVRJT04QIxIhCh1GRUFUVVJFX0dPT0dMRV9BTkRfR0lUSFVCX1NTTxAkEhcKE0ZFQVRVUkVfVVNFUl9HUk9VUFMQJRIoCiRGRUFUVVJFX0RJU0FMTE9XX1NFTEZfU0VSVklDRV9TSUdOVVAQJhIlCiFGRUFUVVJFX0RBVEFCQVNFX1NFQ1JFVF9WQVJJQUJMRVMQJxIlCiFGRUFUVVJFX0NVU1RPTV9JTlNUQU5DRV9TWU5DX1RJTUUQKBIsCihGRUFUVVJFX0NVU1RPTV9JTlNUQU5DRV9DT05ORUNUSU9OX0xJTUlUECkSGwoXRkVBVFVSRV9SSVNLX0FTU0VTU01FTlQQKhIdChlGRUFUVVJFX0FQUFJPVkFMX1dPUktGTE9XECsSFQoRRkVBVFVSRV9BVURJVF9MT0cQLBIaChZGRUFUVVJFX0VOVEVSUFJJU0VfU1NPEC0SEgoORkVBVFVSRV9UV09fRkEQLhIhCh1GRUFUVVJFX1BBU1NXT1JEX1JFU1RSSUNUSU9OUxAvEiQKIEZFQVRVUkVfRElTQUxMT1dfUEFTU1dPUkRfU0lHTklOEDASGAoURkVBVFVSRV9DVVNUT01fUk9MRVMQMRIhCh1GRUFUVVJFX1JFUVVFU1RfUk9MRV9XT1JLRkxPVxAyEhgKFEZFQVRVUkVfREFUQV9NQVNLSU5HEDMSHwobRkVBVFVSRV9EQVRBX0NMQVNTSUZJQ0FUSU9OEDQSEAoMRkVBVFVSRV9TQ0lNEDUSGgoWRkVBVFVSRV9ESVJFQ1RPUllfU1lOQxA2EiUKIUZFQVRVUkVfU0lHTl9JTl9GUkVRVUVOQ1lfQ09OVFJPTBA3EiMKH0ZFQVRVUkVfRVhURVJOQUxfU0VDUkVUX01BTkFHRVIQOBIpCiVGRUFUVVJFX1VTRVJfRU1BSUxfRE9NQUlOX1JFU1RSSUNUSU9OEDkSIgoeRkVBVFVSRV9FTlZJUk9OTUVOVF9NQU5BR0VNRU5UEDoSHAoYRkVBVFVSRV9JTV9OT1RJRklDQVRJT05TEDsSHgoaRkVBVFVSRV9URVJSQUZPUk1fUFJPVklERVIQPBIbChdGRUFUVVJFX0RBVEFCQVNFX0dST1VQUxA9Eh0KGUZFQVRVUkVfRU5WSVJPTk1FTlRfVElFUlMQPhIiCh5GRUFUVVJFX0RBU0hCT0FSRF9BTk5PVU5DRU1FTlQQPxIkCiBGRUFUVVJFX0FQSV9JTlRFR1JBVElPTl9HVUlEQU5DRRBAEhcKE0ZFQVRVUkVfQ1VTVE9NX0xPR08QQRIVChFGRUFUVVJFX1dBVEVSTUFSSxBCEiIKHkZFQVRVUkVfUk9BRE1BUF9QUklPUklUSVpBVElPThBDEhYKEkZFQVRVUkVfQ1VTVE9NX01TQRBEEh0KGUZFQVRVUkVfQ09NTVVOSVRZX1NVUFBPUlQQRRIZChVGRUFUVVJFX0VNQUlMX1NVUFBPUlQQRhImCiJGRUFUVVJFX0RFRElDQVRFRF9TVVBQT1JUX1dJVEhfU0xBEEcypQIKE1N1YnNjcmlwdGlvblNlcnZpY2UScgoPR2V0U3Vic2NyaXB0aW9uEiMuYnl0ZWJhc2UudjEuR2V0U3Vic2NyaXB0aW9uUmVxdWVzdBoZLmJ5dGViYXNlLnYxLlN1YnNjcmlwdGlvbiIf2kEAgOowAYLT5JMCEhIQL3YxL3N1YnNjcmlwdGlvbhKZAQoSVXBkYXRlU3Vic2NyaXB0aW9uEiYuYnl0ZWJhc2UudjEuVXBkYXRlU3Vic2NyaXB0aW9uUmVxdWVzdBoZLmJ5dGViYXNlLnYxLlN1YnNjcmlwdGlvbiJA2kEFcGF0Y2iK6jAPYmIuc2V0dGluZ3Muc2V0kOowAYLT5JMCGzoHbGljZW5zZTIQL3YxL3N1YnNjcmlwdGlvbkI0WjJnaXRodWIuY29tL2J5dGViYXNlL2J5dGViYXNlL3Byb3RvL2dlbmVyYXRlZC1nby92MWIGcHJvdG8z", [file_google_api_annotations, file_google_api_client, file_google_api_field_behavior, file_google_protobuf_timestamp, file_v1_annotation]);

/**
 * Describes the message bytebase.v1.GetSubscriptionRequest.
 * Use `create(GetSubscriptionRequestSchema)` to create a new message.
 */
export const GetSubscriptionRequestSchema = /*@__PURE__*/
  messageDesc(file_v1_subscription_service, 0);

/**
 * Describes the message bytebase.v1.UpdateSubscriptionRequest.
 * Use `create(UpdateSubscriptionRequestSchema)` to create a new message.
 */
export const UpdateSubscriptionRequestSchema = /*@__PURE__*/
  messageDesc(file_v1_subscription_service, 1);

/**
 * Describes the message bytebase.v1.Subscription.
 * Use `create(SubscriptionSchema)` to create a new message.
 */
export const SubscriptionSchema = /*@__PURE__*/
  messageDesc(file_v1_subscription_service, 2);

/**
 * Describes the message bytebase.v1.PlanConfig.
 * Use `create(PlanConfigSchema)` to create a new message.
 */
export const PlanConfigSchema = /*@__PURE__*/
  messageDesc(file_v1_subscription_service, 3);

/**
 * Describes the message bytebase.v1.PlanLimitConfig.
 * Use `create(PlanLimitConfigSchema)` to create a new message.
 */
export const PlanLimitConfigSchema = /*@__PURE__*/
  messageDesc(file_v1_subscription_service, 4);

/**
 * Describes the enum bytebase.v1.PlanType.
 */
export const PlanTypeSchema = /*@__PURE__*/
  enumDesc(file_v1_subscription_service, 0);

/**
 * @generated from enum bytebase.v1.PlanType
 */
export const PlanType = /*@__PURE__*/
  tsEnum(PlanTypeSchema);

/**
 * Describes the enum bytebase.v1.PlanFeature.
 */
export const PlanFeatureSchema = /*@__PURE__*/
  enumDesc(file_v1_subscription_service, 1);

/**
 * PlanFeature represents the available features in Bytebase
 *
 * @generated from enum bytebase.v1.PlanFeature
 */
export const PlanFeature = /*@__PURE__*/
  tsEnum(PlanFeatureSchema);

/**
 * @generated from service bytebase.v1.SubscriptionService
 */
export const SubscriptionService = /*@__PURE__*/
  serviceDesc(file_v1_subscription_service, 0);

