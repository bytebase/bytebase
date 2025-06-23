import { fromJson, toJson } from "@bufbuild/protobuf";
import type { User as OldUser, UserType as OldUserType } from "@/types/proto/v1/user_service";
import { User as OldUserProto, UserType } from "@/types/proto/v1/user_service";
import type { User as NewUser, UserType as NewUserType } from "@/types/proto-es/v1/user_service_pb";
import { UserSchema, UserType as NewUserTypeEnum } from "@/types/proto-es/v1/user_service_pb";

// Convert old proto User to proto-es User
export const convertOldUserToNew = (oldUser: OldUser): NewUser => {
  const json = OldUserProto.toJSON(oldUser) as any; // Type assertion needed due to proto type incompatibility
  return fromJson(UserSchema, json);
};

// Convert proto-es User to old proto User
export const convertNewUserToOld = (newUser: NewUser): OldUser => {
  const json = toJson(UserSchema, newUser);
  return OldUserProto.fromJSON(json);
};

// Convert old UserType enum to new UserType enum
export const convertOldUserTypeToNew = (oldType: OldUserType): NewUserType => {
  const mapping: Record<OldUserType, NewUserType> = {
    [UserType.USER_TYPE_UNSPECIFIED]: NewUserTypeEnum.USER_TYPE_UNSPECIFIED,
    [UserType.USER]: NewUserTypeEnum.USER,
    [UserType.SYSTEM_BOT]: NewUserTypeEnum.SYSTEM_BOT,
    [UserType.SERVICE_ACCOUNT]: NewUserTypeEnum.SERVICE_ACCOUNT,
    [UserType.UNRECOGNIZED]: NewUserTypeEnum.USER_TYPE_UNSPECIFIED,
  };
  return mapping[oldType] ?? NewUserTypeEnum.USER_TYPE_UNSPECIFIED;
};

// Convert new UserType enum to old UserType enum
export const convertNewUserTypeToOld = (newType: NewUserType): OldUserType => {
  const mapping: Record<NewUserType, OldUserType> = {
    [NewUserTypeEnum.USER_TYPE_UNSPECIFIED]: UserType.USER_TYPE_UNSPECIFIED,
    [NewUserTypeEnum.USER]: UserType.USER,
    [NewUserTypeEnum.SYSTEM_BOT]: UserType.SYSTEM_BOT,
    [NewUserTypeEnum.SERVICE_ACCOUNT]: UserType.SERVICE_ACCOUNT,
  };
  return mapping[newType] ?? UserType.UNRECOGNIZED;
};

// Convert old UserType enum to string format for requests
export const convertOldUserTypeToNewString = (oldType: OldUserType): string => {
  const newEnumValue = convertOldUserTypeToNew(oldType);
  return NewUserTypeEnum[newEnumValue]; // Convert numeric enum to string
};