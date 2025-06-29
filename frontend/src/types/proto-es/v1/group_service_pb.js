// @generated by protoc-gen-es v2.5.2
// @generated from file v1/group_service.proto (package bytebase.v1, syntax proto3)
/* eslint-disable */

import { enumDesc, fileDesc, messageDesc, serviceDesc, tsEnum } from "@bufbuild/protobuf/codegenv2";
import { file_google_api_annotations } from "../google/api/annotations_pb";
import { file_google_api_client } from "../google/api/client_pb";
import { file_google_api_field_behavior } from "../google/api/field_behavior_pb";
import { file_google_api_resource } from "../google/api/resource_pb";
import { file_google_protobuf_empty, file_google_protobuf_field_mask } from "@bufbuild/protobuf/wkt";
import { file_v1_annotation } from "./annotation_pb";

/**
 * Describes the file v1/group_service.proto.
 */
export const file_v1_group_service = /*@__PURE__*/
  fileDesc("ChZ2MS9ncm91cF9zZXJ2aWNlLnByb3RvEgtieXRlYmFzZS52MSI8Cg9HZXRHcm91cFJlcXVlc3QSKQoEbmFtZRgBIAEoCUIb4kEBAvpBFAoSYnl0ZWJhc2UuY29tL0dyb3VwIjoKEUxpc3RHcm91cHNSZXF1ZXN0EhEKCXBhZ2Vfc2l6ZRgBIAEoBRISCgpwYWdlX3Rva2VuGAIgASgJIlEKEkxpc3RHcm91cHNSZXNwb25zZRIiCgZncm91cHMYASADKAsyEi5ieXRlYmFzZS52MS5Hcm91cBIXCg9uZXh0X3BhZ2VfdG9rZW4YAiABKAkiWAoSQ3JlYXRlR3JvdXBSZXF1ZXN0EicKBWdyb3VwGAEgASgLMhIuYnl0ZWJhc2UudjEuR3JvdXBCBOJBAQISGQoLZ3JvdXBfZW1haWwYAiABKAlCBOJBAQIihQEKElVwZGF0ZUdyb3VwUmVxdWVzdBInCgVncm91cBgBIAEoCzISLmJ5dGViYXNlLnYxLkdyb3VwQgTiQQECEi8KC3VwZGF0ZV9tYXNrGAIgASgLMhouZ29vZ2xlLnByb3RvYnVmLkZpZWxkTWFzaxIVCg1hbGxvd19taXNzaW5nGAMgASgIIj8KEkRlbGV0ZUdyb3VwUmVxdWVzdBIpCgRuYW1lGAEgASgJQhviQQEC+kEUChJieXRlYmFzZS5jb20vR3JvdXAifwoLR3JvdXBNZW1iZXISDgoGbWVtYmVyGAEgASgJEisKBHJvbGUYAiABKA4yHS5ieXRlYmFzZS52MS5Hcm91cE1lbWJlci5Sb2xlIjMKBFJvbGUSFAoQUk9MRV9VTlNQRUNJRklFRBAAEgkKBU9XTkVSEAESCgoGTUVNQkVSEAIiowEKBUdyb3VwEhIKBG5hbWUYASABKAlCBOJBAQMSDQoFdGl0bGUYAiABKAkSEwoLZGVzY3JpcHRpb24YAyABKAkSKQoHbWVtYmVycxgFIAMoCzIYLmJ5dGViYXNlLnYxLkdyb3VwTWVtYmVyEg4KBnNvdXJjZRgHIAEoCTon6kEkChJieXRlYmFzZS5jb20vR3JvdXASDmdyb3Vwcy97Z3JvdXB9Mq0FCgxHcm91cFNlcnZpY2USdQoIR2V0R3JvdXASHC5ieXRlYmFzZS52MS5HZXRHcm91cFJlcXVlc3QaEi5ieXRlYmFzZS52MS5Hcm91cCI32kEEbmFtZYrqMA1iYi5ncm91cHMuZ2V0kOowAYLT5JMCFRITL3YxL3tuYW1lPWdyb3Vwcy8qfRJ6CgpMaXN0R3JvdXBzEh4uYnl0ZWJhc2UudjEuTGlzdEdyb3Vwc1JlcXVlc3QaHy5ieXRlYmFzZS52MS5MaXN0R3JvdXBzUmVzcG9uc2UiK9pBAIrqMA5iYi5ncm91cHMubGlzdJDqMAGC0+STAgwSCi92MS9ncm91cHMSgQEKC0NyZWF0ZUdyb3VwEh8uYnl0ZWJhc2UudjEuQ3JlYXRlR3JvdXBSZXF1ZXN0GhIuYnl0ZWJhc2UudjEuR3JvdXAiPdpBBWdyb3VwiuowEGJiLmdyb3Vwcy5jcmVhdGWQ6jABmOowAYLT5JMCEzoFZ3JvdXAiCi92MS9ncm91cHMSnAEKC1VwZGF0ZUdyb3VwEh8uYnl0ZWJhc2UudjEuVXBkYXRlR3JvdXBSZXF1ZXN0GhIuYnl0ZWJhc2UudjEuR3JvdXAiWNpBEWdyb3VwLHVwZGF0ZV9tYXNriuowEGJiLmdyb3Vwcy51cGRhdGWQ6jACmOowAYLT5JMCIjoFZ3JvdXAyGS92MS97Z3JvdXAubmFtZT1ncm91cHMvKn0ShgEKC0RlbGV0ZUdyb3VwEh8uYnl0ZWJhc2UudjEuRGVsZXRlR3JvdXBSZXF1ZXN0GhYuZ29vZ2xlLnByb3RvYnVmLkVtcHR5Ij7aQQRuYW1liuowEGJiLmdyb3Vwcy5kZWxldGWQ6jACmOowAYLT5JMCFSoTL3YxL3tuYW1lPWdyb3Vwcy8qfUI0WjJnaXRodWIuY29tL2J5dGViYXNlL2J5dGViYXNlL3Byb3RvL2dlbmVyYXRlZC1nby92MWIGcHJvdG8z", [file_google_api_annotations, file_google_api_client, file_google_api_field_behavior, file_google_api_resource, file_google_protobuf_empty, file_google_protobuf_field_mask, file_v1_annotation]);

/**
 * Describes the message bytebase.v1.GetGroupRequest.
 * Use `create(GetGroupRequestSchema)` to create a new message.
 */
export const GetGroupRequestSchema = /*@__PURE__*/
  messageDesc(file_v1_group_service, 0);

/**
 * Describes the message bytebase.v1.ListGroupsRequest.
 * Use `create(ListGroupsRequestSchema)` to create a new message.
 */
export const ListGroupsRequestSchema = /*@__PURE__*/
  messageDesc(file_v1_group_service, 1);

/**
 * Describes the message bytebase.v1.ListGroupsResponse.
 * Use `create(ListGroupsResponseSchema)` to create a new message.
 */
export const ListGroupsResponseSchema = /*@__PURE__*/
  messageDesc(file_v1_group_service, 2);

/**
 * Describes the message bytebase.v1.CreateGroupRequest.
 * Use `create(CreateGroupRequestSchema)` to create a new message.
 */
export const CreateGroupRequestSchema = /*@__PURE__*/
  messageDesc(file_v1_group_service, 3);

/**
 * Describes the message bytebase.v1.UpdateGroupRequest.
 * Use `create(UpdateGroupRequestSchema)` to create a new message.
 */
export const UpdateGroupRequestSchema = /*@__PURE__*/
  messageDesc(file_v1_group_service, 4);

/**
 * Describes the message bytebase.v1.DeleteGroupRequest.
 * Use `create(DeleteGroupRequestSchema)` to create a new message.
 */
export const DeleteGroupRequestSchema = /*@__PURE__*/
  messageDesc(file_v1_group_service, 5);

/**
 * Describes the message bytebase.v1.GroupMember.
 * Use `create(GroupMemberSchema)` to create a new message.
 */
export const GroupMemberSchema = /*@__PURE__*/
  messageDesc(file_v1_group_service, 6);

/**
 * Describes the enum bytebase.v1.GroupMember.Role.
 */
export const GroupMember_RoleSchema = /*@__PURE__*/
  enumDesc(file_v1_group_service, 6, 0);

/**
 * @generated from enum bytebase.v1.GroupMember.Role
 */
export const GroupMember_Role = /*@__PURE__*/
  tsEnum(GroupMember_RoleSchema);

/**
 * Describes the message bytebase.v1.Group.
 * Use `create(GroupSchema)` to create a new message.
 */
export const GroupSchema = /*@__PURE__*/
  messageDesc(file_v1_group_service, 7);

/**
 * @generated from service bytebase.v1.GroupService
 */
export const GroupService = /*@__PURE__*/
  serviceDesc(file_v1_group_service, 0);

