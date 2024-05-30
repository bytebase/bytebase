/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "google.protobuf";

/** The full set of known editions. */
export enum Edition {
  /** EDITION_UNKNOWN - A placeholder for an unknown edition value. */
  EDITION_UNKNOWN = "EDITION_UNKNOWN",
  /**
   * EDITION_LEGACY - A placeholder edition for specifying default behaviors *before* a feature
   * was first introduced.  This is effectively an "infinite past".
   */
  EDITION_LEGACY = "EDITION_LEGACY",
  /**
   * EDITION_PROTO2 - Legacy syntax "editions".  These pre-date editions, but behave much like
   * distinct editions.  These can't be used to specify the edition of proto
   * files, but feature definitions must supply proto2/proto3 defaults for
   * backwards compatibility.
   */
  EDITION_PROTO2 = "EDITION_PROTO2",
  EDITION_PROTO3 = "EDITION_PROTO3",
  /**
   * EDITION_2023 - Editions that have been released.  The specific values are arbitrary and
   * should not be depended on, but they will always be time-ordered for easy
   * comparison.
   */
  EDITION_2023 = "EDITION_2023",
  EDITION_2024 = "EDITION_2024",
  /**
   * EDITION_1_TEST_ONLY - Placeholder editions for testing feature resolution.  These should not be
   * used or relyed on outside of tests.
   */
  EDITION_1_TEST_ONLY = "EDITION_1_TEST_ONLY",
  EDITION_2_TEST_ONLY = "EDITION_2_TEST_ONLY",
  EDITION_99997_TEST_ONLY = "EDITION_99997_TEST_ONLY",
  EDITION_99998_TEST_ONLY = "EDITION_99998_TEST_ONLY",
  EDITION_99999_TEST_ONLY = "EDITION_99999_TEST_ONLY",
  /**
   * EDITION_MAX - Placeholder for specifying unbounded edition support.  This should only
   * ever be used by plugins that can expect to never require any changes to
   * support a new edition.
   */
  EDITION_MAX = "EDITION_MAX",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function editionFromJSON(object: any): Edition {
  switch (object) {
    case 0:
    case "EDITION_UNKNOWN":
      return Edition.EDITION_UNKNOWN;
    case 900:
    case "EDITION_LEGACY":
      return Edition.EDITION_LEGACY;
    case 998:
    case "EDITION_PROTO2":
      return Edition.EDITION_PROTO2;
    case 999:
    case "EDITION_PROTO3":
      return Edition.EDITION_PROTO3;
    case 1000:
    case "EDITION_2023":
      return Edition.EDITION_2023;
    case 1001:
    case "EDITION_2024":
      return Edition.EDITION_2024;
    case 1:
    case "EDITION_1_TEST_ONLY":
      return Edition.EDITION_1_TEST_ONLY;
    case 2:
    case "EDITION_2_TEST_ONLY":
      return Edition.EDITION_2_TEST_ONLY;
    case 99997:
    case "EDITION_99997_TEST_ONLY":
      return Edition.EDITION_99997_TEST_ONLY;
    case 99998:
    case "EDITION_99998_TEST_ONLY":
      return Edition.EDITION_99998_TEST_ONLY;
    case 99999:
    case "EDITION_99999_TEST_ONLY":
      return Edition.EDITION_99999_TEST_ONLY;
    case 2147483647:
    case "EDITION_MAX":
      return Edition.EDITION_MAX;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Edition.UNRECOGNIZED;
  }
}

export function editionToJSON(object: Edition): string {
  switch (object) {
    case Edition.EDITION_UNKNOWN:
      return "EDITION_UNKNOWN";
    case Edition.EDITION_LEGACY:
      return "EDITION_LEGACY";
    case Edition.EDITION_PROTO2:
      return "EDITION_PROTO2";
    case Edition.EDITION_PROTO3:
      return "EDITION_PROTO3";
    case Edition.EDITION_2023:
      return "EDITION_2023";
    case Edition.EDITION_2024:
      return "EDITION_2024";
    case Edition.EDITION_1_TEST_ONLY:
      return "EDITION_1_TEST_ONLY";
    case Edition.EDITION_2_TEST_ONLY:
      return "EDITION_2_TEST_ONLY";
    case Edition.EDITION_99997_TEST_ONLY:
      return "EDITION_99997_TEST_ONLY";
    case Edition.EDITION_99998_TEST_ONLY:
      return "EDITION_99998_TEST_ONLY";
    case Edition.EDITION_99999_TEST_ONLY:
      return "EDITION_99999_TEST_ONLY";
    case Edition.EDITION_MAX:
      return "EDITION_MAX";
    case Edition.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function editionToNumber(object: Edition): number {
  switch (object) {
    case Edition.EDITION_UNKNOWN:
      return 0;
    case Edition.EDITION_LEGACY:
      return 900;
    case Edition.EDITION_PROTO2:
      return 998;
    case Edition.EDITION_PROTO3:
      return 999;
    case Edition.EDITION_2023:
      return 1000;
    case Edition.EDITION_2024:
      return 1001;
    case Edition.EDITION_1_TEST_ONLY:
      return 1;
    case Edition.EDITION_2_TEST_ONLY:
      return 2;
    case Edition.EDITION_99997_TEST_ONLY:
      return 99997;
    case Edition.EDITION_99998_TEST_ONLY:
      return 99998;
    case Edition.EDITION_99999_TEST_ONLY:
      return 99999;
    case Edition.EDITION_MAX:
      return 2147483647;
    case Edition.UNRECOGNIZED:
    default:
      return -1;
  }
}

/**
 * The protocol compiler can output a FileDescriptorSet containing the .proto
 * files it parses.
 */
export interface FileDescriptorSet {
  file: FileDescriptorProto[];
}

/** Describes a complete .proto file. */
export interface FileDescriptorProto {
  /** file name, relative to root of source tree */
  name: string;
  /** e.g. "foo", "foo.bar", etc. */
  package: string;
  /** Names of files imported by this file. */
  dependency: string[];
  /** Indexes of the public imported files in the dependency list above. */
  publicDependency: number[];
  /**
   * Indexes of the weak imported files in the dependency list.
   * For Google-internal migration only. Do not use.
   */
  weakDependency: number[];
  /** All top-level definitions in this file. */
  messageType: DescriptorProto[];
  enumType: EnumDescriptorProto[];
  service: ServiceDescriptorProto[];
  extension: FieldDescriptorProto[];
  options:
    | FileOptions
    | undefined;
  /**
   * This field contains optional information about the original source code.
   * You may safely remove this entire field without harming runtime
   * functionality of the descriptors -- the information is needed only by
   * development tools.
   */
  sourceCodeInfo:
    | SourceCodeInfo
    | undefined;
  /**
   * The syntax of the proto file.
   * The supported values are "proto2", "proto3", and "editions".
   *
   * If `edition` is present, this value must be "editions".
   */
  syntax: string;
  /** The edition of the proto file. */
  edition: Edition;
}

/** Describes a message type. */
export interface DescriptorProto {
  name: string;
  field: FieldDescriptorProto[];
  extension: FieldDescriptorProto[];
  nestedType: DescriptorProto[];
  enumType: EnumDescriptorProto[];
  extensionRange: DescriptorProto_ExtensionRange[];
  oneofDecl: OneofDescriptorProto[];
  options: MessageOptions | undefined;
  reservedRange: DescriptorProto_ReservedRange[];
  /**
   * Reserved field names, which may not be used by fields in the same message.
   * A given name may only be reserved once.
   */
  reservedName: string[];
}

export interface DescriptorProto_ExtensionRange {
  /** Inclusive. */
  start: number;
  /** Exclusive. */
  end: number;
  options: ExtensionRangeOptions | undefined;
}

/**
 * Range of reserved tag numbers. Reserved tag numbers may not be used by
 * fields or extension ranges in the same message. Reserved ranges may
 * not overlap.
 */
export interface DescriptorProto_ReservedRange {
  /** Inclusive. */
  start: number;
  /** Exclusive. */
  end: number;
}

export interface ExtensionRangeOptions {
  /** The parser stores options it doesn't recognize here. See above. */
  uninterpretedOption: UninterpretedOption[];
  /**
   * For external users: DO NOT USE. We are in the process of open sourcing
   * extension declaration and executing internal cleanups before it can be
   * used externally.
   */
  declaration: ExtensionRangeOptions_Declaration[];
  /** Any features defined in the specific edition. */
  features:
    | FeatureSet
    | undefined;
  /**
   * The verification state of the range.
   * TODO: flip the default to DECLARATION once all empty ranges
   * are marked as UNVERIFIED.
   */
  verification: ExtensionRangeOptions_VerificationState;
}

/** The verification state of the extension range. */
export enum ExtensionRangeOptions_VerificationState {
  /** DECLARATION - All the extensions of the range must be declared. */
  DECLARATION = "DECLARATION",
  UNVERIFIED = "UNVERIFIED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function extensionRangeOptions_VerificationStateFromJSON(object: any): ExtensionRangeOptions_VerificationState {
  switch (object) {
    case 0:
    case "DECLARATION":
      return ExtensionRangeOptions_VerificationState.DECLARATION;
    case 1:
    case "UNVERIFIED":
      return ExtensionRangeOptions_VerificationState.UNVERIFIED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ExtensionRangeOptions_VerificationState.UNRECOGNIZED;
  }
}

export function extensionRangeOptions_VerificationStateToJSON(object: ExtensionRangeOptions_VerificationState): string {
  switch (object) {
    case ExtensionRangeOptions_VerificationState.DECLARATION:
      return "DECLARATION";
    case ExtensionRangeOptions_VerificationState.UNVERIFIED:
      return "UNVERIFIED";
    case ExtensionRangeOptions_VerificationState.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function extensionRangeOptions_VerificationStateToNumber(
  object: ExtensionRangeOptions_VerificationState,
): number {
  switch (object) {
    case ExtensionRangeOptions_VerificationState.DECLARATION:
      return 0;
    case ExtensionRangeOptions_VerificationState.UNVERIFIED:
      return 1;
    case ExtensionRangeOptions_VerificationState.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface ExtensionRangeOptions_Declaration {
  /** The extension number declared within the extension range. */
  number: number;
  /**
   * The fully-qualified name of the extension field. There must be a leading
   * dot in front of the full name.
   */
  fullName: string;
  /**
   * The fully-qualified type name of the extension field. Unlike
   * Metadata.type, Declaration.type must have a leading dot for messages
   * and enums.
   */
  type: string;
  /**
   * If true, indicates that the number is reserved in the extension range,
   * and any extension field with the number will fail to compile. Set this
   * when a declared extension field is deleted.
   */
  reserved: boolean;
  /**
   * If true, indicates that the extension must be defined as repeated.
   * Otherwise the extension must be defined as optional.
   */
  repeated: boolean;
}

/** Describes a field within a message. */
export interface FieldDescriptorProto {
  name: string;
  number: number;
  label: FieldDescriptorProto_Label;
  /**
   * If type_name is set, this need not be set.  If both this and type_name
   * are set, this must be one of TYPE_ENUM, TYPE_MESSAGE or TYPE_GROUP.
   */
  type: FieldDescriptorProto_Type;
  /**
   * For message and enum types, this is the name of the type.  If the name
   * starts with a '.', it is fully-qualified.  Otherwise, C++-like scoping
   * rules are used to find the type (i.e. first the nested types within this
   * message are searched, then within the parent, on up to the root
   * namespace).
   */
  typeName: string;
  /**
   * For extensions, this is the name of the type being extended.  It is
   * resolved in the same manner as type_name.
   */
  extendee: string;
  /**
   * For numeric types, contains the original text representation of the value.
   * For booleans, "true" or "false".
   * For strings, contains the default text contents (not escaped in any way).
   * For bytes, contains the C escaped value.  All bytes >= 128 are escaped.
   */
  defaultValue: string;
  /**
   * If set, gives the index of a oneof in the containing type's oneof_decl
   * list.  This field is a member of that oneof.
   */
  oneofIndex: number;
  /**
   * JSON name of this field. The value is set by protocol compiler. If the
   * user has set a "json_name" option on this field, that option's value
   * will be used. Otherwise, it's deduced from the field's name by converting
   * it to camelCase.
   */
  jsonName: string;
  options:
    | FieldOptions
    | undefined;
  /**
   * If true, this is a proto3 "optional". When a proto3 field is optional, it
   * tracks presence regardless of field type.
   *
   * When proto3_optional is true, this field must belong to a oneof to signal
   * to old proto3 clients that presence is tracked for this field. This oneof
   * is known as a "synthetic" oneof, and this field must be its sole member
   * (each proto3 optional field gets its own synthetic oneof). Synthetic oneofs
   * exist in the descriptor only, and do not generate any API. Synthetic oneofs
   * must be ordered after all "real" oneofs.
   *
   * For message fields, proto3_optional doesn't create any semantic change,
   * since non-repeated message fields always track presence. However it still
   * indicates the semantic detail of whether the user wrote "optional" or not.
   * This can be useful for round-tripping the .proto file. For consistency we
   * give message fields a synthetic oneof also, even though it is not required
   * to track presence. This is especially important because the parser can't
   * tell if a field is a message or an enum, so it must always create a
   * synthetic oneof.
   *
   * Proto2 optional fields do not set this flag, because they already indicate
   * optional with `LABEL_OPTIONAL`.
   */
  proto3Optional: boolean;
}

export enum FieldDescriptorProto_Type {
  /**
   * TYPE_DOUBLE - 0 is reserved for errors.
   * Order is weird for historical reasons.
   */
  TYPE_DOUBLE = "TYPE_DOUBLE",
  TYPE_FLOAT = "TYPE_FLOAT",
  /**
   * TYPE_INT64 - Not ZigZag encoded.  Negative numbers take 10 bytes.  Use TYPE_SINT64 if
   * negative values are likely.
   */
  TYPE_INT64 = "TYPE_INT64",
  TYPE_UINT64 = "TYPE_UINT64",
  /**
   * TYPE_INT32 - Not ZigZag encoded.  Negative numbers take 10 bytes.  Use TYPE_SINT32 if
   * negative values are likely.
   */
  TYPE_INT32 = "TYPE_INT32",
  TYPE_FIXED64 = "TYPE_FIXED64",
  TYPE_FIXED32 = "TYPE_FIXED32",
  TYPE_BOOL = "TYPE_BOOL",
  TYPE_STRING = "TYPE_STRING",
  /**
   * TYPE_GROUP - Tag-delimited aggregate.
   * Group type is deprecated and not supported after google.protobuf. However, Proto3
   * implementations should still be able to parse the group wire format and
   * treat group fields as unknown fields.  In Editions, the group wire format
   * can be enabled via the `message_encoding` feature.
   */
  TYPE_GROUP = "TYPE_GROUP",
  /** TYPE_MESSAGE - Length-delimited aggregate. */
  TYPE_MESSAGE = "TYPE_MESSAGE",
  /** TYPE_BYTES - New in version 2. */
  TYPE_BYTES = "TYPE_BYTES",
  TYPE_UINT32 = "TYPE_UINT32",
  TYPE_ENUM = "TYPE_ENUM",
  TYPE_SFIXED32 = "TYPE_SFIXED32",
  TYPE_SFIXED64 = "TYPE_SFIXED64",
  /** TYPE_SINT32 - Uses ZigZag encoding. */
  TYPE_SINT32 = "TYPE_SINT32",
  /** TYPE_SINT64 - Uses ZigZag encoding. */
  TYPE_SINT64 = "TYPE_SINT64",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function fieldDescriptorProto_TypeFromJSON(object: any): FieldDescriptorProto_Type {
  switch (object) {
    case 1:
    case "TYPE_DOUBLE":
      return FieldDescriptorProto_Type.TYPE_DOUBLE;
    case 2:
    case "TYPE_FLOAT":
      return FieldDescriptorProto_Type.TYPE_FLOAT;
    case 3:
    case "TYPE_INT64":
      return FieldDescriptorProto_Type.TYPE_INT64;
    case 4:
    case "TYPE_UINT64":
      return FieldDescriptorProto_Type.TYPE_UINT64;
    case 5:
    case "TYPE_INT32":
      return FieldDescriptorProto_Type.TYPE_INT32;
    case 6:
    case "TYPE_FIXED64":
      return FieldDescriptorProto_Type.TYPE_FIXED64;
    case 7:
    case "TYPE_FIXED32":
      return FieldDescriptorProto_Type.TYPE_FIXED32;
    case 8:
    case "TYPE_BOOL":
      return FieldDescriptorProto_Type.TYPE_BOOL;
    case 9:
    case "TYPE_STRING":
      return FieldDescriptorProto_Type.TYPE_STRING;
    case 10:
    case "TYPE_GROUP":
      return FieldDescriptorProto_Type.TYPE_GROUP;
    case 11:
    case "TYPE_MESSAGE":
      return FieldDescriptorProto_Type.TYPE_MESSAGE;
    case 12:
    case "TYPE_BYTES":
      return FieldDescriptorProto_Type.TYPE_BYTES;
    case 13:
    case "TYPE_UINT32":
      return FieldDescriptorProto_Type.TYPE_UINT32;
    case 14:
    case "TYPE_ENUM":
      return FieldDescriptorProto_Type.TYPE_ENUM;
    case 15:
    case "TYPE_SFIXED32":
      return FieldDescriptorProto_Type.TYPE_SFIXED32;
    case 16:
    case "TYPE_SFIXED64":
      return FieldDescriptorProto_Type.TYPE_SFIXED64;
    case 17:
    case "TYPE_SINT32":
      return FieldDescriptorProto_Type.TYPE_SINT32;
    case 18:
    case "TYPE_SINT64":
      return FieldDescriptorProto_Type.TYPE_SINT64;
    case -1:
    case "UNRECOGNIZED":
    default:
      return FieldDescriptorProto_Type.UNRECOGNIZED;
  }
}

export function fieldDescriptorProto_TypeToJSON(object: FieldDescriptorProto_Type): string {
  switch (object) {
    case FieldDescriptorProto_Type.TYPE_DOUBLE:
      return "TYPE_DOUBLE";
    case FieldDescriptorProto_Type.TYPE_FLOAT:
      return "TYPE_FLOAT";
    case FieldDescriptorProto_Type.TYPE_INT64:
      return "TYPE_INT64";
    case FieldDescriptorProto_Type.TYPE_UINT64:
      return "TYPE_UINT64";
    case FieldDescriptorProto_Type.TYPE_INT32:
      return "TYPE_INT32";
    case FieldDescriptorProto_Type.TYPE_FIXED64:
      return "TYPE_FIXED64";
    case FieldDescriptorProto_Type.TYPE_FIXED32:
      return "TYPE_FIXED32";
    case FieldDescriptorProto_Type.TYPE_BOOL:
      return "TYPE_BOOL";
    case FieldDescriptorProto_Type.TYPE_STRING:
      return "TYPE_STRING";
    case FieldDescriptorProto_Type.TYPE_GROUP:
      return "TYPE_GROUP";
    case FieldDescriptorProto_Type.TYPE_MESSAGE:
      return "TYPE_MESSAGE";
    case FieldDescriptorProto_Type.TYPE_BYTES:
      return "TYPE_BYTES";
    case FieldDescriptorProto_Type.TYPE_UINT32:
      return "TYPE_UINT32";
    case FieldDescriptorProto_Type.TYPE_ENUM:
      return "TYPE_ENUM";
    case FieldDescriptorProto_Type.TYPE_SFIXED32:
      return "TYPE_SFIXED32";
    case FieldDescriptorProto_Type.TYPE_SFIXED64:
      return "TYPE_SFIXED64";
    case FieldDescriptorProto_Type.TYPE_SINT32:
      return "TYPE_SINT32";
    case FieldDescriptorProto_Type.TYPE_SINT64:
      return "TYPE_SINT64";
    case FieldDescriptorProto_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function fieldDescriptorProto_TypeToNumber(object: FieldDescriptorProto_Type): number {
  switch (object) {
    case FieldDescriptorProto_Type.TYPE_DOUBLE:
      return 1;
    case FieldDescriptorProto_Type.TYPE_FLOAT:
      return 2;
    case FieldDescriptorProto_Type.TYPE_INT64:
      return 3;
    case FieldDescriptorProto_Type.TYPE_UINT64:
      return 4;
    case FieldDescriptorProto_Type.TYPE_INT32:
      return 5;
    case FieldDescriptorProto_Type.TYPE_FIXED64:
      return 6;
    case FieldDescriptorProto_Type.TYPE_FIXED32:
      return 7;
    case FieldDescriptorProto_Type.TYPE_BOOL:
      return 8;
    case FieldDescriptorProto_Type.TYPE_STRING:
      return 9;
    case FieldDescriptorProto_Type.TYPE_GROUP:
      return 10;
    case FieldDescriptorProto_Type.TYPE_MESSAGE:
      return 11;
    case FieldDescriptorProto_Type.TYPE_BYTES:
      return 12;
    case FieldDescriptorProto_Type.TYPE_UINT32:
      return 13;
    case FieldDescriptorProto_Type.TYPE_ENUM:
      return 14;
    case FieldDescriptorProto_Type.TYPE_SFIXED32:
      return 15;
    case FieldDescriptorProto_Type.TYPE_SFIXED64:
      return 16;
    case FieldDescriptorProto_Type.TYPE_SINT32:
      return 17;
    case FieldDescriptorProto_Type.TYPE_SINT64:
      return 18;
    case FieldDescriptorProto_Type.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum FieldDescriptorProto_Label {
  /** LABEL_OPTIONAL - 0 is reserved for errors */
  LABEL_OPTIONAL = "LABEL_OPTIONAL",
  LABEL_REPEATED = "LABEL_REPEATED",
  /**
   * LABEL_REQUIRED - The required label is only allowed in google.protobuf.  In proto3 and Editions
   * it's explicitly prohibited.  In Editions, the `field_presence` feature
   * can be used to get this behavior.
   */
  LABEL_REQUIRED = "LABEL_REQUIRED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function fieldDescriptorProto_LabelFromJSON(object: any): FieldDescriptorProto_Label {
  switch (object) {
    case 1:
    case "LABEL_OPTIONAL":
      return FieldDescriptorProto_Label.LABEL_OPTIONAL;
    case 3:
    case "LABEL_REPEATED":
      return FieldDescriptorProto_Label.LABEL_REPEATED;
    case 2:
    case "LABEL_REQUIRED":
      return FieldDescriptorProto_Label.LABEL_REQUIRED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return FieldDescriptorProto_Label.UNRECOGNIZED;
  }
}

export function fieldDescriptorProto_LabelToJSON(object: FieldDescriptorProto_Label): string {
  switch (object) {
    case FieldDescriptorProto_Label.LABEL_OPTIONAL:
      return "LABEL_OPTIONAL";
    case FieldDescriptorProto_Label.LABEL_REPEATED:
      return "LABEL_REPEATED";
    case FieldDescriptorProto_Label.LABEL_REQUIRED:
      return "LABEL_REQUIRED";
    case FieldDescriptorProto_Label.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function fieldDescriptorProto_LabelToNumber(object: FieldDescriptorProto_Label): number {
  switch (object) {
    case FieldDescriptorProto_Label.LABEL_OPTIONAL:
      return 1;
    case FieldDescriptorProto_Label.LABEL_REPEATED:
      return 3;
    case FieldDescriptorProto_Label.LABEL_REQUIRED:
      return 2;
    case FieldDescriptorProto_Label.UNRECOGNIZED:
    default:
      return -1;
  }
}

/** Describes a oneof. */
export interface OneofDescriptorProto {
  name: string;
  options: OneofOptions | undefined;
}

/** Describes an enum type. */
export interface EnumDescriptorProto {
  name: string;
  value: EnumValueDescriptorProto[];
  options:
    | EnumOptions
    | undefined;
  /**
   * Range of reserved numeric values. Reserved numeric values may not be used
   * by enum values in the same enum declaration. Reserved ranges may not
   * overlap.
   */
  reservedRange: EnumDescriptorProto_EnumReservedRange[];
  /**
   * Reserved enum value names, which may not be reused. A given name may only
   * be reserved once.
   */
  reservedName: string[];
}

/**
 * Range of reserved numeric values. Reserved values may not be used by
 * entries in the same enum. Reserved ranges may not overlap.
 *
 * Note that this is distinct from DescriptorProto.ReservedRange in that it
 * is inclusive such that it can appropriately represent the entire int32
 * domain.
 */
export interface EnumDescriptorProto_EnumReservedRange {
  /** Inclusive. */
  start: number;
  /** Inclusive. */
  end: number;
}

/** Describes a value within an enum. */
export interface EnumValueDescriptorProto {
  name: string;
  number: number;
  options: EnumValueOptions | undefined;
}

/** Describes a service. */
export interface ServiceDescriptorProto {
  name: string;
  method: MethodDescriptorProto[];
  options: ServiceOptions | undefined;
}

/** Describes a method of a service. */
export interface MethodDescriptorProto {
  name: string;
  /**
   * Input and output type names.  These are resolved in the same way as
   * FieldDescriptorProto.type_name, but must refer to a message type.
   */
  inputType: string;
  outputType: string;
  options:
    | MethodOptions
    | undefined;
  /** Identifies if client streams multiple client messages */
  clientStreaming: boolean;
  /** Identifies if server streams multiple server messages */
  serverStreaming: boolean;
}

export interface FileOptions {
  /**
   * Sets the Java package where classes generated from this .proto will be
   * placed.  By default, the proto package is used, but this is often
   * inappropriate because proto packages do not normally start with backwards
   * domain names.
   */
  javaPackage: string;
  /**
   * Controls the name of the wrapper Java class generated for the .proto file.
   * That class will always contain the .proto file's getDescriptor() method as
   * well as any top-level extensions defined in the .proto file.
   * If java_multiple_files is disabled, then all the other classes from the
   * .proto file will be nested inside the single wrapper outer class.
   */
  javaOuterClassname: string;
  /**
   * If enabled, then the Java code generator will generate a separate .java
   * file for each top-level message, enum, and service defined in the .proto
   * file.  Thus, these types will *not* be nested inside the wrapper class
   * named by java_outer_classname.  However, the wrapper class will still be
   * generated to contain the file's getDescriptor() method as well as any
   * top-level extensions defined in the file.
   */
  javaMultipleFiles: boolean;
  /**
   * This option does nothing.
   *
   * @deprecated
   */
  javaGenerateEqualsAndHash: boolean;
  /**
   * A proto2 file can set this to true to opt in to UTF-8 checking for Java,
   * which will throw an exception if invalid UTF-8 is parsed from the wire or
   * assigned to a string field.
   *
   * TODO: clarify exactly what kinds of field types this option
   * applies to, and update these docs accordingly.
   *
   * Proto3 files already perform these checks. Setting the option explicitly to
   * false has no effect: it cannot be used to opt proto3 files out of UTF-8
   * checks.
   */
  javaStringCheckUtf8: boolean;
  optimizeFor: FileOptions_OptimizeMode;
  /**
   * Sets the Go package where structs generated from this .proto will be
   * placed. If omitted, the Go package will be derived from the following:
   *   - The basename of the package import path, if provided.
   *   - Otherwise, the package statement in the .proto file, if present.
   *   - Otherwise, the basename of the .proto file, without extension.
   */
  goPackage: string;
  /**
   * Should generic services be generated in each language?  "Generic" services
   * are not specific to any particular RPC system.  They are generated by the
   * main code generators in each language (without additional plugins).
   * Generic services were the only kind of service generation supported by
   * early versions of google.protobuf.
   *
   * Generic services are now considered deprecated in favor of using plugins
   * that generate code specific to your particular RPC system.  Therefore,
   * these default to false.  Old code which depends on generic services should
   * explicitly set them to true.
   */
  ccGenericServices: boolean;
  javaGenericServices: boolean;
  pyGenericServices: boolean;
  /**
   * Is this file deprecated?
   * Depending on the target platform, this can emit Deprecated annotations
   * for everything in the file, or it will be completely ignored; in the very
   * least, this is a formalization for deprecating files.
   */
  deprecated: boolean;
  /**
   * Enables the use of arenas for the proto messages in this file. This applies
   * only to generated classes for C++.
   */
  ccEnableArenas: boolean;
  /**
   * Sets the objective c class prefix which is prepended to all objective c
   * generated classes from this .proto. There is no default.
   */
  objcClassPrefix: string;
  /** Namespace for generated classes; defaults to the package. */
  csharpNamespace: string;
  /**
   * By default Swift generators will take the proto package and CamelCase it
   * replacing '.' with underscore and use that to prefix the types/symbols
   * defined. When this options is provided, they will use this value instead
   * to prefix the types/symbols defined.
   */
  swiftPrefix: string;
  /**
   * Sets the php class prefix which is prepended to all php generated classes
   * from this .proto. Default is empty.
   */
  phpClassPrefix: string;
  /**
   * Use this option to change the namespace of php generated classes. Default
   * is empty. When this option is empty, the package name will be used for
   * determining the namespace.
   */
  phpNamespace: string;
  /**
   * Use this option to change the namespace of php generated metadata classes.
   * Default is empty. When this option is empty, the proto file name will be
   * used for determining the namespace.
   */
  phpMetadataNamespace: string;
  /**
   * Use this option to change the package of ruby generated classes. Default
   * is empty. When this option is not set, the package name will be used for
   * determining the ruby package.
   */
  rubyPackage: string;
  /** Any features defined in the specific edition. */
  features:
    | FeatureSet
    | undefined;
  /**
   * The parser stores options it doesn't recognize here.
   * See the documentation for the "Options" section above.
   */
  uninterpretedOption: UninterpretedOption[];
}

/** Generated classes can be optimized for speed or code size. */
export enum FileOptions_OptimizeMode {
  /** SPEED - Generate complete code for parsing, serialization, */
  SPEED = "SPEED",
  /** CODE_SIZE - etc. */
  CODE_SIZE = "CODE_SIZE",
  /** LITE_RUNTIME - Generate code using MessageLite and the lite runtime. */
  LITE_RUNTIME = "LITE_RUNTIME",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function fileOptions_OptimizeModeFromJSON(object: any): FileOptions_OptimizeMode {
  switch (object) {
    case 1:
    case "SPEED":
      return FileOptions_OptimizeMode.SPEED;
    case 2:
    case "CODE_SIZE":
      return FileOptions_OptimizeMode.CODE_SIZE;
    case 3:
    case "LITE_RUNTIME":
      return FileOptions_OptimizeMode.LITE_RUNTIME;
    case -1:
    case "UNRECOGNIZED":
    default:
      return FileOptions_OptimizeMode.UNRECOGNIZED;
  }
}

export function fileOptions_OptimizeModeToJSON(object: FileOptions_OptimizeMode): string {
  switch (object) {
    case FileOptions_OptimizeMode.SPEED:
      return "SPEED";
    case FileOptions_OptimizeMode.CODE_SIZE:
      return "CODE_SIZE";
    case FileOptions_OptimizeMode.LITE_RUNTIME:
      return "LITE_RUNTIME";
    case FileOptions_OptimizeMode.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function fileOptions_OptimizeModeToNumber(object: FileOptions_OptimizeMode): number {
  switch (object) {
    case FileOptions_OptimizeMode.SPEED:
      return 1;
    case FileOptions_OptimizeMode.CODE_SIZE:
      return 2;
    case FileOptions_OptimizeMode.LITE_RUNTIME:
      return 3;
    case FileOptions_OptimizeMode.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface MessageOptions {
  /**
   * Set true to use the old proto1 MessageSet wire format for extensions.
   * This is provided for backwards-compatibility with the MessageSet wire
   * format.  You should not use this for any other reason:  It's less
   * efficient, has fewer features, and is more complicated.
   *
   * The message must be defined exactly as follows:
   *   message Foo {
   *     option message_set_wire_format = true;
   *     extensions 4 to max;
   *   }
   * Note that the message cannot have any defined fields; MessageSets only
   * have extensions.
   *
   * All extensions of your type must be singular messages; e.g. they cannot
   * be int32s, enums, or repeated messages.
   *
   * Because this is an option, the above two restrictions are not enforced by
   * the protocol compiler.
   */
  messageSetWireFormat: boolean;
  /**
   * Disables the generation of the standard "descriptor()" accessor, which can
   * conflict with a field of the same name.  This is meant to make migration
   * from proto1 easier; new code should avoid fields named "descriptor".
   */
  noStandardDescriptorAccessor: boolean;
  /**
   * Is this message deprecated?
   * Depending on the target platform, this can emit Deprecated annotations
   * for the message, or it will be completely ignored; in the very least,
   * this is a formalization for deprecating messages.
   */
  deprecated: boolean;
  /**
   * Whether the message is an automatically generated map entry type for the
   * maps field.
   *
   * For maps fields:
   *     map<KeyType, ValueType> map_field = 1;
   * The parsed descriptor looks like:
   *     message MapFieldEntry {
   *         option map_entry = true;
   *         optional KeyType key = 1;
   *         optional ValueType value = 2;
   *     }
   *     repeated MapFieldEntry map_field = 1;
   *
   * Implementations may choose not to generate the map_entry=true message, but
   * use a native map in the target language to hold the keys and values.
   * The reflection APIs in such implementations still need to work as
   * if the field is a repeated message field.
   *
   * NOTE: Do not set the option in .proto files. Always use the maps syntax
   * instead. The option should only be implicitly set by the proto compiler
   * parser.
   */
  mapEntry: boolean;
  /**
   * Enable the legacy handling of JSON field name conflicts.  This lowercases
   * and strips underscored from the fields before comparison in proto3 only.
   * The new behavior takes `json_name` into account and applies to proto2 as
   * well.
   *
   * This should only be used as a temporary measure against broken builds due
   * to the change in behavior for JSON field name conflicts.
   *
   * TODO This is legacy behavior we plan to remove once downstream
   * teams have had time to migrate.
   *
   * @deprecated
   */
  deprecatedLegacyJsonFieldConflicts: boolean;
  /** Any features defined in the specific edition. */
  features:
    | FeatureSet
    | undefined;
  /** The parser stores options it doesn't recognize here. See above. */
  uninterpretedOption: UninterpretedOption[];
}

export interface FieldOptions {
  /**
   * The ctype option instructs the C++ code generator to use a different
   * representation of the field than it normally would.  See the specific
   * options below.  This option is only implemented to support use of
   * [ctype=CORD] and [ctype=STRING] (the default) on non-repeated fields of
   * type "bytes" in the open source release -- sorry, we'll try to include
   * other types in a future version!
   */
  ctype: FieldOptions_CType;
  /**
   * The packed option can be enabled for repeated primitive fields to enable
   * a more efficient representation on the wire. Rather than repeatedly
   * writing the tag and type for each element, the entire array is encoded as
   * a single length-delimited blob. In proto3, only explicit setting it to
   * false will avoid using packed encoding.  This option is prohibited in
   * Editions, but the `repeated_field_encoding` feature can be used to control
   * the behavior.
   */
  packed: boolean;
  /**
   * The jstype option determines the JavaScript type used for values of the
   * field.  The option is permitted only for 64 bit integral and fixed types
   * (int64, uint64, sint64, fixed64, sfixed64).  A field with jstype JS_STRING
   * is represented as JavaScript string, which avoids loss of precision that
   * can happen when a large value is converted to a floating point JavaScript.
   * Specifying JS_NUMBER for the jstype causes the generated JavaScript code to
   * use the JavaScript "number" type.  The behavior of the default option
   * JS_NORMAL is implementation dependent.
   *
   * This option is an enum to permit additional types to be added, e.g.
   * goog.math.Integer.
   */
  jstype: FieldOptions_JSType;
  /**
   * Should this field be parsed lazily?  Lazy applies only to message-type
   * fields.  It means that when the outer message is initially parsed, the
   * inner message's contents will not be parsed but instead stored in encoded
   * form.  The inner message will actually be parsed when it is first accessed.
   *
   * This is only a hint.  Implementations are free to choose whether to use
   * eager or lazy parsing regardless of the value of this option.  However,
   * setting this option true suggests that the protocol author believes that
   * using lazy parsing on this field is worth the additional bookkeeping
   * overhead typically needed to implement it.
   *
   * This option does not affect the public interface of any generated code;
   * all method signatures remain the same.  Furthermore, thread-safety of the
   * interface is not affected by this option; const methods remain safe to
   * call from multiple threads concurrently, while non-const methods continue
   * to require exclusive access.
   *
   * Note that lazy message fields are still eagerly verified to check
   * ill-formed wireformat or missing required fields. Calling IsInitialized()
   * on the outer message would fail if the inner message has missing required
   * fields. Failed verification would result in parsing failure (except when
   * uninitialized messages are acceptable).
   */
  lazy: boolean;
  /**
   * unverified_lazy does no correctness checks on the byte stream. This should
   * only be used where lazy with verification is prohibitive for performance
   * reasons.
   */
  unverifiedLazy: boolean;
  /**
   * Is this field deprecated?
   * Depending on the target platform, this can emit Deprecated annotations
   * for accessors, or it will be completely ignored; in the very least, this
   * is a formalization for deprecating fields.
   */
  deprecated: boolean;
  /** For Google-internal migration only. Do not use. */
  weak: boolean;
  /**
   * Indicate that the field value should not be printed out when using debug
   * formats, e.g. when the field contains sensitive credentials.
   */
  debugRedact: boolean;
  retention: FieldOptions_OptionRetention;
  targets: FieldOptions_OptionTargetType[];
  editionDefaults: FieldOptions_EditionDefault[];
  /** Any features defined in the specific edition. */
  features: FeatureSet | undefined;
  featureSupport:
    | FieldOptions_FeatureSupport
    | undefined;
  /** The parser stores options it doesn't recognize here. See above. */
  uninterpretedOption: UninterpretedOption[];
}

export enum FieldOptions_CType {
  /** STRING - Default mode. */
  STRING = "STRING",
  /**
   * CORD - The option [ctype=CORD] may be applied to a non-repeated field of type
   * "bytes". It indicates that in C++, the data should be stored in a Cord
   * instead of a string.  For very large strings, this may reduce memory
   * fragmentation. It may also allow better performance when parsing from a
   * Cord, or when parsing with aliasing enabled, as the parsed Cord may then
   * alias the original buffer.
   */
  CORD = "CORD",
  STRING_PIECE = "STRING_PIECE",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function fieldOptions_CTypeFromJSON(object: any): FieldOptions_CType {
  switch (object) {
    case 0:
    case "STRING":
      return FieldOptions_CType.STRING;
    case 1:
    case "CORD":
      return FieldOptions_CType.CORD;
    case 2:
    case "STRING_PIECE":
      return FieldOptions_CType.STRING_PIECE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return FieldOptions_CType.UNRECOGNIZED;
  }
}

export function fieldOptions_CTypeToJSON(object: FieldOptions_CType): string {
  switch (object) {
    case FieldOptions_CType.STRING:
      return "STRING";
    case FieldOptions_CType.CORD:
      return "CORD";
    case FieldOptions_CType.STRING_PIECE:
      return "STRING_PIECE";
    case FieldOptions_CType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function fieldOptions_CTypeToNumber(object: FieldOptions_CType): number {
  switch (object) {
    case FieldOptions_CType.STRING:
      return 0;
    case FieldOptions_CType.CORD:
      return 1;
    case FieldOptions_CType.STRING_PIECE:
      return 2;
    case FieldOptions_CType.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum FieldOptions_JSType {
  /** JS_NORMAL - Use the default type. */
  JS_NORMAL = "JS_NORMAL",
  /** JS_STRING - Use JavaScript strings. */
  JS_STRING = "JS_STRING",
  /** JS_NUMBER - Use JavaScript numbers. */
  JS_NUMBER = "JS_NUMBER",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function fieldOptions_JSTypeFromJSON(object: any): FieldOptions_JSType {
  switch (object) {
    case 0:
    case "JS_NORMAL":
      return FieldOptions_JSType.JS_NORMAL;
    case 1:
    case "JS_STRING":
      return FieldOptions_JSType.JS_STRING;
    case 2:
    case "JS_NUMBER":
      return FieldOptions_JSType.JS_NUMBER;
    case -1:
    case "UNRECOGNIZED":
    default:
      return FieldOptions_JSType.UNRECOGNIZED;
  }
}

export function fieldOptions_JSTypeToJSON(object: FieldOptions_JSType): string {
  switch (object) {
    case FieldOptions_JSType.JS_NORMAL:
      return "JS_NORMAL";
    case FieldOptions_JSType.JS_STRING:
      return "JS_STRING";
    case FieldOptions_JSType.JS_NUMBER:
      return "JS_NUMBER";
    case FieldOptions_JSType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function fieldOptions_JSTypeToNumber(object: FieldOptions_JSType): number {
  switch (object) {
    case FieldOptions_JSType.JS_NORMAL:
      return 0;
    case FieldOptions_JSType.JS_STRING:
      return 1;
    case FieldOptions_JSType.JS_NUMBER:
      return 2;
    case FieldOptions_JSType.UNRECOGNIZED:
    default:
      return -1;
  }
}

/**
 * If set to RETENTION_SOURCE, the option will be omitted from the binary.
 * Note: as of January 2023, support for this is in progress and does not yet
 * have an effect (b/264593489).
 */
export enum FieldOptions_OptionRetention {
  RETENTION_UNKNOWN = "RETENTION_UNKNOWN",
  RETENTION_RUNTIME = "RETENTION_RUNTIME",
  RETENTION_SOURCE = "RETENTION_SOURCE",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function fieldOptions_OptionRetentionFromJSON(object: any): FieldOptions_OptionRetention {
  switch (object) {
    case 0:
    case "RETENTION_UNKNOWN":
      return FieldOptions_OptionRetention.RETENTION_UNKNOWN;
    case 1:
    case "RETENTION_RUNTIME":
      return FieldOptions_OptionRetention.RETENTION_RUNTIME;
    case 2:
    case "RETENTION_SOURCE":
      return FieldOptions_OptionRetention.RETENTION_SOURCE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return FieldOptions_OptionRetention.UNRECOGNIZED;
  }
}

export function fieldOptions_OptionRetentionToJSON(object: FieldOptions_OptionRetention): string {
  switch (object) {
    case FieldOptions_OptionRetention.RETENTION_UNKNOWN:
      return "RETENTION_UNKNOWN";
    case FieldOptions_OptionRetention.RETENTION_RUNTIME:
      return "RETENTION_RUNTIME";
    case FieldOptions_OptionRetention.RETENTION_SOURCE:
      return "RETENTION_SOURCE";
    case FieldOptions_OptionRetention.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function fieldOptions_OptionRetentionToNumber(object: FieldOptions_OptionRetention): number {
  switch (object) {
    case FieldOptions_OptionRetention.RETENTION_UNKNOWN:
      return 0;
    case FieldOptions_OptionRetention.RETENTION_RUNTIME:
      return 1;
    case FieldOptions_OptionRetention.RETENTION_SOURCE:
      return 2;
    case FieldOptions_OptionRetention.UNRECOGNIZED:
    default:
      return -1;
  }
}

/**
 * This indicates the types of entities that the field may apply to when used
 * as an option. If it is unset, then the field may be freely used as an
 * option on any kind of entity. Note: as of January 2023, support for this is
 * in progress and does not yet have an effect (b/264593489).
 */
export enum FieldOptions_OptionTargetType {
  TARGET_TYPE_UNKNOWN = "TARGET_TYPE_UNKNOWN",
  TARGET_TYPE_FILE = "TARGET_TYPE_FILE",
  TARGET_TYPE_EXTENSION_RANGE = "TARGET_TYPE_EXTENSION_RANGE",
  TARGET_TYPE_MESSAGE = "TARGET_TYPE_MESSAGE",
  TARGET_TYPE_FIELD = "TARGET_TYPE_FIELD",
  TARGET_TYPE_ONEOF = "TARGET_TYPE_ONEOF",
  TARGET_TYPE_ENUM = "TARGET_TYPE_ENUM",
  TARGET_TYPE_ENUM_ENTRY = "TARGET_TYPE_ENUM_ENTRY",
  TARGET_TYPE_SERVICE = "TARGET_TYPE_SERVICE",
  TARGET_TYPE_METHOD = "TARGET_TYPE_METHOD",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function fieldOptions_OptionTargetTypeFromJSON(object: any): FieldOptions_OptionTargetType {
  switch (object) {
    case 0:
    case "TARGET_TYPE_UNKNOWN":
      return FieldOptions_OptionTargetType.TARGET_TYPE_UNKNOWN;
    case 1:
    case "TARGET_TYPE_FILE":
      return FieldOptions_OptionTargetType.TARGET_TYPE_FILE;
    case 2:
    case "TARGET_TYPE_EXTENSION_RANGE":
      return FieldOptions_OptionTargetType.TARGET_TYPE_EXTENSION_RANGE;
    case 3:
    case "TARGET_TYPE_MESSAGE":
      return FieldOptions_OptionTargetType.TARGET_TYPE_MESSAGE;
    case 4:
    case "TARGET_TYPE_FIELD":
      return FieldOptions_OptionTargetType.TARGET_TYPE_FIELD;
    case 5:
    case "TARGET_TYPE_ONEOF":
      return FieldOptions_OptionTargetType.TARGET_TYPE_ONEOF;
    case 6:
    case "TARGET_TYPE_ENUM":
      return FieldOptions_OptionTargetType.TARGET_TYPE_ENUM;
    case 7:
    case "TARGET_TYPE_ENUM_ENTRY":
      return FieldOptions_OptionTargetType.TARGET_TYPE_ENUM_ENTRY;
    case 8:
    case "TARGET_TYPE_SERVICE":
      return FieldOptions_OptionTargetType.TARGET_TYPE_SERVICE;
    case 9:
    case "TARGET_TYPE_METHOD":
      return FieldOptions_OptionTargetType.TARGET_TYPE_METHOD;
    case -1:
    case "UNRECOGNIZED":
    default:
      return FieldOptions_OptionTargetType.UNRECOGNIZED;
  }
}

export function fieldOptions_OptionTargetTypeToJSON(object: FieldOptions_OptionTargetType): string {
  switch (object) {
    case FieldOptions_OptionTargetType.TARGET_TYPE_UNKNOWN:
      return "TARGET_TYPE_UNKNOWN";
    case FieldOptions_OptionTargetType.TARGET_TYPE_FILE:
      return "TARGET_TYPE_FILE";
    case FieldOptions_OptionTargetType.TARGET_TYPE_EXTENSION_RANGE:
      return "TARGET_TYPE_EXTENSION_RANGE";
    case FieldOptions_OptionTargetType.TARGET_TYPE_MESSAGE:
      return "TARGET_TYPE_MESSAGE";
    case FieldOptions_OptionTargetType.TARGET_TYPE_FIELD:
      return "TARGET_TYPE_FIELD";
    case FieldOptions_OptionTargetType.TARGET_TYPE_ONEOF:
      return "TARGET_TYPE_ONEOF";
    case FieldOptions_OptionTargetType.TARGET_TYPE_ENUM:
      return "TARGET_TYPE_ENUM";
    case FieldOptions_OptionTargetType.TARGET_TYPE_ENUM_ENTRY:
      return "TARGET_TYPE_ENUM_ENTRY";
    case FieldOptions_OptionTargetType.TARGET_TYPE_SERVICE:
      return "TARGET_TYPE_SERVICE";
    case FieldOptions_OptionTargetType.TARGET_TYPE_METHOD:
      return "TARGET_TYPE_METHOD";
    case FieldOptions_OptionTargetType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function fieldOptions_OptionTargetTypeToNumber(object: FieldOptions_OptionTargetType): number {
  switch (object) {
    case FieldOptions_OptionTargetType.TARGET_TYPE_UNKNOWN:
      return 0;
    case FieldOptions_OptionTargetType.TARGET_TYPE_FILE:
      return 1;
    case FieldOptions_OptionTargetType.TARGET_TYPE_EXTENSION_RANGE:
      return 2;
    case FieldOptions_OptionTargetType.TARGET_TYPE_MESSAGE:
      return 3;
    case FieldOptions_OptionTargetType.TARGET_TYPE_FIELD:
      return 4;
    case FieldOptions_OptionTargetType.TARGET_TYPE_ONEOF:
      return 5;
    case FieldOptions_OptionTargetType.TARGET_TYPE_ENUM:
      return 6;
    case FieldOptions_OptionTargetType.TARGET_TYPE_ENUM_ENTRY:
      return 7;
    case FieldOptions_OptionTargetType.TARGET_TYPE_SERVICE:
      return 8;
    case FieldOptions_OptionTargetType.TARGET_TYPE_METHOD:
      return 9;
    case FieldOptions_OptionTargetType.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface FieldOptions_EditionDefault {
  edition: Edition;
  /** Textproto value. */
  value: string;
}

/** Information about the support window of a feature. */
export interface FieldOptions_FeatureSupport {
  /**
   * The edition that this feature was first available in.  In editions
   * earlier than this one, the default assigned to EDITION_LEGACY will be
   * used, and proto files will not be able to override it.
   */
  editionIntroduced: Edition;
  /**
   * The edition this feature becomes deprecated in.  Using this after this
   * edition may trigger warnings.
   */
  editionDeprecated: Edition;
  /**
   * The deprecation warning text if this feature is used after the edition it
   * was marked deprecated in.
   */
  deprecationWarning: string;
  /**
   * The edition this feature is no longer available in.  In editions after
   * this one, the last default assigned will be used, and proto files will
   * not be able to override it.
   */
  editionRemoved: Edition;
}

export interface OneofOptions {
  /** Any features defined in the specific edition. */
  features:
    | FeatureSet
    | undefined;
  /** The parser stores options it doesn't recognize here. See above. */
  uninterpretedOption: UninterpretedOption[];
}

export interface EnumOptions {
  /**
   * Set this option to true to allow mapping different tag names to the same
   * value.
   */
  allowAlias: boolean;
  /**
   * Is this enum deprecated?
   * Depending on the target platform, this can emit Deprecated annotations
   * for the enum, or it will be completely ignored; in the very least, this
   * is a formalization for deprecating enums.
   */
  deprecated: boolean;
  /**
   * Enable the legacy handling of JSON field name conflicts.  This lowercases
   * and strips underscored from the fields before comparison in proto3 only.
   * The new behavior takes `json_name` into account and applies to proto2 as
   * well.
   * TODO Remove this legacy behavior once downstream teams have
   * had time to migrate.
   *
   * @deprecated
   */
  deprecatedLegacyJsonFieldConflicts: boolean;
  /** Any features defined in the specific edition. */
  features:
    | FeatureSet
    | undefined;
  /** The parser stores options it doesn't recognize here. See above. */
  uninterpretedOption: UninterpretedOption[];
}

export interface EnumValueOptions {
  /**
   * Is this enum value deprecated?
   * Depending on the target platform, this can emit Deprecated annotations
   * for the enum value, or it will be completely ignored; in the very least,
   * this is a formalization for deprecating enum values.
   */
  deprecated: boolean;
  /** Any features defined in the specific edition. */
  features:
    | FeatureSet
    | undefined;
  /**
   * Indicate that fields annotated with this enum value should not be printed
   * out when using debug formats, e.g. when the field contains sensitive
   * credentials.
   */
  debugRedact: boolean;
  /** Information about the support window of a feature value. */
  featureSupport:
    | FieldOptions_FeatureSupport
    | undefined;
  /** The parser stores options it doesn't recognize here. See above. */
  uninterpretedOption: UninterpretedOption[];
}

export interface ServiceOptions {
  /** Any features defined in the specific edition. */
  features:
    | FeatureSet
    | undefined;
  /**
   * Is this service deprecated?
   * Depending on the target platform, this can emit Deprecated annotations
   * for the service, or it will be completely ignored; in the very least,
   * this is a formalization for deprecating services.
   */
  deprecated: boolean;
  /** The parser stores options it doesn't recognize here. See above. */
  uninterpretedOption: UninterpretedOption[];
}

export interface MethodOptions {
  /**
   * Is this method deprecated?
   * Depending on the target platform, this can emit Deprecated annotations
   * for the method, or it will be completely ignored; in the very least,
   * this is a formalization for deprecating methods.
   */
  deprecated: boolean;
  idempotencyLevel: MethodOptions_IdempotencyLevel;
  /** Any features defined in the specific edition. */
  features:
    | FeatureSet
    | undefined;
  /** The parser stores options it doesn't recognize here. See above. */
  uninterpretedOption: UninterpretedOption[];
}

/**
 * Is this method side-effect-free (or safe in HTTP parlance), or idempotent,
 * or neither? HTTP based RPC implementation may choose GET verb for safe
 * methods, and PUT verb for idempotent methods instead of the default POST.
 */
export enum MethodOptions_IdempotencyLevel {
  IDEMPOTENCY_UNKNOWN = "IDEMPOTENCY_UNKNOWN",
  /** NO_SIDE_EFFECTS - implies idempotent */
  NO_SIDE_EFFECTS = "NO_SIDE_EFFECTS",
  /** IDEMPOTENT - idempotent, but may have side effects */
  IDEMPOTENT = "IDEMPOTENT",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function methodOptions_IdempotencyLevelFromJSON(object: any): MethodOptions_IdempotencyLevel {
  switch (object) {
    case 0:
    case "IDEMPOTENCY_UNKNOWN":
      return MethodOptions_IdempotencyLevel.IDEMPOTENCY_UNKNOWN;
    case 1:
    case "NO_SIDE_EFFECTS":
      return MethodOptions_IdempotencyLevel.NO_SIDE_EFFECTS;
    case 2:
    case "IDEMPOTENT":
      return MethodOptions_IdempotencyLevel.IDEMPOTENT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return MethodOptions_IdempotencyLevel.UNRECOGNIZED;
  }
}

export function methodOptions_IdempotencyLevelToJSON(object: MethodOptions_IdempotencyLevel): string {
  switch (object) {
    case MethodOptions_IdempotencyLevel.IDEMPOTENCY_UNKNOWN:
      return "IDEMPOTENCY_UNKNOWN";
    case MethodOptions_IdempotencyLevel.NO_SIDE_EFFECTS:
      return "NO_SIDE_EFFECTS";
    case MethodOptions_IdempotencyLevel.IDEMPOTENT:
      return "IDEMPOTENT";
    case MethodOptions_IdempotencyLevel.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function methodOptions_IdempotencyLevelToNumber(object: MethodOptions_IdempotencyLevel): number {
  switch (object) {
    case MethodOptions_IdempotencyLevel.IDEMPOTENCY_UNKNOWN:
      return 0;
    case MethodOptions_IdempotencyLevel.NO_SIDE_EFFECTS:
      return 1;
    case MethodOptions_IdempotencyLevel.IDEMPOTENT:
      return 2;
    case MethodOptions_IdempotencyLevel.UNRECOGNIZED:
    default:
      return -1;
  }
}

/**
 * A message representing a option the parser does not recognize. This only
 * appears in options protos created by the compiler::Parser class.
 * DescriptorPool resolves these when building Descriptor objects. Therefore,
 * options protos in descriptor objects (e.g. returned by Descriptor::options(),
 * or produced by Descriptor::CopyTo()) will never have UninterpretedOptions
 * in them.
 */
export interface UninterpretedOption {
  name: UninterpretedOption_NamePart[];
  /**
   * The value of the uninterpreted option, in whatever type the tokenizer
   * identified it as during parsing. Exactly one of these should be set.
   */
  identifierValue: string;
  positiveIntValue: Long;
  negativeIntValue: Long;
  doubleValue: number;
  stringValue: Uint8Array;
  aggregateValue: string;
}

/**
 * The name of the uninterpreted option.  Each string represents a segment in
 * a dot-separated name.  is_extension is true iff a segment represents an
 * extension (denoted with parentheses in options specs in .proto files).
 * E.g.,{ ["foo", false], ["bar.baz", true], ["moo", false] } represents
 * "foo.(bar.baz).moo".
 */
export interface UninterpretedOption_NamePart {
  namePart: string;
  isExtension: boolean;
}

/**
 * TODO Enums in C++ gencode (and potentially other languages) are
 * not well scoped.  This means that each of the feature enums below can clash
 * with each other.  The short names we've chosen maximize call-site
 * readability, but leave us very open to this scenario.  A future feature will
 * be designed and implemented to handle this, hopefully before we ever hit a
 * conflict here.
 */
export interface FeatureSet {
  fieldPresence: FeatureSet_FieldPresence;
  enumType: FeatureSet_EnumType;
  repeatedFieldEncoding: FeatureSet_RepeatedFieldEncoding;
  utf8Validation: FeatureSet_Utf8Validation;
  messageEncoding: FeatureSet_MessageEncoding;
  jsonFormat: FeatureSet_JsonFormat;
}

export enum FeatureSet_FieldPresence {
  FIELD_PRESENCE_UNKNOWN = "FIELD_PRESENCE_UNKNOWN",
  EXPLICIT = "EXPLICIT",
  IMPLICIT = "IMPLICIT",
  LEGACY_REQUIRED = "LEGACY_REQUIRED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function featureSet_FieldPresenceFromJSON(object: any): FeatureSet_FieldPresence {
  switch (object) {
    case 0:
    case "FIELD_PRESENCE_UNKNOWN":
      return FeatureSet_FieldPresence.FIELD_PRESENCE_UNKNOWN;
    case 1:
    case "EXPLICIT":
      return FeatureSet_FieldPresence.EXPLICIT;
    case 2:
    case "IMPLICIT":
      return FeatureSet_FieldPresence.IMPLICIT;
    case 3:
    case "LEGACY_REQUIRED":
      return FeatureSet_FieldPresence.LEGACY_REQUIRED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return FeatureSet_FieldPresence.UNRECOGNIZED;
  }
}

export function featureSet_FieldPresenceToJSON(object: FeatureSet_FieldPresence): string {
  switch (object) {
    case FeatureSet_FieldPresence.FIELD_PRESENCE_UNKNOWN:
      return "FIELD_PRESENCE_UNKNOWN";
    case FeatureSet_FieldPresence.EXPLICIT:
      return "EXPLICIT";
    case FeatureSet_FieldPresence.IMPLICIT:
      return "IMPLICIT";
    case FeatureSet_FieldPresence.LEGACY_REQUIRED:
      return "LEGACY_REQUIRED";
    case FeatureSet_FieldPresence.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function featureSet_FieldPresenceToNumber(object: FeatureSet_FieldPresence): number {
  switch (object) {
    case FeatureSet_FieldPresence.FIELD_PRESENCE_UNKNOWN:
      return 0;
    case FeatureSet_FieldPresence.EXPLICIT:
      return 1;
    case FeatureSet_FieldPresence.IMPLICIT:
      return 2;
    case FeatureSet_FieldPresence.LEGACY_REQUIRED:
      return 3;
    case FeatureSet_FieldPresence.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum FeatureSet_EnumType {
  ENUM_TYPE_UNKNOWN = "ENUM_TYPE_UNKNOWN",
  OPEN = "OPEN",
  CLOSED = "CLOSED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function featureSet_EnumTypeFromJSON(object: any): FeatureSet_EnumType {
  switch (object) {
    case 0:
    case "ENUM_TYPE_UNKNOWN":
      return FeatureSet_EnumType.ENUM_TYPE_UNKNOWN;
    case 1:
    case "OPEN":
      return FeatureSet_EnumType.OPEN;
    case 2:
    case "CLOSED":
      return FeatureSet_EnumType.CLOSED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return FeatureSet_EnumType.UNRECOGNIZED;
  }
}

export function featureSet_EnumTypeToJSON(object: FeatureSet_EnumType): string {
  switch (object) {
    case FeatureSet_EnumType.ENUM_TYPE_UNKNOWN:
      return "ENUM_TYPE_UNKNOWN";
    case FeatureSet_EnumType.OPEN:
      return "OPEN";
    case FeatureSet_EnumType.CLOSED:
      return "CLOSED";
    case FeatureSet_EnumType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function featureSet_EnumTypeToNumber(object: FeatureSet_EnumType): number {
  switch (object) {
    case FeatureSet_EnumType.ENUM_TYPE_UNKNOWN:
      return 0;
    case FeatureSet_EnumType.OPEN:
      return 1;
    case FeatureSet_EnumType.CLOSED:
      return 2;
    case FeatureSet_EnumType.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum FeatureSet_RepeatedFieldEncoding {
  REPEATED_FIELD_ENCODING_UNKNOWN = "REPEATED_FIELD_ENCODING_UNKNOWN",
  PACKED = "PACKED",
  EXPANDED = "EXPANDED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function featureSet_RepeatedFieldEncodingFromJSON(object: any): FeatureSet_RepeatedFieldEncoding {
  switch (object) {
    case 0:
    case "REPEATED_FIELD_ENCODING_UNKNOWN":
      return FeatureSet_RepeatedFieldEncoding.REPEATED_FIELD_ENCODING_UNKNOWN;
    case 1:
    case "PACKED":
      return FeatureSet_RepeatedFieldEncoding.PACKED;
    case 2:
    case "EXPANDED":
      return FeatureSet_RepeatedFieldEncoding.EXPANDED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return FeatureSet_RepeatedFieldEncoding.UNRECOGNIZED;
  }
}

export function featureSet_RepeatedFieldEncodingToJSON(object: FeatureSet_RepeatedFieldEncoding): string {
  switch (object) {
    case FeatureSet_RepeatedFieldEncoding.REPEATED_FIELD_ENCODING_UNKNOWN:
      return "REPEATED_FIELD_ENCODING_UNKNOWN";
    case FeatureSet_RepeatedFieldEncoding.PACKED:
      return "PACKED";
    case FeatureSet_RepeatedFieldEncoding.EXPANDED:
      return "EXPANDED";
    case FeatureSet_RepeatedFieldEncoding.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function featureSet_RepeatedFieldEncodingToNumber(object: FeatureSet_RepeatedFieldEncoding): number {
  switch (object) {
    case FeatureSet_RepeatedFieldEncoding.REPEATED_FIELD_ENCODING_UNKNOWN:
      return 0;
    case FeatureSet_RepeatedFieldEncoding.PACKED:
      return 1;
    case FeatureSet_RepeatedFieldEncoding.EXPANDED:
      return 2;
    case FeatureSet_RepeatedFieldEncoding.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum FeatureSet_Utf8Validation {
  UTF8_VALIDATION_UNKNOWN = "UTF8_VALIDATION_UNKNOWN",
  VERIFY = "VERIFY",
  NONE = "NONE",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function featureSet_Utf8ValidationFromJSON(object: any): FeatureSet_Utf8Validation {
  switch (object) {
    case 0:
    case "UTF8_VALIDATION_UNKNOWN":
      return FeatureSet_Utf8Validation.UTF8_VALIDATION_UNKNOWN;
    case 2:
    case "VERIFY":
      return FeatureSet_Utf8Validation.VERIFY;
    case 3:
    case "NONE":
      return FeatureSet_Utf8Validation.NONE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return FeatureSet_Utf8Validation.UNRECOGNIZED;
  }
}

export function featureSet_Utf8ValidationToJSON(object: FeatureSet_Utf8Validation): string {
  switch (object) {
    case FeatureSet_Utf8Validation.UTF8_VALIDATION_UNKNOWN:
      return "UTF8_VALIDATION_UNKNOWN";
    case FeatureSet_Utf8Validation.VERIFY:
      return "VERIFY";
    case FeatureSet_Utf8Validation.NONE:
      return "NONE";
    case FeatureSet_Utf8Validation.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function featureSet_Utf8ValidationToNumber(object: FeatureSet_Utf8Validation): number {
  switch (object) {
    case FeatureSet_Utf8Validation.UTF8_VALIDATION_UNKNOWN:
      return 0;
    case FeatureSet_Utf8Validation.VERIFY:
      return 2;
    case FeatureSet_Utf8Validation.NONE:
      return 3;
    case FeatureSet_Utf8Validation.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum FeatureSet_MessageEncoding {
  MESSAGE_ENCODING_UNKNOWN = "MESSAGE_ENCODING_UNKNOWN",
  LENGTH_PREFIXED = "LENGTH_PREFIXED",
  DELIMITED = "DELIMITED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function featureSet_MessageEncodingFromJSON(object: any): FeatureSet_MessageEncoding {
  switch (object) {
    case 0:
    case "MESSAGE_ENCODING_UNKNOWN":
      return FeatureSet_MessageEncoding.MESSAGE_ENCODING_UNKNOWN;
    case 1:
    case "LENGTH_PREFIXED":
      return FeatureSet_MessageEncoding.LENGTH_PREFIXED;
    case 2:
    case "DELIMITED":
      return FeatureSet_MessageEncoding.DELIMITED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return FeatureSet_MessageEncoding.UNRECOGNIZED;
  }
}

export function featureSet_MessageEncodingToJSON(object: FeatureSet_MessageEncoding): string {
  switch (object) {
    case FeatureSet_MessageEncoding.MESSAGE_ENCODING_UNKNOWN:
      return "MESSAGE_ENCODING_UNKNOWN";
    case FeatureSet_MessageEncoding.LENGTH_PREFIXED:
      return "LENGTH_PREFIXED";
    case FeatureSet_MessageEncoding.DELIMITED:
      return "DELIMITED";
    case FeatureSet_MessageEncoding.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function featureSet_MessageEncodingToNumber(object: FeatureSet_MessageEncoding): number {
  switch (object) {
    case FeatureSet_MessageEncoding.MESSAGE_ENCODING_UNKNOWN:
      return 0;
    case FeatureSet_MessageEncoding.LENGTH_PREFIXED:
      return 1;
    case FeatureSet_MessageEncoding.DELIMITED:
      return 2;
    case FeatureSet_MessageEncoding.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum FeatureSet_JsonFormat {
  JSON_FORMAT_UNKNOWN = "JSON_FORMAT_UNKNOWN",
  ALLOW = "ALLOW",
  LEGACY_BEST_EFFORT = "LEGACY_BEST_EFFORT",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function featureSet_JsonFormatFromJSON(object: any): FeatureSet_JsonFormat {
  switch (object) {
    case 0:
    case "JSON_FORMAT_UNKNOWN":
      return FeatureSet_JsonFormat.JSON_FORMAT_UNKNOWN;
    case 1:
    case "ALLOW":
      return FeatureSet_JsonFormat.ALLOW;
    case 2:
    case "LEGACY_BEST_EFFORT":
      return FeatureSet_JsonFormat.LEGACY_BEST_EFFORT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return FeatureSet_JsonFormat.UNRECOGNIZED;
  }
}

export function featureSet_JsonFormatToJSON(object: FeatureSet_JsonFormat): string {
  switch (object) {
    case FeatureSet_JsonFormat.JSON_FORMAT_UNKNOWN:
      return "JSON_FORMAT_UNKNOWN";
    case FeatureSet_JsonFormat.ALLOW:
      return "ALLOW";
    case FeatureSet_JsonFormat.LEGACY_BEST_EFFORT:
      return "LEGACY_BEST_EFFORT";
    case FeatureSet_JsonFormat.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function featureSet_JsonFormatToNumber(object: FeatureSet_JsonFormat): number {
  switch (object) {
    case FeatureSet_JsonFormat.JSON_FORMAT_UNKNOWN:
      return 0;
    case FeatureSet_JsonFormat.ALLOW:
      return 1;
    case FeatureSet_JsonFormat.LEGACY_BEST_EFFORT:
      return 2;
    case FeatureSet_JsonFormat.UNRECOGNIZED:
    default:
      return -1;
  }
}

/**
 * A compiled specification for the defaults of a set of features.  These
 * messages are generated from FeatureSet extensions and can be used to seed
 * feature resolution. The resolution with this object becomes a simple search
 * for the closest matching edition, followed by proto merges.
 */
export interface FeatureSetDefaults {
  defaults: FeatureSetDefaults_FeatureSetEditionDefault[];
  /**
   * The minimum supported edition (inclusive) when this was constructed.
   * Editions before this will not have defaults.
   */
  minimumEdition: Edition;
  /**
   * The maximum known edition (inclusive) when this was constructed. Editions
   * after this will not have reliable defaults.
   */
  maximumEdition: Edition;
}

/**
 * A map from every known edition with a unique set of defaults to its
 * defaults. Not all editions may be contained here.  For a given edition,
 * the defaults at the closest matching edition ordered at or before it should
 * be used.  This field must be in strict ascending order by edition.
 */
export interface FeatureSetDefaults_FeatureSetEditionDefault {
  edition: Edition;
  /** Defaults of features that can be overridden in this edition. */
  overridableFeatures:
    | FeatureSet
    | undefined;
  /** Defaults of features that can't be overridden in this edition. */
  fixedFeatures: FeatureSet | undefined;
}

/**
 * Encapsulates information about the original source file from which a
 * FileDescriptorProto was generated.
 */
export interface SourceCodeInfo {
  /**
   * A Location identifies a piece of source code in a .proto file which
   * corresponds to a particular definition.  This information is intended
   * to be useful to IDEs, code indexers, documentation generators, and similar
   * tools.
   *
   * For example, say we have a file like:
   *   message Foo {
   *     optional string foo = 1;
   *   }
   * Let's look at just the field definition:
   *   optional string foo = 1;
   *   ^       ^^     ^^  ^  ^^^
   *   a       bc     de  f  ghi
   * We have the following locations:
   *   span   path               represents
   *   [a,i)  [ 4, 0, 2, 0 ]     The whole field definition.
   *   [a,b)  [ 4, 0, 2, 0, 4 ]  The label (optional).
   *   [c,d)  [ 4, 0, 2, 0, 5 ]  The type (string).
   *   [e,f)  [ 4, 0, 2, 0, 1 ]  The name (foo).
   *   [g,h)  [ 4, 0, 2, 0, 3 ]  The number (1).
   *
   * Notes:
   * - A location may refer to a repeated field itself (i.e. not to any
   *   particular index within it).  This is used whenever a set of elements are
   *   logically enclosed in a single code segment.  For example, an entire
   *   extend block (possibly containing multiple extension definitions) will
   *   have an outer location whose path refers to the "extensions" repeated
   *   field without an index.
   * - Multiple locations may have the same path.  This happens when a single
   *   logical declaration is spread out across multiple places.  The most
   *   obvious example is the "extend" block again -- there may be multiple
   *   extend blocks in the same scope, each of which will have the same path.
   * - A location's span is not always a subset of its parent's span.  For
   *   example, the "extendee" of an extension declaration appears at the
   *   beginning of the "extend" block and is shared by all extensions within
   *   the block.
   * - Just because a location's span is a subset of some other location's span
   *   does not mean that it is a descendant.  For example, a "group" defines
   *   both a type and a field in a single declaration.  Thus, the locations
   *   corresponding to the type and field and their components will overlap.
   * - Code which tries to interpret locations should probably be designed to
   *   ignore those that it doesn't understand, as more types of locations could
   *   be recorded in the future.
   */
  location: SourceCodeInfo_Location[];
}

export interface SourceCodeInfo_Location {
  /**
   * Identifies which part of the FileDescriptorProto was defined at this
   * location.
   *
   * Each element is a field number or an index.  They form a path from
   * the root FileDescriptorProto to the place where the definition appears.
   * For example, this path:
   *   [ 4, 3, 2, 7, 1 ]
   * refers to:
   *   file.message_type(3)  // 4, 3
   *       .field(7)         // 2, 7
   *       .name()           // 1
   * This is because FileDescriptorProto.message_type has field number 4:
   *   repeated DescriptorProto message_type = 4;
   * and DescriptorProto.field has field number 2:
   *   repeated FieldDescriptorProto field = 2;
   * and FieldDescriptorProto.name has field number 1:
   *   optional string name = 1;
   *
   * Thus, the above path gives the location of a field name.  If we removed
   * the last element:
   *   [ 4, 3, 2, 7 ]
   * this path refers to the whole field declaration (from the beginning
   * of the label to the terminating semicolon).
   */
  path: number[];
  /**
   * Always has exactly three or four elements: start line, start column,
   * end line (optional, otherwise assumed same as start line), end column.
   * These are packed into a single field for efficiency.  Note that line
   * and column numbers are zero-based -- typically you will want to add
   * 1 to each before displaying to a user.
   */
  span: number[];
  /**
   * If this SourceCodeInfo represents a complete declaration, these are any
   * comments appearing before and after the declaration which appear to be
   * attached to the declaration.
   *
   * A series of line comments appearing on consecutive lines, with no other
   * tokens appearing on those lines, will be treated as a single comment.
   *
   * leading_detached_comments will keep paragraphs of comments that appear
   * before (but not connected to) the current element. Each paragraph,
   * separated by empty lines, will be one comment element in the repeated
   * field.
   *
   * Only the comment content is provided; comment markers (e.g. //) are
   * stripped out.  For block comments, leading whitespace and an asterisk
   * will be stripped from the beginning of each line other than the first.
   * Newlines are included in the output.
   *
   * Examples:
   *
   *   optional int32 foo = 1;  // Comment attached to foo.
   *   // Comment attached to bar.
   *   optional int32 bar = 2;
   *
   *   optional string baz = 3;
   *   // Comment attached to baz.
   *   // Another line attached to baz.
   *
   *   // Comment attached to moo.
   *   //
   *   // Another line attached to moo.
   *   optional double moo = 4;
   *
   *   // Detached comment for corge. This is not leading or trailing comments
   *   // to moo or corge because there are blank lines separating it from
   *   // both.
   *
   *   // Detached comment for corge paragraph 2.
   *
   *   optional string corge = 5;
   *   /* Block comment attached
   *    * to corge.  Leading asterisks
   *    * will be removed. * /
   *   /* Block comment attached to
   *    * grault. * /
   *   optional int32 grault = 6;
   *
   *   // ignored detached comments.
   */
  leadingComments: string;
  trailingComments: string;
  leadingDetachedComments: string[];
}

/**
 * Describes the relationship between generated code and its original source
 * file. A GeneratedCodeInfo message is associated with only one generated
 * source file, but may contain references to different source .proto files.
 */
export interface GeneratedCodeInfo {
  /**
   * An Annotation connects some span of text in generated code to an element
   * of its generating .proto file.
   */
  annotation: GeneratedCodeInfo_Annotation[];
}

export interface GeneratedCodeInfo_Annotation {
  /**
   * Identifies the element in the original source .proto file. This field
   * is formatted the same as SourceCodeInfo.Location.path.
   */
  path: number[];
  /** Identifies the filesystem path to the original source .proto. */
  sourceFile: string;
  /**
   * Identifies the starting offset in bytes in the generated code
   * that relates to the identified object.
   */
  begin: number;
  /**
   * Identifies the ending offset in bytes in the generated code that
   * relates to the identified object. The end offset should be one past
   * the last relevant byte (so the length of the text = end - begin).
   */
  end: number;
  semantic: GeneratedCodeInfo_Annotation_Semantic;
}

/**
 * Represents the identified object's effect on the element in the original
 * .proto file.
 */
export enum GeneratedCodeInfo_Annotation_Semantic {
  /** NONE - There is no effect or the effect is indescribable. */
  NONE = "NONE",
  /** SET - The element is set or otherwise mutated. */
  SET = "SET",
  /** ALIAS - An alias to the element is returned. */
  ALIAS = "ALIAS",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function generatedCodeInfo_Annotation_SemanticFromJSON(object: any): GeneratedCodeInfo_Annotation_Semantic {
  switch (object) {
    case 0:
    case "NONE":
      return GeneratedCodeInfo_Annotation_Semantic.NONE;
    case 1:
    case "SET":
      return GeneratedCodeInfo_Annotation_Semantic.SET;
    case 2:
    case "ALIAS":
      return GeneratedCodeInfo_Annotation_Semantic.ALIAS;
    case -1:
    case "UNRECOGNIZED":
    default:
      return GeneratedCodeInfo_Annotation_Semantic.UNRECOGNIZED;
  }
}

export function generatedCodeInfo_Annotation_SemanticToJSON(object: GeneratedCodeInfo_Annotation_Semantic): string {
  switch (object) {
    case GeneratedCodeInfo_Annotation_Semantic.NONE:
      return "NONE";
    case GeneratedCodeInfo_Annotation_Semantic.SET:
      return "SET";
    case GeneratedCodeInfo_Annotation_Semantic.ALIAS:
      return "ALIAS";
    case GeneratedCodeInfo_Annotation_Semantic.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function generatedCodeInfo_Annotation_SemanticToNumber(object: GeneratedCodeInfo_Annotation_Semantic): number {
  switch (object) {
    case GeneratedCodeInfo_Annotation_Semantic.NONE:
      return 0;
    case GeneratedCodeInfo_Annotation_Semantic.SET:
      return 1;
    case GeneratedCodeInfo_Annotation_Semantic.ALIAS:
      return 2;
    case GeneratedCodeInfo_Annotation_Semantic.UNRECOGNIZED:
    default:
      return -1;
  }
}

function createBaseFileDescriptorSet(): FileDescriptorSet {
  return { file: [] };
}

export const FileDescriptorSet = {
  encode(message: FileDescriptorSet, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.file) {
      FileDescriptorProto.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FileDescriptorSet {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFileDescriptorSet();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.file.push(FileDescriptorProto.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FileDescriptorSet {
    return {
      file: globalThis.Array.isArray(object?.file) ? object.file.map((e: any) => FileDescriptorProto.fromJSON(e)) : [],
    };
  },

  toJSON(message: FileDescriptorSet): unknown {
    const obj: any = {};
    if (message.file?.length) {
      obj.file = message.file.map((e) => FileDescriptorProto.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<FileDescriptorSet>): FileDescriptorSet {
    return FileDescriptorSet.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<FileDescriptorSet>): FileDescriptorSet {
    const message = createBaseFileDescriptorSet();
    message.file = object.file?.map((e) => FileDescriptorProto.fromPartial(e)) || [];
    return message;
  },
};

function createBaseFileDescriptorProto(): FileDescriptorProto {
  return {
    name: "",
    package: "",
    dependency: [],
    publicDependency: [],
    weakDependency: [],
    messageType: [],
    enumType: [],
    service: [],
    extension: [],
    options: undefined,
    sourceCodeInfo: undefined,
    syntax: "",
    edition: Edition.EDITION_UNKNOWN,
  };
}

export const FileDescriptorProto = {
  encode(message: FileDescriptorProto, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.package !== "") {
      writer.uint32(18).string(message.package);
    }
    for (const v of message.dependency) {
      writer.uint32(26).string(v!);
    }
    writer.uint32(82).fork();
    for (const v of message.publicDependency) {
      writer.int32(v);
    }
    writer.ldelim();
    writer.uint32(90).fork();
    for (const v of message.weakDependency) {
      writer.int32(v);
    }
    writer.ldelim();
    for (const v of message.messageType) {
      DescriptorProto.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.enumType) {
      EnumDescriptorProto.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.service) {
      ServiceDescriptorProto.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    for (const v of message.extension) {
      FieldDescriptorProto.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    if (message.options !== undefined) {
      FileOptions.encode(message.options, writer.uint32(66).fork()).ldelim();
    }
    if (message.sourceCodeInfo !== undefined) {
      SourceCodeInfo.encode(message.sourceCodeInfo, writer.uint32(74).fork()).ldelim();
    }
    if (message.syntax !== "") {
      writer.uint32(98).string(message.syntax);
    }
    if (message.edition !== Edition.EDITION_UNKNOWN) {
      writer.uint32(112).int32(editionToNumber(message.edition));
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FileDescriptorProto {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFileDescriptorProto();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.package = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.dependency.push(reader.string());
          continue;
        case 10:
          if (tag === 80) {
            message.publicDependency.push(reader.int32());

            continue;
          }

          if (tag === 82) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.publicDependency.push(reader.int32());
            }

            continue;
          }

          break;
        case 11:
          if (tag === 88) {
            message.weakDependency.push(reader.int32());

            continue;
          }

          if (tag === 90) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.weakDependency.push(reader.int32());
            }

            continue;
          }

          break;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.messageType.push(DescriptorProto.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.enumType.push(EnumDescriptorProto.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.service.push(ServiceDescriptorProto.decode(reader, reader.uint32()));
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.extension.push(FieldDescriptorProto.decode(reader, reader.uint32()));
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.options = FileOptions.decode(reader, reader.uint32());
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.sourceCodeInfo = SourceCodeInfo.decode(reader, reader.uint32());
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.syntax = reader.string();
          continue;
        case 14:
          if (tag !== 112) {
            break;
          }

          message.edition = editionFromJSON(reader.int32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FileDescriptorProto {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      package: isSet(object.package) ? globalThis.String(object.package) : "",
      dependency: globalThis.Array.isArray(object?.dependency)
        ? object.dependency.map((e: any) => globalThis.String(e))
        : [],
      publicDependency: globalThis.Array.isArray(object?.publicDependency)
        ? object.publicDependency.map((e: any) => globalThis.Number(e))
        : [],
      weakDependency: globalThis.Array.isArray(object?.weakDependency)
        ? object.weakDependency.map((e: any) => globalThis.Number(e))
        : [],
      messageType: globalThis.Array.isArray(object?.messageType)
        ? object.messageType.map((e: any) => DescriptorProto.fromJSON(e))
        : [],
      enumType: globalThis.Array.isArray(object?.enumType)
        ? object.enumType.map((e: any) => EnumDescriptorProto.fromJSON(e))
        : [],
      service: globalThis.Array.isArray(object?.service)
        ? object.service.map((e: any) => ServiceDescriptorProto.fromJSON(e))
        : [],
      extension: globalThis.Array.isArray(object?.extension)
        ? object.extension.map((e: any) => FieldDescriptorProto.fromJSON(e))
        : [],
      options: isSet(object.options) ? FileOptions.fromJSON(object.options) : undefined,
      sourceCodeInfo: isSet(object.sourceCodeInfo) ? SourceCodeInfo.fromJSON(object.sourceCodeInfo) : undefined,
      syntax: isSet(object.syntax) ? globalThis.String(object.syntax) : "",
      edition: isSet(object.edition) ? editionFromJSON(object.edition) : Edition.EDITION_UNKNOWN,
    };
  },

  toJSON(message: FileDescriptorProto): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.package !== "") {
      obj.package = message.package;
    }
    if (message.dependency?.length) {
      obj.dependency = message.dependency;
    }
    if (message.publicDependency?.length) {
      obj.publicDependency = message.publicDependency.map((e) => Math.round(e));
    }
    if (message.weakDependency?.length) {
      obj.weakDependency = message.weakDependency.map((e) => Math.round(e));
    }
    if (message.messageType?.length) {
      obj.messageType = message.messageType.map((e) => DescriptorProto.toJSON(e));
    }
    if (message.enumType?.length) {
      obj.enumType = message.enumType.map((e) => EnumDescriptorProto.toJSON(e));
    }
    if (message.service?.length) {
      obj.service = message.service.map((e) => ServiceDescriptorProto.toJSON(e));
    }
    if (message.extension?.length) {
      obj.extension = message.extension.map((e) => FieldDescriptorProto.toJSON(e));
    }
    if (message.options !== undefined) {
      obj.options = FileOptions.toJSON(message.options);
    }
    if (message.sourceCodeInfo !== undefined) {
      obj.sourceCodeInfo = SourceCodeInfo.toJSON(message.sourceCodeInfo);
    }
    if (message.syntax !== "") {
      obj.syntax = message.syntax;
    }
    if (message.edition !== Edition.EDITION_UNKNOWN) {
      obj.edition = editionToJSON(message.edition);
    }
    return obj;
  },

  create(base?: DeepPartial<FileDescriptorProto>): FileDescriptorProto {
    return FileDescriptorProto.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<FileDescriptorProto>): FileDescriptorProto {
    const message = createBaseFileDescriptorProto();
    message.name = object.name ?? "";
    message.package = object.package ?? "";
    message.dependency = object.dependency?.map((e) => e) || [];
    message.publicDependency = object.publicDependency?.map((e) => e) || [];
    message.weakDependency = object.weakDependency?.map((e) => e) || [];
    message.messageType = object.messageType?.map((e) => DescriptorProto.fromPartial(e)) || [];
    message.enumType = object.enumType?.map((e) => EnumDescriptorProto.fromPartial(e)) || [];
    message.service = object.service?.map((e) => ServiceDescriptorProto.fromPartial(e)) || [];
    message.extension = object.extension?.map((e) => FieldDescriptorProto.fromPartial(e)) || [];
    message.options = (object.options !== undefined && object.options !== null)
      ? FileOptions.fromPartial(object.options)
      : undefined;
    message.sourceCodeInfo = (object.sourceCodeInfo !== undefined && object.sourceCodeInfo !== null)
      ? SourceCodeInfo.fromPartial(object.sourceCodeInfo)
      : undefined;
    message.syntax = object.syntax ?? "";
    message.edition = object.edition ?? Edition.EDITION_UNKNOWN;
    return message;
  },
};

function createBaseDescriptorProto(): DescriptorProto {
  return {
    name: "",
    field: [],
    extension: [],
    nestedType: [],
    enumType: [],
    extensionRange: [],
    oneofDecl: [],
    options: undefined,
    reservedRange: [],
    reservedName: [],
  };
}

export const DescriptorProto = {
  encode(message: DescriptorProto, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.field) {
      FieldDescriptorProto.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.extension) {
      FieldDescriptorProto.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    for (const v of message.nestedType) {
      DescriptorProto.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.enumType) {
      EnumDescriptorProto.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.extensionRange) {
      DescriptorProto_ExtensionRange.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.oneofDecl) {
      OneofDescriptorProto.encode(v!, writer.uint32(66).fork()).ldelim();
    }
    if (message.options !== undefined) {
      MessageOptions.encode(message.options, writer.uint32(58).fork()).ldelim();
    }
    for (const v of message.reservedRange) {
      DescriptorProto_ReservedRange.encode(v!, writer.uint32(74).fork()).ldelim();
    }
    for (const v of message.reservedName) {
      writer.uint32(82).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DescriptorProto {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDescriptorProto();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.field.push(FieldDescriptorProto.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.extension.push(FieldDescriptorProto.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.nestedType.push(DescriptorProto.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.enumType.push(EnumDescriptorProto.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.extensionRange.push(DescriptorProto_ExtensionRange.decode(reader, reader.uint32()));
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.oneofDecl.push(OneofDescriptorProto.decode(reader, reader.uint32()));
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.options = MessageOptions.decode(reader, reader.uint32());
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.reservedRange.push(DescriptorProto_ReservedRange.decode(reader, reader.uint32()));
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.reservedName.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DescriptorProto {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      field: globalThis.Array.isArray(object?.field)
        ? object.field.map((e: any) => FieldDescriptorProto.fromJSON(e))
        : [],
      extension: globalThis.Array.isArray(object?.extension)
        ? object.extension.map((e: any) => FieldDescriptorProto.fromJSON(e))
        : [],
      nestedType: globalThis.Array.isArray(object?.nestedType)
        ? object.nestedType.map((e: any) => DescriptorProto.fromJSON(e))
        : [],
      enumType: globalThis.Array.isArray(object?.enumType)
        ? object.enumType.map((e: any) => EnumDescriptorProto.fromJSON(e))
        : [],
      extensionRange: globalThis.Array.isArray(object?.extensionRange)
        ? object.extensionRange.map((e: any) => DescriptorProto_ExtensionRange.fromJSON(e))
        : [],
      oneofDecl: globalThis.Array.isArray(object?.oneofDecl)
        ? object.oneofDecl.map((e: any) => OneofDescriptorProto.fromJSON(e))
        : [],
      options: isSet(object.options) ? MessageOptions.fromJSON(object.options) : undefined,
      reservedRange: globalThis.Array.isArray(object?.reservedRange)
        ? object.reservedRange.map((e: any) => DescriptorProto_ReservedRange.fromJSON(e))
        : [],
      reservedName: globalThis.Array.isArray(object?.reservedName)
        ? object.reservedName.map((e: any) => globalThis.String(e))
        : [],
    };
  },

  toJSON(message: DescriptorProto): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.field?.length) {
      obj.field = message.field.map((e) => FieldDescriptorProto.toJSON(e));
    }
    if (message.extension?.length) {
      obj.extension = message.extension.map((e) => FieldDescriptorProto.toJSON(e));
    }
    if (message.nestedType?.length) {
      obj.nestedType = message.nestedType.map((e) => DescriptorProto.toJSON(e));
    }
    if (message.enumType?.length) {
      obj.enumType = message.enumType.map((e) => EnumDescriptorProto.toJSON(e));
    }
    if (message.extensionRange?.length) {
      obj.extensionRange = message.extensionRange.map((e) => DescriptorProto_ExtensionRange.toJSON(e));
    }
    if (message.oneofDecl?.length) {
      obj.oneofDecl = message.oneofDecl.map((e) => OneofDescriptorProto.toJSON(e));
    }
    if (message.options !== undefined) {
      obj.options = MessageOptions.toJSON(message.options);
    }
    if (message.reservedRange?.length) {
      obj.reservedRange = message.reservedRange.map((e) => DescriptorProto_ReservedRange.toJSON(e));
    }
    if (message.reservedName?.length) {
      obj.reservedName = message.reservedName;
    }
    return obj;
  },

  create(base?: DeepPartial<DescriptorProto>): DescriptorProto {
    return DescriptorProto.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DescriptorProto>): DescriptorProto {
    const message = createBaseDescriptorProto();
    message.name = object.name ?? "";
    message.field = object.field?.map((e) => FieldDescriptorProto.fromPartial(e)) || [];
    message.extension = object.extension?.map((e) => FieldDescriptorProto.fromPartial(e)) || [];
    message.nestedType = object.nestedType?.map((e) => DescriptorProto.fromPartial(e)) || [];
    message.enumType = object.enumType?.map((e) => EnumDescriptorProto.fromPartial(e)) || [];
    message.extensionRange = object.extensionRange?.map((e) => DescriptorProto_ExtensionRange.fromPartial(e)) || [];
    message.oneofDecl = object.oneofDecl?.map((e) => OneofDescriptorProto.fromPartial(e)) || [];
    message.options = (object.options !== undefined && object.options !== null)
      ? MessageOptions.fromPartial(object.options)
      : undefined;
    message.reservedRange = object.reservedRange?.map((e) => DescriptorProto_ReservedRange.fromPartial(e)) || [];
    message.reservedName = object.reservedName?.map((e) => e) || [];
    return message;
  },
};

function createBaseDescriptorProto_ExtensionRange(): DescriptorProto_ExtensionRange {
  return { start: 0, end: 0, options: undefined };
}

export const DescriptorProto_ExtensionRange = {
  encode(message: DescriptorProto_ExtensionRange, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.start !== 0) {
      writer.uint32(8).int32(message.start);
    }
    if (message.end !== 0) {
      writer.uint32(16).int32(message.end);
    }
    if (message.options !== undefined) {
      ExtensionRangeOptions.encode(message.options, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DescriptorProto_ExtensionRange {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDescriptorProto_ExtensionRange();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.start = reader.int32();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.end = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.options = ExtensionRangeOptions.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DescriptorProto_ExtensionRange {
    return {
      start: isSet(object.start) ? globalThis.Number(object.start) : 0,
      end: isSet(object.end) ? globalThis.Number(object.end) : 0,
      options: isSet(object.options) ? ExtensionRangeOptions.fromJSON(object.options) : undefined,
    };
  },

  toJSON(message: DescriptorProto_ExtensionRange): unknown {
    const obj: any = {};
    if (message.start !== 0) {
      obj.start = Math.round(message.start);
    }
    if (message.end !== 0) {
      obj.end = Math.round(message.end);
    }
    if (message.options !== undefined) {
      obj.options = ExtensionRangeOptions.toJSON(message.options);
    }
    return obj;
  },

  create(base?: DeepPartial<DescriptorProto_ExtensionRange>): DescriptorProto_ExtensionRange {
    return DescriptorProto_ExtensionRange.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DescriptorProto_ExtensionRange>): DescriptorProto_ExtensionRange {
    const message = createBaseDescriptorProto_ExtensionRange();
    message.start = object.start ?? 0;
    message.end = object.end ?? 0;
    message.options = (object.options !== undefined && object.options !== null)
      ? ExtensionRangeOptions.fromPartial(object.options)
      : undefined;
    return message;
  },
};

function createBaseDescriptorProto_ReservedRange(): DescriptorProto_ReservedRange {
  return { start: 0, end: 0 };
}

export const DescriptorProto_ReservedRange = {
  encode(message: DescriptorProto_ReservedRange, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.start !== 0) {
      writer.uint32(8).int32(message.start);
    }
    if (message.end !== 0) {
      writer.uint32(16).int32(message.end);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DescriptorProto_ReservedRange {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDescriptorProto_ReservedRange();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.start = reader.int32();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.end = reader.int32();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DescriptorProto_ReservedRange {
    return {
      start: isSet(object.start) ? globalThis.Number(object.start) : 0,
      end: isSet(object.end) ? globalThis.Number(object.end) : 0,
    };
  },

  toJSON(message: DescriptorProto_ReservedRange): unknown {
    const obj: any = {};
    if (message.start !== 0) {
      obj.start = Math.round(message.start);
    }
    if (message.end !== 0) {
      obj.end = Math.round(message.end);
    }
    return obj;
  },

  create(base?: DeepPartial<DescriptorProto_ReservedRange>): DescriptorProto_ReservedRange {
    return DescriptorProto_ReservedRange.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DescriptorProto_ReservedRange>): DescriptorProto_ReservedRange {
    const message = createBaseDescriptorProto_ReservedRange();
    message.start = object.start ?? 0;
    message.end = object.end ?? 0;
    return message;
  },
};

function createBaseExtensionRangeOptions(): ExtensionRangeOptions {
  return {
    uninterpretedOption: [],
    declaration: [],
    features: undefined,
    verification: ExtensionRangeOptions_VerificationState.DECLARATION,
  };
}

export const ExtensionRangeOptions = {
  encode(message: ExtensionRangeOptions, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.uninterpretedOption) {
      UninterpretedOption.encode(v!, writer.uint32(7994).fork()).ldelim();
    }
    for (const v of message.declaration) {
      ExtensionRangeOptions_Declaration.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.features !== undefined) {
      FeatureSet.encode(message.features, writer.uint32(402).fork()).ldelim();
    }
    if (message.verification !== ExtensionRangeOptions_VerificationState.DECLARATION) {
      writer.uint32(24).int32(extensionRangeOptions_VerificationStateToNumber(message.verification));
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExtensionRangeOptions {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExtensionRangeOptions();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 999:
          if (tag !== 7994) {
            break;
          }

          message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.declaration.push(ExtensionRangeOptions_Declaration.decode(reader, reader.uint32()));
          continue;
        case 50:
          if (tag !== 402) {
            break;
          }

          message.features = FeatureSet.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.verification = extensionRangeOptions_VerificationStateFromJSON(reader.int32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ExtensionRangeOptions {
    return {
      uninterpretedOption: globalThis.Array.isArray(object?.uninterpretedOption)
        ? object.uninterpretedOption.map((e: any) => UninterpretedOption.fromJSON(e))
        : [],
      declaration: globalThis.Array.isArray(object?.declaration)
        ? object.declaration.map((e: any) => ExtensionRangeOptions_Declaration.fromJSON(e))
        : [],
      features: isSet(object.features) ? FeatureSet.fromJSON(object.features) : undefined,
      verification: isSet(object.verification)
        ? extensionRangeOptions_VerificationStateFromJSON(object.verification)
        : ExtensionRangeOptions_VerificationState.DECLARATION,
    };
  },

  toJSON(message: ExtensionRangeOptions): unknown {
    const obj: any = {};
    if (message.uninterpretedOption?.length) {
      obj.uninterpretedOption = message.uninterpretedOption.map((e) => UninterpretedOption.toJSON(e));
    }
    if (message.declaration?.length) {
      obj.declaration = message.declaration.map((e) => ExtensionRangeOptions_Declaration.toJSON(e));
    }
    if (message.features !== undefined) {
      obj.features = FeatureSet.toJSON(message.features);
    }
    if (message.verification !== ExtensionRangeOptions_VerificationState.DECLARATION) {
      obj.verification = extensionRangeOptions_VerificationStateToJSON(message.verification);
    }
    return obj;
  },

  create(base?: DeepPartial<ExtensionRangeOptions>): ExtensionRangeOptions {
    return ExtensionRangeOptions.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ExtensionRangeOptions>): ExtensionRangeOptions {
    const message = createBaseExtensionRangeOptions();
    message.uninterpretedOption = object.uninterpretedOption?.map((e) => UninterpretedOption.fromPartial(e)) || [];
    message.declaration = object.declaration?.map((e) => ExtensionRangeOptions_Declaration.fromPartial(e)) || [];
    message.features = (object.features !== undefined && object.features !== null)
      ? FeatureSet.fromPartial(object.features)
      : undefined;
    message.verification = object.verification ?? ExtensionRangeOptions_VerificationState.DECLARATION;
    return message;
  },
};

function createBaseExtensionRangeOptions_Declaration(): ExtensionRangeOptions_Declaration {
  return { number: 0, fullName: "", type: "", reserved: false, repeated: false };
}

export const ExtensionRangeOptions_Declaration = {
  encode(message: ExtensionRangeOptions_Declaration, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.number !== 0) {
      writer.uint32(8).int32(message.number);
    }
    if (message.fullName !== "") {
      writer.uint32(18).string(message.fullName);
    }
    if (message.type !== "") {
      writer.uint32(26).string(message.type);
    }
    if (message.reserved === true) {
      writer.uint32(40).bool(message.reserved);
    }
    if (message.repeated === true) {
      writer.uint32(48).bool(message.repeated);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExtensionRangeOptions_Declaration {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExtensionRangeOptions_Declaration();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.number = reader.int32();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.fullName = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.type = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.reserved = reader.bool();
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.repeated = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ExtensionRangeOptions_Declaration {
    return {
      number: isSet(object.number) ? globalThis.Number(object.number) : 0,
      fullName: isSet(object.fullName) ? globalThis.String(object.fullName) : "",
      type: isSet(object.type) ? globalThis.String(object.type) : "",
      reserved: isSet(object.reserved) ? globalThis.Boolean(object.reserved) : false,
      repeated: isSet(object.repeated) ? globalThis.Boolean(object.repeated) : false,
    };
  },

  toJSON(message: ExtensionRangeOptions_Declaration): unknown {
    const obj: any = {};
    if (message.number !== 0) {
      obj.number = Math.round(message.number);
    }
    if (message.fullName !== "") {
      obj.fullName = message.fullName;
    }
    if (message.type !== "") {
      obj.type = message.type;
    }
    if (message.reserved === true) {
      obj.reserved = message.reserved;
    }
    if (message.repeated === true) {
      obj.repeated = message.repeated;
    }
    return obj;
  },

  create(base?: DeepPartial<ExtensionRangeOptions_Declaration>): ExtensionRangeOptions_Declaration {
    return ExtensionRangeOptions_Declaration.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ExtensionRangeOptions_Declaration>): ExtensionRangeOptions_Declaration {
    const message = createBaseExtensionRangeOptions_Declaration();
    message.number = object.number ?? 0;
    message.fullName = object.fullName ?? "";
    message.type = object.type ?? "";
    message.reserved = object.reserved ?? false;
    message.repeated = object.repeated ?? false;
    return message;
  },
};

function createBaseFieldDescriptorProto(): FieldDescriptorProto {
  return {
    name: "",
    number: 0,
    label: FieldDescriptorProto_Label.LABEL_OPTIONAL,
    type: FieldDescriptorProto_Type.TYPE_DOUBLE,
    typeName: "",
    extendee: "",
    defaultValue: "",
    oneofIndex: 0,
    jsonName: "",
    options: undefined,
    proto3Optional: false,
  };
}

export const FieldDescriptorProto = {
  encode(message: FieldDescriptorProto, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.number !== 0) {
      writer.uint32(24).int32(message.number);
    }
    if (message.label !== FieldDescriptorProto_Label.LABEL_OPTIONAL) {
      writer.uint32(32).int32(fieldDescriptorProto_LabelToNumber(message.label));
    }
    if (message.type !== FieldDescriptorProto_Type.TYPE_DOUBLE) {
      writer.uint32(40).int32(fieldDescriptorProto_TypeToNumber(message.type));
    }
    if (message.typeName !== "") {
      writer.uint32(50).string(message.typeName);
    }
    if (message.extendee !== "") {
      writer.uint32(18).string(message.extendee);
    }
    if (message.defaultValue !== "") {
      writer.uint32(58).string(message.defaultValue);
    }
    if (message.oneofIndex !== 0) {
      writer.uint32(72).int32(message.oneofIndex);
    }
    if (message.jsonName !== "") {
      writer.uint32(82).string(message.jsonName);
    }
    if (message.options !== undefined) {
      FieldOptions.encode(message.options, writer.uint32(66).fork()).ldelim();
    }
    if (message.proto3Optional === true) {
      writer.uint32(136).bool(message.proto3Optional);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FieldDescriptorProto {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFieldDescriptorProto();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.number = reader.int32();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.label = fieldDescriptorProto_LabelFromJSON(reader.int32());
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.type = fieldDescriptorProto_TypeFromJSON(reader.int32());
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.typeName = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.extendee = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.defaultValue = reader.string();
          continue;
        case 9:
          if (tag !== 72) {
            break;
          }

          message.oneofIndex = reader.int32();
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.jsonName = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.options = FieldOptions.decode(reader, reader.uint32());
          continue;
        case 17:
          if (tag !== 136) {
            break;
          }

          message.proto3Optional = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FieldDescriptorProto {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      number: isSet(object.number) ? globalThis.Number(object.number) : 0,
      label: isSet(object.label)
        ? fieldDescriptorProto_LabelFromJSON(object.label)
        : FieldDescriptorProto_Label.LABEL_OPTIONAL,
      type: isSet(object.type) ? fieldDescriptorProto_TypeFromJSON(object.type) : FieldDescriptorProto_Type.TYPE_DOUBLE,
      typeName: isSet(object.typeName) ? globalThis.String(object.typeName) : "",
      extendee: isSet(object.extendee) ? globalThis.String(object.extendee) : "",
      defaultValue: isSet(object.defaultValue) ? globalThis.String(object.defaultValue) : "",
      oneofIndex: isSet(object.oneofIndex) ? globalThis.Number(object.oneofIndex) : 0,
      jsonName: isSet(object.jsonName) ? globalThis.String(object.jsonName) : "",
      options: isSet(object.options) ? FieldOptions.fromJSON(object.options) : undefined,
      proto3Optional: isSet(object.proto3Optional) ? globalThis.Boolean(object.proto3Optional) : false,
    };
  },

  toJSON(message: FieldDescriptorProto): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.number !== 0) {
      obj.number = Math.round(message.number);
    }
    if (message.label !== FieldDescriptorProto_Label.LABEL_OPTIONAL) {
      obj.label = fieldDescriptorProto_LabelToJSON(message.label);
    }
    if (message.type !== FieldDescriptorProto_Type.TYPE_DOUBLE) {
      obj.type = fieldDescriptorProto_TypeToJSON(message.type);
    }
    if (message.typeName !== "") {
      obj.typeName = message.typeName;
    }
    if (message.extendee !== "") {
      obj.extendee = message.extendee;
    }
    if (message.defaultValue !== "") {
      obj.defaultValue = message.defaultValue;
    }
    if (message.oneofIndex !== 0) {
      obj.oneofIndex = Math.round(message.oneofIndex);
    }
    if (message.jsonName !== "") {
      obj.jsonName = message.jsonName;
    }
    if (message.options !== undefined) {
      obj.options = FieldOptions.toJSON(message.options);
    }
    if (message.proto3Optional === true) {
      obj.proto3Optional = message.proto3Optional;
    }
    return obj;
  },

  create(base?: DeepPartial<FieldDescriptorProto>): FieldDescriptorProto {
    return FieldDescriptorProto.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<FieldDescriptorProto>): FieldDescriptorProto {
    const message = createBaseFieldDescriptorProto();
    message.name = object.name ?? "";
    message.number = object.number ?? 0;
    message.label = object.label ?? FieldDescriptorProto_Label.LABEL_OPTIONAL;
    message.type = object.type ?? FieldDescriptorProto_Type.TYPE_DOUBLE;
    message.typeName = object.typeName ?? "";
    message.extendee = object.extendee ?? "";
    message.defaultValue = object.defaultValue ?? "";
    message.oneofIndex = object.oneofIndex ?? 0;
    message.jsonName = object.jsonName ?? "";
    message.options = (object.options !== undefined && object.options !== null)
      ? FieldOptions.fromPartial(object.options)
      : undefined;
    message.proto3Optional = object.proto3Optional ?? false;
    return message;
  },
};

function createBaseOneofDescriptorProto(): OneofDescriptorProto {
  return { name: "", options: undefined };
}

export const OneofDescriptorProto = {
  encode(message: OneofDescriptorProto, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.options !== undefined) {
      OneofOptions.encode(message.options, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): OneofDescriptorProto {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseOneofDescriptorProto();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.options = OneofOptions.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): OneofDescriptorProto {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      options: isSet(object.options) ? OneofOptions.fromJSON(object.options) : undefined,
    };
  },

  toJSON(message: OneofDescriptorProto): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.options !== undefined) {
      obj.options = OneofOptions.toJSON(message.options);
    }
    return obj;
  },

  create(base?: DeepPartial<OneofDescriptorProto>): OneofDescriptorProto {
    return OneofDescriptorProto.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<OneofDescriptorProto>): OneofDescriptorProto {
    const message = createBaseOneofDescriptorProto();
    message.name = object.name ?? "";
    message.options = (object.options !== undefined && object.options !== null)
      ? OneofOptions.fromPartial(object.options)
      : undefined;
    return message;
  },
};

function createBaseEnumDescriptorProto(): EnumDescriptorProto {
  return { name: "", value: [], options: undefined, reservedRange: [], reservedName: [] };
}

export const EnumDescriptorProto = {
  encode(message: EnumDescriptorProto, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.value) {
      EnumValueDescriptorProto.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.options !== undefined) {
      EnumOptions.encode(message.options, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.reservedRange) {
      EnumDescriptorProto_EnumReservedRange.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.reservedName) {
      writer.uint32(42).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): EnumDescriptorProto {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEnumDescriptorProto();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value.push(EnumValueDescriptorProto.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.options = EnumOptions.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.reservedRange.push(EnumDescriptorProto_EnumReservedRange.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.reservedName.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): EnumDescriptorProto {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      value: globalThis.Array.isArray(object?.value)
        ? object.value.map((e: any) => EnumValueDescriptorProto.fromJSON(e))
        : [],
      options: isSet(object.options) ? EnumOptions.fromJSON(object.options) : undefined,
      reservedRange: globalThis.Array.isArray(object?.reservedRange)
        ? object.reservedRange.map((e: any) => EnumDescriptorProto_EnumReservedRange.fromJSON(e))
        : [],
      reservedName: globalThis.Array.isArray(object?.reservedName)
        ? object.reservedName.map((e: any) => globalThis.String(e))
        : [],
    };
  },

  toJSON(message: EnumDescriptorProto): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.value?.length) {
      obj.value = message.value.map((e) => EnumValueDescriptorProto.toJSON(e));
    }
    if (message.options !== undefined) {
      obj.options = EnumOptions.toJSON(message.options);
    }
    if (message.reservedRange?.length) {
      obj.reservedRange = message.reservedRange.map((e) => EnumDescriptorProto_EnumReservedRange.toJSON(e));
    }
    if (message.reservedName?.length) {
      obj.reservedName = message.reservedName;
    }
    return obj;
  },

  create(base?: DeepPartial<EnumDescriptorProto>): EnumDescriptorProto {
    return EnumDescriptorProto.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<EnumDescriptorProto>): EnumDescriptorProto {
    const message = createBaseEnumDescriptorProto();
    message.name = object.name ?? "";
    message.value = object.value?.map((e) => EnumValueDescriptorProto.fromPartial(e)) || [];
    message.options = (object.options !== undefined && object.options !== null)
      ? EnumOptions.fromPartial(object.options)
      : undefined;
    message.reservedRange = object.reservedRange?.map((e) => EnumDescriptorProto_EnumReservedRange.fromPartial(e)) ||
      [];
    message.reservedName = object.reservedName?.map((e) => e) || [];
    return message;
  },
};

function createBaseEnumDescriptorProto_EnumReservedRange(): EnumDescriptorProto_EnumReservedRange {
  return { start: 0, end: 0 };
}

export const EnumDescriptorProto_EnumReservedRange = {
  encode(message: EnumDescriptorProto_EnumReservedRange, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.start !== 0) {
      writer.uint32(8).int32(message.start);
    }
    if (message.end !== 0) {
      writer.uint32(16).int32(message.end);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): EnumDescriptorProto_EnumReservedRange {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEnumDescriptorProto_EnumReservedRange();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.start = reader.int32();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.end = reader.int32();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): EnumDescriptorProto_EnumReservedRange {
    return {
      start: isSet(object.start) ? globalThis.Number(object.start) : 0,
      end: isSet(object.end) ? globalThis.Number(object.end) : 0,
    };
  },

  toJSON(message: EnumDescriptorProto_EnumReservedRange): unknown {
    const obj: any = {};
    if (message.start !== 0) {
      obj.start = Math.round(message.start);
    }
    if (message.end !== 0) {
      obj.end = Math.round(message.end);
    }
    return obj;
  },

  create(base?: DeepPartial<EnumDescriptorProto_EnumReservedRange>): EnumDescriptorProto_EnumReservedRange {
    return EnumDescriptorProto_EnumReservedRange.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<EnumDescriptorProto_EnumReservedRange>): EnumDescriptorProto_EnumReservedRange {
    const message = createBaseEnumDescriptorProto_EnumReservedRange();
    message.start = object.start ?? 0;
    message.end = object.end ?? 0;
    return message;
  },
};

function createBaseEnumValueDescriptorProto(): EnumValueDescriptorProto {
  return { name: "", number: 0, options: undefined };
}

export const EnumValueDescriptorProto = {
  encode(message: EnumValueDescriptorProto, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.number !== 0) {
      writer.uint32(16).int32(message.number);
    }
    if (message.options !== undefined) {
      EnumValueOptions.encode(message.options, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): EnumValueDescriptorProto {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEnumValueDescriptorProto();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.number = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.options = EnumValueOptions.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): EnumValueDescriptorProto {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      number: isSet(object.number) ? globalThis.Number(object.number) : 0,
      options: isSet(object.options) ? EnumValueOptions.fromJSON(object.options) : undefined,
    };
  },

  toJSON(message: EnumValueDescriptorProto): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.number !== 0) {
      obj.number = Math.round(message.number);
    }
    if (message.options !== undefined) {
      obj.options = EnumValueOptions.toJSON(message.options);
    }
    return obj;
  },

  create(base?: DeepPartial<EnumValueDescriptorProto>): EnumValueDescriptorProto {
    return EnumValueDescriptorProto.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<EnumValueDescriptorProto>): EnumValueDescriptorProto {
    const message = createBaseEnumValueDescriptorProto();
    message.name = object.name ?? "";
    message.number = object.number ?? 0;
    message.options = (object.options !== undefined && object.options !== null)
      ? EnumValueOptions.fromPartial(object.options)
      : undefined;
    return message;
  },
};

function createBaseServiceDescriptorProto(): ServiceDescriptorProto {
  return { name: "", method: [], options: undefined };
}

export const ServiceDescriptorProto = {
  encode(message: ServiceDescriptorProto, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.method) {
      MethodDescriptorProto.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.options !== undefined) {
      ServiceOptions.encode(message.options, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ServiceDescriptorProto {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseServiceDescriptorProto();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.method.push(MethodDescriptorProto.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.options = ServiceOptions.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ServiceDescriptorProto {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      method: globalThis.Array.isArray(object?.method)
        ? object.method.map((e: any) => MethodDescriptorProto.fromJSON(e))
        : [],
      options: isSet(object.options) ? ServiceOptions.fromJSON(object.options) : undefined,
    };
  },

  toJSON(message: ServiceDescriptorProto): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.method?.length) {
      obj.method = message.method.map((e) => MethodDescriptorProto.toJSON(e));
    }
    if (message.options !== undefined) {
      obj.options = ServiceOptions.toJSON(message.options);
    }
    return obj;
  },

  create(base?: DeepPartial<ServiceDescriptorProto>): ServiceDescriptorProto {
    return ServiceDescriptorProto.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ServiceDescriptorProto>): ServiceDescriptorProto {
    const message = createBaseServiceDescriptorProto();
    message.name = object.name ?? "";
    message.method = object.method?.map((e) => MethodDescriptorProto.fromPartial(e)) || [];
    message.options = (object.options !== undefined && object.options !== null)
      ? ServiceOptions.fromPartial(object.options)
      : undefined;
    return message;
  },
};

function createBaseMethodDescriptorProto(): MethodDescriptorProto {
  return {
    name: "",
    inputType: "",
    outputType: "",
    options: undefined,
    clientStreaming: false,
    serverStreaming: false,
  };
}

export const MethodDescriptorProto = {
  encode(message: MethodDescriptorProto, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.inputType !== "") {
      writer.uint32(18).string(message.inputType);
    }
    if (message.outputType !== "") {
      writer.uint32(26).string(message.outputType);
    }
    if (message.options !== undefined) {
      MethodOptions.encode(message.options, writer.uint32(34).fork()).ldelim();
    }
    if (message.clientStreaming === true) {
      writer.uint32(40).bool(message.clientStreaming);
    }
    if (message.serverStreaming === true) {
      writer.uint32(48).bool(message.serverStreaming);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MethodDescriptorProto {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMethodDescriptorProto();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.inputType = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.outputType = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.options = MethodOptions.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.clientStreaming = reader.bool();
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.serverStreaming = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MethodDescriptorProto {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      inputType: isSet(object.inputType) ? globalThis.String(object.inputType) : "",
      outputType: isSet(object.outputType) ? globalThis.String(object.outputType) : "",
      options: isSet(object.options) ? MethodOptions.fromJSON(object.options) : undefined,
      clientStreaming: isSet(object.clientStreaming) ? globalThis.Boolean(object.clientStreaming) : false,
      serverStreaming: isSet(object.serverStreaming) ? globalThis.Boolean(object.serverStreaming) : false,
    };
  },

  toJSON(message: MethodDescriptorProto): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.inputType !== "") {
      obj.inputType = message.inputType;
    }
    if (message.outputType !== "") {
      obj.outputType = message.outputType;
    }
    if (message.options !== undefined) {
      obj.options = MethodOptions.toJSON(message.options);
    }
    if (message.clientStreaming === true) {
      obj.clientStreaming = message.clientStreaming;
    }
    if (message.serverStreaming === true) {
      obj.serverStreaming = message.serverStreaming;
    }
    return obj;
  },

  create(base?: DeepPartial<MethodDescriptorProto>): MethodDescriptorProto {
    return MethodDescriptorProto.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<MethodDescriptorProto>): MethodDescriptorProto {
    const message = createBaseMethodDescriptorProto();
    message.name = object.name ?? "";
    message.inputType = object.inputType ?? "";
    message.outputType = object.outputType ?? "";
    message.options = (object.options !== undefined && object.options !== null)
      ? MethodOptions.fromPartial(object.options)
      : undefined;
    message.clientStreaming = object.clientStreaming ?? false;
    message.serverStreaming = object.serverStreaming ?? false;
    return message;
  },
};

function createBaseFileOptions(): FileOptions {
  return {
    javaPackage: "",
    javaOuterClassname: "",
    javaMultipleFiles: false,
    javaGenerateEqualsAndHash: false,
    javaStringCheckUtf8: false,
    optimizeFor: FileOptions_OptimizeMode.SPEED,
    goPackage: "",
    ccGenericServices: false,
    javaGenericServices: false,
    pyGenericServices: false,
    deprecated: false,
    ccEnableArenas: false,
    objcClassPrefix: "",
    csharpNamespace: "",
    swiftPrefix: "",
    phpClassPrefix: "",
    phpNamespace: "",
    phpMetadataNamespace: "",
    rubyPackage: "",
    features: undefined,
    uninterpretedOption: [],
  };
}

export const FileOptions = {
  encode(message: FileOptions, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.javaPackage !== "") {
      writer.uint32(10).string(message.javaPackage);
    }
    if (message.javaOuterClassname !== "") {
      writer.uint32(66).string(message.javaOuterClassname);
    }
    if (message.javaMultipleFiles === true) {
      writer.uint32(80).bool(message.javaMultipleFiles);
    }
    if (message.javaGenerateEqualsAndHash === true) {
      writer.uint32(160).bool(message.javaGenerateEqualsAndHash);
    }
    if (message.javaStringCheckUtf8 === true) {
      writer.uint32(216).bool(message.javaStringCheckUtf8);
    }
    if (message.optimizeFor !== FileOptions_OptimizeMode.SPEED) {
      writer.uint32(72).int32(fileOptions_OptimizeModeToNumber(message.optimizeFor));
    }
    if (message.goPackage !== "") {
      writer.uint32(90).string(message.goPackage);
    }
    if (message.ccGenericServices === true) {
      writer.uint32(128).bool(message.ccGenericServices);
    }
    if (message.javaGenericServices === true) {
      writer.uint32(136).bool(message.javaGenericServices);
    }
    if (message.pyGenericServices === true) {
      writer.uint32(144).bool(message.pyGenericServices);
    }
    if (message.deprecated === true) {
      writer.uint32(184).bool(message.deprecated);
    }
    if (message.ccEnableArenas === true) {
      writer.uint32(248).bool(message.ccEnableArenas);
    }
    if (message.objcClassPrefix !== "") {
      writer.uint32(290).string(message.objcClassPrefix);
    }
    if (message.csharpNamespace !== "") {
      writer.uint32(298).string(message.csharpNamespace);
    }
    if (message.swiftPrefix !== "") {
      writer.uint32(314).string(message.swiftPrefix);
    }
    if (message.phpClassPrefix !== "") {
      writer.uint32(322).string(message.phpClassPrefix);
    }
    if (message.phpNamespace !== "") {
      writer.uint32(330).string(message.phpNamespace);
    }
    if (message.phpMetadataNamespace !== "") {
      writer.uint32(354).string(message.phpMetadataNamespace);
    }
    if (message.rubyPackage !== "") {
      writer.uint32(362).string(message.rubyPackage);
    }
    if (message.features !== undefined) {
      FeatureSet.encode(message.features, writer.uint32(402).fork()).ldelim();
    }
    for (const v of message.uninterpretedOption) {
      UninterpretedOption.encode(v!, writer.uint32(7994).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FileOptions {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFileOptions();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.javaPackage = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.javaOuterClassname = reader.string();
          continue;
        case 10:
          if (tag !== 80) {
            break;
          }

          message.javaMultipleFiles = reader.bool();
          continue;
        case 20:
          if (tag !== 160) {
            break;
          }

          message.javaGenerateEqualsAndHash = reader.bool();
          continue;
        case 27:
          if (tag !== 216) {
            break;
          }

          message.javaStringCheckUtf8 = reader.bool();
          continue;
        case 9:
          if (tag !== 72) {
            break;
          }

          message.optimizeFor = fileOptions_OptimizeModeFromJSON(reader.int32());
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.goPackage = reader.string();
          continue;
        case 16:
          if (tag !== 128) {
            break;
          }

          message.ccGenericServices = reader.bool();
          continue;
        case 17:
          if (tag !== 136) {
            break;
          }

          message.javaGenericServices = reader.bool();
          continue;
        case 18:
          if (tag !== 144) {
            break;
          }

          message.pyGenericServices = reader.bool();
          continue;
        case 23:
          if (tag !== 184) {
            break;
          }

          message.deprecated = reader.bool();
          continue;
        case 31:
          if (tag !== 248) {
            break;
          }

          message.ccEnableArenas = reader.bool();
          continue;
        case 36:
          if (tag !== 290) {
            break;
          }

          message.objcClassPrefix = reader.string();
          continue;
        case 37:
          if (tag !== 298) {
            break;
          }

          message.csharpNamespace = reader.string();
          continue;
        case 39:
          if (tag !== 314) {
            break;
          }

          message.swiftPrefix = reader.string();
          continue;
        case 40:
          if (tag !== 322) {
            break;
          }

          message.phpClassPrefix = reader.string();
          continue;
        case 41:
          if (tag !== 330) {
            break;
          }

          message.phpNamespace = reader.string();
          continue;
        case 44:
          if (tag !== 354) {
            break;
          }

          message.phpMetadataNamespace = reader.string();
          continue;
        case 45:
          if (tag !== 362) {
            break;
          }

          message.rubyPackage = reader.string();
          continue;
        case 50:
          if (tag !== 402) {
            break;
          }

          message.features = FeatureSet.decode(reader, reader.uint32());
          continue;
        case 999:
          if (tag !== 7994) {
            break;
          }

          message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FileOptions {
    return {
      javaPackage: isSet(object.javaPackage) ? globalThis.String(object.javaPackage) : "",
      javaOuterClassname: isSet(object.javaOuterClassname) ? globalThis.String(object.javaOuterClassname) : "",
      javaMultipleFiles: isSet(object.javaMultipleFiles) ? globalThis.Boolean(object.javaMultipleFiles) : false,
      javaGenerateEqualsAndHash: isSet(object.javaGenerateEqualsAndHash)
        ? globalThis.Boolean(object.javaGenerateEqualsAndHash)
        : false,
      javaStringCheckUtf8: isSet(object.javaStringCheckUtf8) ? globalThis.Boolean(object.javaStringCheckUtf8) : false,
      optimizeFor: isSet(object.optimizeFor)
        ? fileOptions_OptimizeModeFromJSON(object.optimizeFor)
        : FileOptions_OptimizeMode.SPEED,
      goPackage: isSet(object.goPackage) ? globalThis.String(object.goPackage) : "",
      ccGenericServices: isSet(object.ccGenericServices) ? globalThis.Boolean(object.ccGenericServices) : false,
      javaGenericServices: isSet(object.javaGenericServices) ? globalThis.Boolean(object.javaGenericServices) : false,
      pyGenericServices: isSet(object.pyGenericServices) ? globalThis.Boolean(object.pyGenericServices) : false,
      deprecated: isSet(object.deprecated) ? globalThis.Boolean(object.deprecated) : false,
      ccEnableArenas: isSet(object.ccEnableArenas) ? globalThis.Boolean(object.ccEnableArenas) : false,
      objcClassPrefix: isSet(object.objcClassPrefix) ? globalThis.String(object.objcClassPrefix) : "",
      csharpNamespace: isSet(object.csharpNamespace) ? globalThis.String(object.csharpNamespace) : "",
      swiftPrefix: isSet(object.swiftPrefix) ? globalThis.String(object.swiftPrefix) : "",
      phpClassPrefix: isSet(object.phpClassPrefix) ? globalThis.String(object.phpClassPrefix) : "",
      phpNamespace: isSet(object.phpNamespace) ? globalThis.String(object.phpNamespace) : "",
      phpMetadataNamespace: isSet(object.phpMetadataNamespace) ? globalThis.String(object.phpMetadataNamespace) : "",
      rubyPackage: isSet(object.rubyPackage) ? globalThis.String(object.rubyPackage) : "",
      features: isSet(object.features) ? FeatureSet.fromJSON(object.features) : undefined,
      uninterpretedOption: globalThis.Array.isArray(object?.uninterpretedOption)
        ? object.uninterpretedOption.map((e: any) => UninterpretedOption.fromJSON(e))
        : [],
    };
  },

  toJSON(message: FileOptions): unknown {
    const obj: any = {};
    if (message.javaPackage !== "") {
      obj.javaPackage = message.javaPackage;
    }
    if (message.javaOuterClassname !== "") {
      obj.javaOuterClassname = message.javaOuterClassname;
    }
    if (message.javaMultipleFiles === true) {
      obj.javaMultipleFiles = message.javaMultipleFiles;
    }
    if (message.javaGenerateEqualsAndHash === true) {
      obj.javaGenerateEqualsAndHash = message.javaGenerateEqualsAndHash;
    }
    if (message.javaStringCheckUtf8 === true) {
      obj.javaStringCheckUtf8 = message.javaStringCheckUtf8;
    }
    if (message.optimizeFor !== FileOptions_OptimizeMode.SPEED) {
      obj.optimizeFor = fileOptions_OptimizeModeToJSON(message.optimizeFor);
    }
    if (message.goPackage !== "") {
      obj.goPackage = message.goPackage;
    }
    if (message.ccGenericServices === true) {
      obj.ccGenericServices = message.ccGenericServices;
    }
    if (message.javaGenericServices === true) {
      obj.javaGenericServices = message.javaGenericServices;
    }
    if (message.pyGenericServices === true) {
      obj.pyGenericServices = message.pyGenericServices;
    }
    if (message.deprecated === true) {
      obj.deprecated = message.deprecated;
    }
    if (message.ccEnableArenas === true) {
      obj.ccEnableArenas = message.ccEnableArenas;
    }
    if (message.objcClassPrefix !== "") {
      obj.objcClassPrefix = message.objcClassPrefix;
    }
    if (message.csharpNamespace !== "") {
      obj.csharpNamespace = message.csharpNamespace;
    }
    if (message.swiftPrefix !== "") {
      obj.swiftPrefix = message.swiftPrefix;
    }
    if (message.phpClassPrefix !== "") {
      obj.phpClassPrefix = message.phpClassPrefix;
    }
    if (message.phpNamespace !== "") {
      obj.phpNamespace = message.phpNamespace;
    }
    if (message.phpMetadataNamespace !== "") {
      obj.phpMetadataNamespace = message.phpMetadataNamespace;
    }
    if (message.rubyPackage !== "") {
      obj.rubyPackage = message.rubyPackage;
    }
    if (message.features !== undefined) {
      obj.features = FeatureSet.toJSON(message.features);
    }
    if (message.uninterpretedOption?.length) {
      obj.uninterpretedOption = message.uninterpretedOption.map((e) => UninterpretedOption.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<FileOptions>): FileOptions {
    return FileOptions.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<FileOptions>): FileOptions {
    const message = createBaseFileOptions();
    message.javaPackage = object.javaPackage ?? "";
    message.javaOuterClassname = object.javaOuterClassname ?? "";
    message.javaMultipleFiles = object.javaMultipleFiles ?? false;
    message.javaGenerateEqualsAndHash = object.javaGenerateEqualsAndHash ?? false;
    message.javaStringCheckUtf8 = object.javaStringCheckUtf8 ?? false;
    message.optimizeFor = object.optimizeFor ?? FileOptions_OptimizeMode.SPEED;
    message.goPackage = object.goPackage ?? "";
    message.ccGenericServices = object.ccGenericServices ?? false;
    message.javaGenericServices = object.javaGenericServices ?? false;
    message.pyGenericServices = object.pyGenericServices ?? false;
    message.deprecated = object.deprecated ?? false;
    message.ccEnableArenas = object.ccEnableArenas ?? false;
    message.objcClassPrefix = object.objcClassPrefix ?? "";
    message.csharpNamespace = object.csharpNamespace ?? "";
    message.swiftPrefix = object.swiftPrefix ?? "";
    message.phpClassPrefix = object.phpClassPrefix ?? "";
    message.phpNamespace = object.phpNamespace ?? "";
    message.phpMetadataNamespace = object.phpMetadataNamespace ?? "";
    message.rubyPackage = object.rubyPackage ?? "";
    message.features = (object.features !== undefined && object.features !== null)
      ? FeatureSet.fromPartial(object.features)
      : undefined;
    message.uninterpretedOption = object.uninterpretedOption?.map((e) => UninterpretedOption.fromPartial(e)) || [];
    return message;
  },
};

function createBaseMessageOptions(): MessageOptions {
  return {
    messageSetWireFormat: false,
    noStandardDescriptorAccessor: false,
    deprecated: false,
    mapEntry: false,
    deprecatedLegacyJsonFieldConflicts: false,
    features: undefined,
    uninterpretedOption: [],
  };
}

export const MessageOptions = {
  encode(message: MessageOptions, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.messageSetWireFormat === true) {
      writer.uint32(8).bool(message.messageSetWireFormat);
    }
    if (message.noStandardDescriptorAccessor === true) {
      writer.uint32(16).bool(message.noStandardDescriptorAccessor);
    }
    if (message.deprecated === true) {
      writer.uint32(24).bool(message.deprecated);
    }
    if (message.mapEntry === true) {
      writer.uint32(56).bool(message.mapEntry);
    }
    if (message.deprecatedLegacyJsonFieldConflicts === true) {
      writer.uint32(88).bool(message.deprecatedLegacyJsonFieldConflicts);
    }
    if (message.features !== undefined) {
      FeatureSet.encode(message.features, writer.uint32(98).fork()).ldelim();
    }
    for (const v of message.uninterpretedOption) {
      UninterpretedOption.encode(v!, writer.uint32(7994).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MessageOptions {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMessageOptions();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.messageSetWireFormat = reader.bool();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.noStandardDescriptorAccessor = reader.bool();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.deprecated = reader.bool();
          continue;
        case 7:
          if (tag !== 56) {
            break;
          }

          message.mapEntry = reader.bool();
          continue;
        case 11:
          if (tag !== 88) {
            break;
          }

          message.deprecatedLegacyJsonFieldConflicts = reader.bool();
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.features = FeatureSet.decode(reader, reader.uint32());
          continue;
        case 999:
          if (tag !== 7994) {
            break;
          }

          message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MessageOptions {
    return {
      messageSetWireFormat: isSet(object.messageSetWireFormat)
        ? globalThis.Boolean(object.messageSetWireFormat)
        : false,
      noStandardDescriptorAccessor: isSet(object.noStandardDescriptorAccessor)
        ? globalThis.Boolean(object.noStandardDescriptorAccessor)
        : false,
      deprecated: isSet(object.deprecated) ? globalThis.Boolean(object.deprecated) : false,
      mapEntry: isSet(object.mapEntry) ? globalThis.Boolean(object.mapEntry) : false,
      deprecatedLegacyJsonFieldConflicts: isSet(object.deprecatedLegacyJsonFieldConflicts)
        ? globalThis.Boolean(object.deprecatedLegacyJsonFieldConflicts)
        : false,
      features: isSet(object.features) ? FeatureSet.fromJSON(object.features) : undefined,
      uninterpretedOption: globalThis.Array.isArray(object?.uninterpretedOption)
        ? object.uninterpretedOption.map((e: any) => UninterpretedOption.fromJSON(e))
        : [],
    };
  },

  toJSON(message: MessageOptions): unknown {
    const obj: any = {};
    if (message.messageSetWireFormat === true) {
      obj.messageSetWireFormat = message.messageSetWireFormat;
    }
    if (message.noStandardDescriptorAccessor === true) {
      obj.noStandardDescriptorAccessor = message.noStandardDescriptorAccessor;
    }
    if (message.deprecated === true) {
      obj.deprecated = message.deprecated;
    }
    if (message.mapEntry === true) {
      obj.mapEntry = message.mapEntry;
    }
    if (message.deprecatedLegacyJsonFieldConflicts === true) {
      obj.deprecatedLegacyJsonFieldConflicts = message.deprecatedLegacyJsonFieldConflicts;
    }
    if (message.features !== undefined) {
      obj.features = FeatureSet.toJSON(message.features);
    }
    if (message.uninterpretedOption?.length) {
      obj.uninterpretedOption = message.uninterpretedOption.map((e) => UninterpretedOption.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<MessageOptions>): MessageOptions {
    return MessageOptions.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<MessageOptions>): MessageOptions {
    const message = createBaseMessageOptions();
    message.messageSetWireFormat = object.messageSetWireFormat ?? false;
    message.noStandardDescriptorAccessor = object.noStandardDescriptorAccessor ?? false;
    message.deprecated = object.deprecated ?? false;
    message.mapEntry = object.mapEntry ?? false;
    message.deprecatedLegacyJsonFieldConflicts = object.deprecatedLegacyJsonFieldConflicts ?? false;
    message.features = (object.features !== undefined && object.features !== null)
      ? FeatureSet.fromPartial(object.features)
      : undefined;
    message.uninterpretedOption = object.uninterpretedOption?.map((e) => UninterpretedOption.fromPartial(e)) || [];
    return message;
  },
};

function createBaseFieldOptions(): FieldOptions {
  return {
    ctype: FieldOptions_CType.STRING,
    packed: false,
    jstype: FieldOptions_JSType.JS_NORMAL,
    lazy: false,
    unverifiedLazy: false,
    deprecated: false,
    weak: false,
    debugRedact: false,
    retention: FieldOptions_OptionRetention.RETENTION_UNKNOWN,
    targets: [],
    editionDefaults: [],
    features: undefined,
    featureSupport: undefined,
    uninterpretedOption: [],
  };
}

export const FieldOptions = {
  encode(message: FieldOptions, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.ctype !== FieldOptions_CType.STRING) {
      writer.uint32(8).int32(fieldOptions_CTypeToNumber(message.ctype));
    }
    if (message.packed === true) {
      writer.uint32(16).bool(message.packed);
    }
    if (message.jstype !== FieldOptions_JSType.JS_NORMAL) {
      writer.uint32(48).int32(fieldOptions_JSTypeToNumber(message.jstype));
    }
    if (message.lazy === true) {
      writer.uint32(40).bool(message.lazy);
    }
    if (message.unverifiedLazy === true) {
      writer.uint32(120).bool(message.unverifiedLazy);
    }
    if (message.deprecated === true) {
      writer.uint32(24).bool(message.deprecated);
    }
    if (message.weak === true) {
      writer.uint32(80).bool(message.weak);
    }
    if (message.debugRedact === true) {
      writer.uint32(128).bool(message.debugRedact);
    }
    if (message.retention !== FieldOptions_OptionRetention.RETENTION_UNKNOWN) {
      writer.uint32(136).int32(fieldOptions_OptionRetentionToNumber(message.retention));
    }
    writer.uint32(154).fork();
    for (const v of message.targets) {
      writer.int32(fieldOptions_OptionTargetTypeToNumber(v));
    }
    writer.ldelim();
    for (const v of message.editionDefaults) {
      FieldOptions_EditionDefault.encode(v!, writer.uint32(162).fork()).ldelim();
    }
    if (message.features !== undefined) {
      FeatureSet.encode(message.features, writer.uint32(170).fork()).ldelim();
    }
    if (message.featureSupport !== undefined) {
      FieldOptions_FeatureSupport.encode(message.featureSupport, writer.uint32(178).fork()).ldelim();
    }
    for (const v of message.uninterpretedOption) {
      UninterpretedOption.encode(v!, writer.uint32(7994).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FieldOptions {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFieldOptions();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.ctype = fieldOptions_CTypeFromJSON(reader.int32());
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.packed = reader.bool();
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.jstype = fieldOptions_JSTypeFromJSON(reader.int32());
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.lazy = reader.bool();
          continue;
        case 15:
          if (tag !== 120) {
            break;
          }

          message.unverifiedLazy = reader.bool();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.deprecated = reader.bool();
          continue;
        case 10:
          if (tag !== 80) {
            break;
          }

          message.weak = reader.bool();
          continue;
        case 16:
          if (tag !== 128) {
            break;
          }

          message.debugRedact = reader.bool();
          continue;
        case 17:
          if (tag !== 136) {
            break;
          }

          message.retention = fieldOptions_OptionRetentionFromJSON(reader.int32());
          continue;
        case 19:
          if (tag === 152) {
            message.targets.push(fieldOptions_OptionTargetTypeFromJSON(reader.int32()));

            continue;
          }

          if (tag === 154) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.targets.push(fieldOptions_OptionTargetTypeFromJSON(reader.int32()));
            }

            continue;
          }

          break;
        case 20:
          if (tag !== 162) {
            break;
          }

          message.editionDefaults.push(FieldOptions_EditionDefault.decode(reader, reader.uint32()));
          continue;
        case 21:
          if (tag !== 170) {
            break;
          }

          message.features = FeatureSet.decode(reader, reader.uint32());
          continue;
        case 22:
          if (tag !== 178) {
            break;
          }

          message.featureSupport = FieldOptions_FeatureSupport.decode(reader, reader.uint32());
          continue;
        case 999:
          if (tag !== 7994) {
            break;
          }

          message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FieldOptions {
    return {
      ctype: isSet(object.ctype) ? fieldOptions_CTypeFromJSON(object.ctype) : FieldOptions_CType.STRING,
      packed: isSet(object.packed) ? globalThis.Boolean(object.packed) : false,
      jstype: isSet(object.jstype) ? fieldOptions_JSTypeFromJSON(object.jstype) : FieldOptions_JSType.JS_NORMAL,
      lazy: isSet(object.lazy) ? globalThis.Boolean(object.lazy) : false,
      unverifiedLazy: isSet(object.unverifiedLazy) ? globalThis.Boolean(object.unverifiedLazy) : false,
      deprecated: isSet(object.deprecated) ? globalThis.Boolean(object.deprecated) : false,
      weak: isSet(object.weak) ? globalThis.Boolean(object.weak) : false,
      debugRedact: isSet(object.debugRedact) ? globalThis.Boolean(object.debugRedact) : false,
      retention: isSet(object.retention)
        ? fieldOptions_OptionRetentionFromJSON(object.retention)
        : FieldOptions_OptionRetention.RETENTION_UNKNOWN,
      targets: globalThis.Array.isArray(object?.targets)
        ? object.targets.map((e: any) => fieldOptions_OptionTargetTypeFromJSON(e))
        : [],
      editionDefaults: globalThis.Array.isArray(object?.editionDefaults)
        ? object.editionDefaults.map((e: any) => FieldOptions_EditionDefault.fromJSON(e))
        : [],
      features: isSet(object.features) ? FeatureSet.fromJSON(object.features) : undefined,
      featureSupport: isSet(object.featureSupport)
        ? FieldOptions_FeatureSupport.fromJSON(object.featureSupport)
        : undefined,
      uninterpretedOption: globalThis.Array.isArray(object?.uninterpretedOption)
        ? object.uninterpretedOption.map((e: any) => UninterpretedOption.fromJSON(e))
        : [],
    };
  },

  toJSON(message: FieldOptions): unknown {
    const obj: any = {};
    if (message.ctype !== FieldOptions_CType.STRING) {
      obj.ctype = fieldOptions_CTypeToJSON(message.ctype);
    }
    if (message.packed === true) {
      obj.packed = message.packed;
    }
    if (message.jstype !== FieldOptions_JSType.JS_NORMAL) {
      obj.jstype = fieldOptions_JSTypeToJSON(message.jstype);
    }
    if (message.lazy === true) {
      obj.lazy = message.lazy;
    }
    if (message.unverifiedLazy === true) {
      obj.unverifiedLazy = message.unverifiedLazy;
    }
    if (message.deprecated === true) {
      obj.deprecated = message.deprecated;
    }
    if (message.weak === true) {
      obj.weak = message.weak;
    }
    if (message.debugRedact === true) {
      obj.debugRedact = message.debugRedact;
    }
    if (message.retention !== FieldOptions_OptionRetention.RETENTION_UNKNOWN) {
      obj.retention = fieldOptions_OptionRetentionToJSON(message.retention);
    }
    if (message.targets?.length) {
      obj.targets = message.targets.map((e) => fieldOptions_OptionTargetTypeToJSON(e));
    }
    if (message.editionDefaults?.length) {
      obj.editionDefaults = message.editionDefaults.map((e) => FieldOptions_EditionDefault.toJSON(e));
    }
    if (message.features !== undefined) {
      obj.features = FeatureSet.toJSON(message.features);
    }
    if (message.featureSupport !== undefined) {
      obj.featureSupport = FieldOptions_FeatureSupport.toJSON(message.featureSupport);
    }
    if (message.uninterpretedOption?.length) {
      obj.uninterpretedOption = message.uninterpretedOption.map((e) => UninterpretedOption.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<FieldOptions>): FieldOptions {
    return FieldOptions.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<FieldOptions>): FieldOptions {
    const message = createBaseFieldOptions();
    message.ctype = object.ctype ?? FieldOptions_CType.STRING;
    message.packed = object.packed ?? false;
    message.jstype = object.jstype ?? FieldOptions_JSType.JS_NORMAL;
    message.lazy = object.lazy ?? false;
    message.unverifiedLazy = object.unverifiedLazy ?? false;
    message.deprecated = object.deprecated ?? false;
    message.weak = object.weak ?? false;
    message.debugRedact = object.debugRedact ?? false;
    message.retention = object.retention ?? FieldOptions_OptionRetention.RETENTION_UNKNOWN;
    message.targets = object.targets?.map((e) => e) || [];
    message.editionDefaults = object.editionDefaults?.map((e) => FieldOptions_EditionDefault.fromPartial(e)) || [];
    message.features = (object.features !== undefined && object.features !== null)
      ? FeatureSet.fromPartial(object.features)
      : undefined;
    message.featureSupport = (object.featureSupport !== undefined && object.featureSupport !== null)
      ? FieldOptions_FeatureSupport.fromPartial(object.featureSupport)
      : undefined;
    message.uninterpretedOption = object.uninterpretedOption?.map((e) => UninterpretedOption.fromPartial(e)) || [];
    return message;
  },
};

function createBaseFieldOptions_EditionDefault(): FieldOptions_EditionDefault {
  return { edition: Edition.EDITION_UNKNOWN, value: "" };
}

export const FieldOptions_EditionDefault = {
  encode(message: FieldOptions_EditionDefault, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.edition !== Edition.EDITION_UNKNOWN) {
      writer.uint32(24).int32(editionToNumber(message.edition));
    }
    if (message.value !== "") {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FieldOptions_EditionDefault {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFieldOptions_EditionDefault();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 3:
          if (tag !== 24) {
            break;
          }

          message.edition = editionFromJSON(reader.int32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FieldOptions_EditionDefault {
    return {
      edition: isSet(object.edition) ? editionFromJSON(object.edition) : Edition.EDITION_UNKNOWN,
      value: isSet(object.value) ? globalThis.String(object.value) : "",
    };
  },

  toJSON(message: FieldOptions_EditionDefault): unknown {
    const obj: any = {};
    if (message.edition !== Edition.EDITION_UNKNOWN) {
      obj.edition = editionToJSON(message.edition);
    }
    if (message.value !== "") {
      obj.value = message.value;
    }
    return obj;
  },

  create(base?: DeepPartial<FieldOptions_EditionDefault>): FieldOptions_EditionDefault {
    return FieldOptions_EditionDefault.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<FieldOptions_EditionDefault>): FieldOptions_EditionDefault {
    const message = createBaseFieldOptions_EditionDefault();
    message.edition = object.edition ?? Edition.EDITION_UNKNOWN;
    message.value = object.value ?? "";
    return message;
  },
};

function createBaseFieldOptions_FeatureSupport(): FieldOptions_FeatureSupport {
  return {
    editionIntroduced: Edition.EDITION_UNKNOWN,
    editionDeprecated: Edition.EDITION_UNKNOWN,
    deprecationWarning: "",
    editionRemoved: Edition.EDITION_UNKNOWN,
  };
}

export const FieldOptions_FeatureSupport = {
  encode(message: FieldOptions_FeatureSupport, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.editionIntroduced !== Edition.EDITION_UNKNOWN) {
      writer.uint32(8).int32(editionToNumber(message.editionIntroduced));
    }
    if (message.editionDeprecated !== Edition.EDITION_UNKNOWN) {
      writer.uint32(16).int32(editionToNumber(message.editionDeprecated));
    }
    if (message.deprecationWarning !== "") {
      writer.uint32(26).string(message.deprecationWarning);
    }
    if (message.editionRemoved !== Edition.EDITION_UNKNOWN) {
      writer.uint32(32).int32(editionToNumber(message.editionRemoved));
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FieldOptions_FeatureSupport {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFieldOptions_FeatureSupport();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.editionIntroduced = editionFromJSON(reader.int32());
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.editionDeprecated = editionFromJSON(reader.int32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.deprecationWarning = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.editionRemoved = editionFromJSON(reader.int32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FieldOptions_FeatureSupport {
    return {
      editionIntroduced: isSet(object.editionIntroduced)
        ? editionFromJSON(object.editionIntroduced)
        : Edition.EDITION_UNKNOWN,
      editionDeprecated: isSet(object.editionDeprecated)
        ? editionFromJSON(object.editionDeprecated)
        : Edition.EDITION_UNKNOWN,
      deprecationWarning: isSet(object.deprecationWarning) ? globalThis.String(object.deprecationWarning) : "",
      editionRemoved: isSet(object.editionRemoved) ? editionFromJSON(object.editionRemoved) : Edition.EDITION_UNKNOWN,
    };
  },

  toJSON(message: FieldOptions_FeatureSupport): unknown {
    const obj: any = {};
    if (message.editionIntroduced !== Edition.EDITION_UNKNOWN) {
      obj.editionIntroduced = editionToJSON(message.editionIntroduced);
    }
    if (message.editionDeprecated !== Edition.EDITION_UNKNOWN) {
      obj.editionDeprecated = editionToJSON(message.editionDeprecated);
    }
    if (message.deprecationWarning !== "") {
      obj.deprecationWarning = message.deprecationWarning;
    }
    if (message.editionRemoved !== Edition.EDITION_UNKNOWN) {
      obj.editionRemoved = editionToJSON(message.editionRemoved);
    }
    return obj;
  },

  create(base?: DeepPartial<FieldOptions_FeatureSupport>): FieldOptions_FeatureSupport {
    return FieldOptions_FeatureSupport.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<FieldOptions_FeatureSupport>): FieldOptions_FeatureSupport {
    const message = createBaseFieldOptions_FeatureSupport();
    message.editionIntroduced = object.editionIntroduced ?? Edition.EDITION_UNKNOWN;
    message.editionDeprecated = object.editionDeprecated ?? Edition.EDITION_UNKNOWN;
    message.deprecationWarning = object.deprecationWarning ?? "";
    message.editionRemoved = object.editionRemoved ?? Edition.EDITION_UNKNOWN;
    return message;
  },
};

function createBaseOneofOptions(): OneofOptions {
  return { features: undefined, uninterpretedOption: [] };
}

export const OneofOptions = {
  encode(message: OneofOptions, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.features !== undefined) {
      FeatureSet.encode(message.features, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.uninterpretedOption) {
      UninterpretedOption.encode(v!, writer.uint32(7994).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): OneofOptions {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseOneofOptions();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.features = FeatureSet.decode(reader, reader.uint32());
          continue;
        case 999:
          if (tag !== 7994) {
            break;
          }

          message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): OneofOptions {
    return {
      features: isSet(object.features) ? FeatureSet.fromJSON(object.features) : undefined,
      uninterpretedOption: globalThis.Array.isArray(object?.uninterpretedOption)
        ? object.uninterpretedOption.map((e: any) => UninterpretedOption.fromJSON(e))
        : [],
    };
  },

  toJSON(message: OneofOptions): unknown {
    const obj: any = {};
    if (message.features !== undefined) {
      obj.features = FeatureSet.toJSON(message.features);
    }
    if (message.uninterpretedOption?.length) {
      obj.uninterpretedOption = message.uninterpretedOption.map((e) => UninterpretedOption.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<OneofOptions>): OneofOptions {
    return OneofOptions.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<OneofOptions>): OneofOptions {
    const message = createBaseOneofOptions();
    message.features = (object.features !== undefined && object.features !== null)
      ? FeatureSet.fromPartial(object.features)
      : undefined;
    message.uninterpretedOption = object.uninterpretedOption?.map((e) => UninterpretedOption.fromPartial(e)) || [];
    return message;
  },
};

function createBaseEnumOptions(): EnumOptions {
  return {
    allowAlias: false,
    deprecated: false,
    deprecatedLegacyJsonFieldConflicts: false,
    features: undefined,
    uninterpretedOption: [],
  };
}

export const EnumOptions = {
  encode(message: EnumOptions, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.allowAlias === true) {
      writer.uint32(16).bool(message.allowAlias);
    }
    if (message.deprecated === true) {
      writer.uint32(24).bool(message.deprecated);
    }
    if (message.deprecatedLegacyJsonFieldConflicts === true) {
      writer.uint32(48).bool(message.deprecatedLegacyJsonFieldConflicts);
    }
    if (message.features !== undefined) {
      FeatureSet.encode(message.features, writer.uint32(58).fork()).ldelim();
    }
    for (const v of message.uninterpretedOption) {
      UninterpretedOption.encode(v!, writer.uint32(7994).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): EnumOptions {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEnumOptions();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 2:
          if (tag !== 16) {
            break;
          }

          message.allowAlias = reader.bool();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.deprecated = reader.bool();
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.deprecatedLegacyJsonFieldConflicts = reader.bool();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.features = FeatureSet.decode(reader, reader.uint32());
          continue;
        case 999:
          if (tag !== 7994) {
            break;
          }

          message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): EnumOptions {
    return {
      allowAlias: isSet(object.allowAlias) ? globalThis.Boolean(object.allowAlias) : false,
      deprecated: isSet(object.deprecated) ? globalThis.Boolean(object.deprecated) : false,
      deprecatedLegacyJsonFieldConflicts: isSet(object.deprecatedLegacyJsonFieldConflicts)
        ? globalThis.Boolean(object.deprecatedLegacyJsonFieldConflicts)
        : false,
      features: isSet(object.features) ? FeatureSet.fromJSON(object.features) : undefined,
      uninterpretedOption: globalThis.Array.isArray(object?.uninterpretedOption)
        ? object.uninterpretedOption.map((e: any) => UninterpretedOption.fromJSON(e))
        : [],
    };
  },

  toJSON(message: EnumOptions): unknown {
    const obj: any = {};
    if (message.allowAlias === true) {
      obj.allowAlias = message.allowAlias;
    }
    if (message.deprecated === true) {
      obj.deprecated = message.deprecated;
    }
    if (message.deprecatedLegacyJsonFieldConflicts === true) {
      obj.deprecatedLegacyJsonFieldConflicts = message.deprecatedLegacyJsonFieldConflicts;
    }
    if (message.features !== undefined) {
      obj.features = FeatureSet.toJSON(message.features);
    }
    if (message.uninterpretedOption?.length) {
      obj.uninterpretedOption = message.uninterpretedOption.map((e) => UninterpretedOption.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<EnumOptions>): EnumOptions {
    return EnumOptions.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<EnumOptions>): EnumOptions {
    const message = createBaseEnumOptions();
    message.allowAlias = object.allowAlias ?? false;
    message.deprecated = object.deprecated ?? false;
    message.deprecatedLegacyJsonFieldConflicts = object.deprecatedLegacyJsonFieldConflicts ?? false;
    message.features = (object.features !== undefined && object.features !== null)
      ? FeatureSet.fromPartial(object.features)
      : undefined;
    message.uninterpretedOption = object.uninterpretedOption?.map((e) => UninterpretedOption.fromPartial(e)) || [];
    return message;
  },
};

function createBaseEnumValueOptions(): EnumValueOptions {
  return {
    deprecated: false,
    features: undefined,
    debugRedact: false,
    featureSupport: undefined,
    uninterpretedOption: [],
  };
}

export const EnumValueOptions = {
  encode(message: EnumValueOptions, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.deprecated === true) {
      writer.uint32(8).bool(message.deprecated);
    }
    if (message.features !== undefined) {
      FeatureSet.encode(message.features, writer.uint32(18).fork()).ldelim();
    }
    if (message.debugRedact === true) {
      writer.uint32(24).bool(message.debugRedact);
    }
    if (message.featureSupport !== undefined) {
      FieldOptions_FeatureSupport.encode(message.featureSupport, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.uninterpretedOption) {
      UninterpretedOption.encode(v!, writer.uint32(7994).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): EnumValueOptions {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEnumValueOptions();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.deprecated = reader.bool();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.features = FeatureSet.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.debugRedact = reader.bool();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.featureSupport = FieldOptions_FeatureSupport.decode(reader, reader.uint32());
          continue;
        case 999:
          if (tag !== 7994) {
            break;
          }

          message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): EnumValueOptions {
    return {
      deprecated: isSet(object.deprecated) ? globalThis.Boolean(object.deprecated) : false,
      features: isSet(object.features) ? FeatureSet.fromJSON(object.features) : undefined,
      debugRedact: isSet(object.debugRedact) ? globalThis.Boolean(object.debugRedact) : false,
      featureSupport: isSet(object.featureSupport)
        ? FieldOptions_FeatureSupport.fromJSON(object.featureSupport)
        : undefined,
      uninterpretedOption: globalThis.Array.isArray(object?.uninterpretedOption)
        ? object.uninterpretedOption.map((e: any) => UninterpretedOption.fromJSON(e))
        : [],
    };
  },

  toJSON(message: EnumValueOptions): unknown {
    const obj: any = {};
    if (message.deprecated === true) {
      obj.deprecated = message.deprecated;
    }
    if (message.features !== undefined) {
      obj.features = FeatureSet.toJSON(message.features);
    }
    if (message.debugRedact === true) {
      obj.debugRedact = message.debugRedact;
    }
    if (message.featureSupport !== undefined) {
      obj.featureSupport = FieldOptions_FeatureSupport.toJSON(message.featureSupport);
    }
    if (message.uninterpretedOption?.length) {
      obj.uninterpretedOption = message.uninterpretedOption.map((e) => UninterpretedOption.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<EnumValueOptions>): EnumValueOptions {
    return EnumValueOptions.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<EnumValueOptions>): EnumValueOptions {
    const message = createBaseEnumValueOptions();
    message.deprecated = object.deprecated ?? false;
    message.features = (object.features !== undefined && object.features !== null)
      ? FeatureSet.fromPartial(object.features)
      : undefined;
    message.debugRedact = object.debugRedact ?? false;
    message.featureSupport = (object.featureSupport !== undefined && object.featureSupport !== null)
      ? FieldOptions_FeatureSupport.fromPartial(object.featureSupport)
      : undefined;
    message.uninterpretedOption = object.uninterpretedOption?.map((e) => UninterpretedOption.fromPartial(e)) || [];
    return message;
  },
};

function createBaseServiceOptions(): ServiceOptions {
  return { features: undefined, deprecated: false, uninterpretedOption: [] };
}

export const ServiceOptions = {
  encode(message: ServiceOptions, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.features !== undefined) {
      FeatureSet.encode(message.features, writer.uint32(274).fork()).ldelim();
    }
    if (message.deprecated === true) {
      writer.uint32(264).bool(message.deprecated);
    }
    for (const v of message.uninterpretedOption) {
      UninterpretedOption.encode(v!, writer.uint32(7994).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ServiceOptions {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseServiceOptions();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 34:
          if (tag !== 274) {
            break;
          }

          message.features = FeatureSet.decode(reader, reader.uint32());
          continue;
        case 33:
          if (tag !== 264) {
            break;
          }

          message.deprecated = reader.bool();
          continue;
        case 999:
          if (tag !== 7994) {
            break;
          }

          message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ServiceOptions {
    return {
      features: isSet(object.features) ? FeatureSet.fromJSON(object.features) : undefined,
      deprecated: isSet(object.deprecated) ? globalThis.Boolean(object.deprecated) : false,
      uninterpretedOption: globalThis.Array.isArray(object?.uninterpretedOption)
        ? object.uninterpretedOption.map((e: any) => UninterpretedOption.fromJSON(e))
        : [],
    };
  },

  toJSON(message: ServiceOptions): unknown {
    const obj: any = {};
    if (message.features !== undefined) {
      obj.features = FeatureSet.toJSON(message.features);
    }
    if (message.deprecated === true) {
      obj.deprecated = message.deprecated;
    }
    if (message.uninterpretedOption?.length) {
      obj.uninterpretedOption = message.uninterpretedOption.map((e) => UninterpretedOption.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ServiceOptions>): ServiceOptions {
    return ServiceOptions.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ServiceOptions>): ServiceOptions {
    const message = createBaseServiceOptions();
    message.features = (object.features !== undefined && object.features !== null)
      ? FeatureSet.fromPartial(object.features)
      : undefined;
    message.deprecated = object.deprecated ?? false;
    message.uninterpretedOption = object.uninterpretedOption?.map((e) => UninterpretedOption.fromPartial(e)) || [];
    return message;
  },
};

function createBaseMethodOptions(): MethodOptions {
  return {
    deprecated: false,
    idempotencyLevel: MethodOptions_IdempotencyLevel.IDEMPOTENCY_UNKNOWN,
    features: undefined,
    uninterpretedOption: [],
  };
}

export const MethodOptions = {
  encode(message: MethodOptions, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.deprecated === true) {
      writer.uint32(264).bool(message.deprecated);
    }
    if (message.idempotencyLevel !== MethodOptions_IdempotencyLevel.IDEMPOTENCY_UNKNOWN) {
      writer.uint32(272).int32(methodOptions_IdempotencyLevelToNumber(message.idempotencyLevel));
    }
    if (message.features !== undefined) {
      FeatureSet.encode(message.features, writer.uint32(282).fork()).ldelim();
    }
    for (const v of message.uninterpretedOption) {
      UninterpretedOption.encode(v!, writer.uint32(7994).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MethodOptions {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMethodOptions();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 33:
          if (tag !== 264) {
            break;
          }

          message.deprecated = reader.bool();
          continue;
        case 34:
          if (tag !== 272) {
            break;
          }

          message.idempotencyLevel = methodOptions_IdempotencyLevelFromJSON(reader.int32());
          continue;
        case 35:
          if (tag !== 282) {
            break;
          }

          message.features = FeatureSet.decode(reader, reader.uint32());
          continue;
        case 999:
          if (tag !== 7994) {
            break;
          }

          message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MethodOptions {
    return {
      deprecated: isSet(object.deprecated) ? globalThis.Boolean(object.deprecated) : false,
      idempotencyLevel: isSet(object.idempotencyLevel)
        ? methodOptions_IdempotencyLevelFromJSON(object.idempotencyLevel)
        : MethodOptions_IdempotencyLevel.IDEMPOTENCY_UNKNOWN,
      features: isSet(object.features) ? FeatureSet.fromJSON(object.features) : undefined,
      uninterpretedOption: globalThis.Array.isArray(object?.uninterpretedOption)
        ? object.uninterpretedOption.map((e: any) => UninterpretedOption.fromJSON(e))
        : [],
    };
  },

  toJSON(message: MethodOptions): unknown {
    const obj: any = {};
    if (message.deprecated === true) {
      obj.deprecated = message.deprecated;
    }
    if (message.idempotencyLevel !== MethodOptions_IdempotencyLevel.IDEMPOTENCY_UNKNOWN) {
      obj.idempotencyLevel = methodOptions_IdempotencyLevelToJSON(message.idempotencyLevel);
    }
    if (message.features !== undefined) {
      obj.features = FeatureSet.toJSON(message.features);
    }
    if (message.uninterpretedOption?.length) {
      obj.uninterpretedOption = message.uninterpretedOption.map((e) => UninterpretedOption.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<MethodOptions>): MethodOptions {
    return MethodOptions.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<MethodOptions>): MethodOptions {
    const message = createBaseMethodOptions();
    message.deprecated = object.deprecated ?? false;
    message.idempotencyLevel = object.idempotencyLevel ?? MethodOptions_IdempotencyLevel.IDEMPOTENCY_UNKNOWN;
    message.features = (object.features !== undefined && object.features !== null)
      ? FeatureSet.fromPartial(object.features)
      : undefined;
    message.uninterpretedOption = object.uninterpretedOption?.map((e) => UninterpretedOption.fromPartial(e)) || [];
    return message;
  },
};

function createBaseUninterpretedOption(): UninterpretedOption {
  return {
    name: [],
    identifierValue: "",
    positiveIntValue: Long.UZERO,
    negativeIntValue: Long.ZERO,
    doubleValue: 0,
    stringValue: new Uint8Array(0),
    aggregateValue: "",
  };
}

export const UninterpretedOption = {
  encode(message: UninterpretedOption, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.name) {
      UninterpretedOption_NamePart.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.identifierValue !== "") {
      writer.uint32(26).string(message.identifierValue);
    }
    if (!message.positiveIntValue.isZero()) {
      writer.uint32(32).uint64(message.positiveIntValue);
    }
    if (!message.negativeIntValue.isZero()) {
      writer.uint32(40).int64(message.negativeIntValue);
    }
    if (message.doubleValue !== 0) {
      writer.uint32(49).double(message.doubleValue);
    }
    if (message.stringValue.length !== 0) {
      writer.uint32(58).bytes(message.stringValue);
    }
    if (message.aggregateValue !== "") {
      writer.uint32(66).string(message.aggregateValue);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UninterpretedOption {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUninterpretedOption();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 2:
          if (tag !== 18) {
            break;
          }

          message.name.push(UninterpretedOption_NamePart.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.identifierValue = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.positiveIntValue = reader.uint64() as Long;
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.negativeIntValue = reader.int64() as Long;
          continue;
        case 6:
          if (tag !== 49) {
            break;
          }

          message.doubleValue = reader.double();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.stringValue = reader.bytes();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.aggregateValue = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UninterpretedOption {
    return {
      name: globalThis.Array.isArray(object?.name)
        ? object.name.map((e: any) => UninterpretedOption_NamePart.fromJSON(e))
        : [],
      identifierValue: isSet(object.identifierValue) ? globalThis.String(object.identifierValue) : "",
      positiveIntValue: isSet(object.positiveIntValue) ? Long.fromValue(object.positiveIntValue) : Long.UZERO,
      negativeIntValue: isSet(object.negativeIntValue) ? Long.fromValue(object.negativeIntValue) : Long.ZERO,
      doubleValue: isSet(object.doubleValue) ? globalThis.Number(object.doubleValue) : 0,
      stringValue: isSet(object.stringValue) ? bytesFromBase64(object.stringValue) : new Uint8Array(0),
      aggregateValue: isSet(object.aggregateValue) ? globalThis.String(object.aggregateValue) : "",
    };
  },

  toJSON(message: UninterpretedOption): unknown {
    const obj: any = {};
    if (message.name?.length) {
      obj.name = message.name.map((e) => UninterpretedOption_NamePart.toJSON(e));
    }
    if (message.identifierValue !== "") {
      obj.identifierValue = message.identifierValue;
    }
    if (!message.positiveIntValue.isZero()) {
      obj.positiveIntValue = (message.positiveIntValue || Long.UZERO).toString();
    }
    if (!message.negativeIntValue.isZero()) {
      obj.negativeIntValue = (message.negativeIntValue || Long.ZERO).toString();
    }
    if (message.doubleValue !== 0) {
      obj.doubleValue = message.doubleValue;
    }
    if (message.stringValue.length !== 0) {
      obj.stringValue = base64FromBytes(message.stringValue);
    }
    if (message.aggregateValue !== "") {
      obj.aggregateValue = message.aggregateValue;
    }
    return obj;
  },

  create(base?: DeepPartial<UninterpretedOption>): UninterpretedOption {
    return UninterpretedOption.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UninterpretedOption>): UninterpretedOption {
    const message = createBaseUninterpretedOption();
    message.name = object.name?.map((e) => UninterpretedOption_NamePart.fromPartial(e)) || [];
    message.identifierValue = object.identifierValue ?? "";
    message.positiveIntValue = (object.positiveIntValue !== undefined && object.positiveIntValue !== null)
      ? Long.fromValue(object.positiveIntValue)
      : Long.UZERO;
    message.negativeIntValue = (object.negativeIntValue !== undefined && object.negativeIntValue !== null)
      ? Long.fromValue(object.negativeIntValue)
      : Long.ZERO;
    message.doubleValue = object.doubleValue ?? 0;
    message.stringValue = object.stringValue ?? new Uint8Array(0);
    message.aggregateValue = object.aggregateValue ?? "";
    return message;
  },
};

function createBaseUninterpretedOption_NamePart(): UninterpretedOption_NamePart {
  return { namePart: "", isExtension: false };
}

export const UninterpretedOption_NamePart = {
  encode(message: UninterpretedOption_NamePart, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.namePart !== "") {
      writer.uint32(10).string(message.namePart);
    }
    if (message.isExtension === true) {
      writer.uint32(16).bool(message.isExtension);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UninterpretedOption_NamePart {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUninterpretedOption_NamePart();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.namePart = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.isExtension = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UninterpretedOption_NamePart {
    return {
      namePart: isSet(object.namePart) ? globalThis.String(object.namePart) : "",
      isExtension: isSet(object.isExtension) ? globalThis.Boolean(object.isExtension) : false,
    };
  },

  toJSON(message: UninterpretedOption_NamePart): unknown {
    const obj: any = {};
    if (message.namePart !== "") {
      obj.namePart = message.namePart;
    }
    if (message.isExtension === true) {
      obj.isExtension = message.isExtension;
    }
    return obj;
  },

  create(base?: DeepPartial<UninterpretedOption_NamePart>): UninterpretedOption_NamePart {
    return UninterpretedOption_NamePart.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UninterpretedOption_NamePart>): UninterpretedOption_NamePart {
    const message = createBaseUninterpretedOption_NamePart();
    message.namePart = object.namePart ?? "";
    message.isExtension = object.isExtension ?? false;
    return message;
  },
};

function createBaseFeatureSet(): FeatureSet {
  return {
    fieldPresence: FeatureSet_FieldPresence.FIELD_PRESENCE_UNKNOWN,
    enumType: FeatureSet_EnumType.ENUM_TYPE_UNKNOWN,
    repeatedFieldEncoding: FeatureSet_RepeatedFieldEncoding.REPEATED_FIELD_ENCODING_UNKNOWN,
    utf8Validation: FeatureSet_Utf8Validation.UTF8_VALIDATION_UNKNOWN,
    messageEncoding: FeatureSet_MessageEncoding.MESSAGE_ENCODING_UNKNOWN,
    jsonFormat: FeatureSet_JsonFormat.JSON_FORMAT_UNKNOWN,
  };
}

export const FeatureSet = {
  encode(message: FeatureSet, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.fieldPresence !== FeatureSet_FieldPresence.FIELD_PRESENCE_UNKNOWN) {
      writer.uint32(8).int32(featureSet_FieldPresenceToNumber(message.fieldPresence));
    }
    if (message.enumType !== FeatureSet_EnumType.ENUM_TYPE_UNKNOWN) {
      writer.uint32(16).int32(featureSet_EnumTypeToNumber(message.enumType));
    }
    if (message.repeatedFieldEncoding !== FeatureSet_RepeatedFieldEncoding.REPEATED_FIELD_ENCODING_UNKNOWN) {
      writer.uint32(24).int32(featureSet_RepeatedFieldEncodingToNumber(message.repeatedFieldEncoding));
    }
    if (message.utf8Validation !== FeatureSet_Utf8Validation.UTF8_VALIDATION_UNKNOWN) {
      writer.uint32(32).int32(featureSet_Utf8ValidationToNumber(message.utf8Validation));
    }
    if (message.messageEncoding !== FeatureSet_MessageEncoding.MESSAGE_ENCODING_UNKNOWN) {
      writer.uint32(40).int32(featureSet_MessageEncodingToNumber(message.messageEncoding));
    }
    if (message.jsonFormat !== FeatureSet_JsonFormat.JSON_FORMAT_UNKNOWN) {
      writer.uint32(48).int32(featureSet_JsonFormatToNumber(message.jsonFormat));
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FeatureSet {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFeatureSet();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.fieldPresence = featureSet_FieldPresenceFromJSON(reader.int32());
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.enumType = featureSet_EnumTypeFromJSON(reader.int32());
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.repeatedFieldEncoding = featureSet_RepeatedFieldEncodingFromJSON(reader.int32());
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.utf8Validation = featureSet_Utf8ValidationFromJSON(reader.int32());
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.messageEncoding = featureSet_MessageEncodingFromJSON(reader.int32());
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.jsonFormat = featureSet_JsonFormatFromJSON(reader.int32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FeatureSet {
    return {
      fieldPresence: isSet(object.fieldPresence)
        ? featureSet_FieldPresenceFromJSON(object.fieldPresence)
        : FeatureSet_FieldPresence.FIELD_PRESENCE_UNKNOWN,
      enumType: isSet(object.enumType)
        ? featureSet_EnumTypeFromJSON(object.enumType)
        : FeatureSet_EnumType.ENUM_TYPE_UNKNOWN,
      repeatedFieldEncoding: isSet(object.repeatedFieldEncoding)
        ? featureSet_RepeatedFieldEncodingFromJSON(object.repeatedFieldEncoding)
        : FeatureSet_RepeatedFieldEncoding.REPEATED_FIELD_ENCODING_UNKNOWN,
      utf8Validation: isSet(object.utf8Validation)
        ? featureSet_Utf8ValidationFromJSON(object.utf8Validation)
        : FeatureSet_Utf8Validation.UTF8_VALIDATION_UNKNOWN,
      messageEncoding: isSet(object.messageEncoding)
        ? featureSet_MessageEncodingFromJSON(object.messageEncoding)
        : FeatureSet_MessageEncoding.MESSAGE_ENCODING_UNKNOWN,
      jsonFormat: isSet(object.jsonFormat)
        ? featureSet_JsonFormatFromJSON(object.jsonFormat)
        : FeatureSet_JsonFormat.JSON_FORMAT_UNKNOWN,
    };
  },

  toJSON(message: FeatureSet): unknown {
    const obj: any = {};
    if (message.fieldPresence !== FeatureSet_FieldPresence.FIELD_PRESENCE_UNKNOWN) {
      obj.fieldPresence = featureSet_FieldPresenceToJSON(message.fieldPresence);
    }
    if (message.enumType !== FeatureSet_EnumType.ENUM_TYPE_UNKNOWN) {
      obj.enumType = featureSet_EnumTypeToJSON(message.enumType);
    }
    if (message.repeatedFieldEncoding !== FeatureSet_RepeatedFieldEncoding.REPEATED_FIELD_ENCODING_UNKNOWN) {
      obj.repeatedFieldEncoding = featureSet_RepeatedFieldEncodingToJSON(message.repeatedFieldEncoding);
    }
    if (message.utf8Validation !== FeatureSet_Utf8Validation.UTF8_VALIDATION_UNKNOWN) {
      obj.utf8Validation = featureSet_Utf8ValidationToJSON(message.utf8Validation);
    }
    if (message.messageEncoding !== FeatureSet_MessageEncoding.MESSAGE_ENCODING_UNKNOWN) {
      obj.messageEncoding = featureSet_MessageEncodingToJSON(message.messageEncoding);
    }
    if (message.jsonFormat !== FeatureSet_JsonFormat.JSON_FORMAT_UNKNOWN) {
      obj.jsonFormat = featureSet_JsonFormatToJSON(message.jsonFormat);
    }
    return obj;
  },

  create(base?: DeepPartial<FeatureSet>): FeatureSet {
    return FeatureSet.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<FeatureSet>): FeatureSet {
    const message = createBaseFeatureSet();
    message.fieldPresence = object.fieldPresence ?? FeatureSet_FieldPresence.FIELD_PRESENCE_UNKNOWN;
    message.enumType = object.enumType ?? FeatureSet_EnumType.ENUM_TYPE_UNKNOWN;
    message.repeatedFieldEncoding = object.repeatedFieldEncoding ??
      FeatureSet_RepeatedFieldEncoding.REPEATED_FIELD_ENCODING_UNKNOWN;
    message.utf8Validation = object.utf8Validation ?? FeatureSet_Utf8Validation.UTF8_VALIDATION_UNKNOWN;
    message.messageEncoding = object.messageEncoding ?? FeatureSet_MessageEncoding.MESSAGE_ENCODING_UNKNOWN;
    message.jsonFormat = object.jsonFormat ?? FeatureSet_JsonFormat.JSON_FORMAT_UNKNOWN;
    return message;
  },
};

function createBaseFeatureSetDefaults(): FeatureSetDefaults {
  return { defaults: [], minimumEdition: Edition.EDITION_UNKNOWN, maximumEdition: Edition.EDITION_UNKNOWN };
}

export const FeatureSetDefaults = {
  encode(message: FeatureSetDefaults, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.defaults) {
      FeatureSetDefaults_FeatureSetEditionDefault.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.minimumEdition !== Edition.EDITION_UNKNOWN) {
      writer.uint32(32).int32(editionToNumber(message.minimumEdition));
    }
    if (message.maximumEdition !== Edition.EDITION_UNKNOWN) {
      writer.uint32(40).int32(editionToNumber(message.maximumEdition));
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FeatureSetDefaults {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFeatureSetDefaults();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.defaults.push(FeatureSetDefaults_FeatureSetEditionDefault.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.minimumEdition = editionFromJSON(reader.int32());
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.maximumEdition = editionFromJSON(reader.int32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FeatureSetDefaults {
    return {
      defaults: globalThis.Array.isArray(object?.defaults)
        ? object.defaults.map((e: any) => FeatureSetDefaults_FeatureSetEditionDefault.fromJSON(e))
        : [],
      minimumEdition: isSet(object.minimumEdition) ? editionFromJSON(object.minimumEdition) : Edition.EDITION_UNKNOWN,
      maximumEdition: isSet(object.maximumEdition) ? editionFromJSON(object.maximumEdition) : Edition.EDITION_UNKNOWN,
    };
  },

  toJSON(message: FeatureSetDefaults): unknown {
    const obj: any = {};
    if (message.defaults?.length) {
      obj.defaults = message.defaults.map((e) => FeatureSetDefaults_FeatureSetEditionDefault.toJSON(e));
    }
    if (message.minimumEdition !== Edition.EDITION_UNKNOWN) {
      obj.minimumEdition = editionToJSON(message.minimumEdition);
    }
    if (message.maximumEdition !== Edition.EDITION_UNKNOWN) {
      obj.maximumEdition = editionToJSON(message.maximumEdition);
    }
    return obj;
  },

  create(base?: DeepPartial<FeatureSetDefaults>): FeatureSetDefaults {
    return FeatureSetDefaults.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<FeatureSetDefaults>): FeatureSetDefaults {
    const message = createBaseFeatureSetDefaults();
    message.defaults = object.defaults?.map((e) => FeatureSetDefaults_FeatureSetEditionDefault.fromPartial(e)) || [];
    message.minimumEdition = object.minimumEdition ?? Edition.EDITION_UNKNOWN;
    message.maximumEdition = object.maximumEdition ?? Edition.EDITION_UNKNOWN;
    return message;
  },
};

function createBaseFeatureSetDefaults_FeatureSetEditionDefault(): FeatureSetDefaults_FeatureSetEditionDefault {
  return { edition: Edition.EDITION_UNKNOWN, overridableFeatures: undefined, fixedFeatures: undefined };
}

export const FeatureSetDefaults_FeatureSetEditionDefault = {
  encode(message: FeatureSetDefaults_FeatureSetEditionDefault, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.edition !== Edition.EDITION_UNKNOWN) {
      writer.uint32(24).int32(editionToNumber(message.edition));
    }
    if (message.overridableFeatures !== undefined) {
      FeatureSet.encode(message.overridableFeatures, writer.uint32(34).fork()).ldelim();
    }
    if (message.fixedFeatures !== undefined) {
      FeatureSet.encode(message.fixedFeatures, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FeatureSetDefaults_FeatureSetEditionDefault {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFeatureSetDefaults_FeatureSetEditionDefault();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 3:
          if (tag !== 24) {
            break;
          }

          message.edition = editionFromJSON(reader.int32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.overridableFeatures = FeatureSet.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.fixedFeatures = FeatureSet.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FeatureSetDefaults_FeatureSetEditionDefault {
    return {
      edition: isSet(object.edition) ? editionFromJSON(object.edition) : Edition.EDITION_UNKNOWN,
      overridableFeatures: isSet(object.overridableFeatures)
        ? FeatureSet.fromJSON(object.overridableFeatures)
        : undefined,
      fixedFeatures: isSet(object.fixedFeatures) ? FeatureSet.fromJSON(object.fixedFeatures) : undefined,
    };
  },

  toJSON(message: FeatureSetDefaults_FeatureSetEditionDefault): unknown {
    const obj: any = {};
    if (message.edition !== Edition.EDITION_UNKNOWN) {
      obj.edition = editionToJSON(message.edition);
    }
    if (message.overridableFeatures !== undefined) {
      obj.overridableFeatures = FeatureSet.toJSON(message.overridableFeatures);
    }
    if (message.fixedFeatures !== undefined) {
      obj.fixedFeatures = FeatureSet.toJSON(message.fixedFeatures);
    }
    return obj;
  },

  create(base?: DeepPartial<FeatureSetDefaults_FeatureSetEditionDefault>): FeatureSetDefaults_FeatureSetEditionDefault {
    return FeatureSetDefaults_FeatureSetEditionDefault.fromPartial(base ?? {});
  },
  fromPartial(
    object: DeepPartial<FeatureSetDefaults_FeatureSetEditionDefault>,
  ): FeatureSetDefaults_FeatureSetEditionDefault {
    const message = createBaseFeatureSetDefaults_FeatureSetEditionDefault();
    message.edition = object.edition ?? Edition.EDITION_UNKNOWN;
    message.overridableFeatures = (object.overridableFeatures !== undefined && object.overridableFeatures !== null)
      ? FeatureSet.fromPartial(object.overridableFeatures)
      : undefined;
    message.fixedFeatures = (object.fixedFeatures !== undefined && object.fixedFeatures !== null)
      ? FeatureSet.fromPartial(object.fixedFeatures)
      : undefined;
    return message;
  },
};

function createBaseSourceCodeInfo(): SourceCodeInfo {
  return { location: [] };
}

export const SourceCodeInfo = {
  encode(message: SourceCodeInfo, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.location) {
      SourceCodeInfo_Location.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SourceCodeInfo {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSourceCodeInfo();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.location.push(SourceCodeInfo_Location.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SourceCodeInfo {
    return {
      location: globalThis.Array.isArray(object?.location)
        ? object.location.map((e: any) => SourceCodeInfo_Location.fromJSON(e))
        : [],
    };
  },

  toJSON(message: SourceCodeInfo): unknown {
    const obj: any = {};
    if (message.location?.length) {
      obj.location = message.location.map((e) => SourceCodeInfo_Location.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<SourceCodeInfo>): SourceCodeInfo {
    return SourceCodeInfo.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SourceCodeInfo>): SourceCodeInfo {
    const message = createBaseSourceCodeInfo();
    message.location = object.location?.map((e) => SourceCodeInfo_Location.fromPartial(e)) || [];
    return message;
  },
};

function createBaseSourceCodeInfo_Location(): SourceCodeInfo_Location {
  return { path: [], span: [], leadingComments: "", trailingComments: "", leadingDetachedComments: [] };
}

export const SourceCodeInfo_Location = {
  encode(message: SourceCodeInfo_Location, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    writer.uint32(10).fork();
    for (const v of message.path) {
      writer.int32(v);
    }
    writer.ldelim();
    writer.uint32(18).fork();
    for (const v of message.span) {
      writer.int32(v);
    }
    writer.ldelim();
    if (message.leadingComments !== "") {
      writer.uint32(26).string(message.leadingComments);
    }
    if (message.trailingComments !== "") {
      writer.uint32(34).string(message.trailingComments);
    }
    for (const v of message.leadingDetachedComments) {
      writer.uint32(50).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SourceCodeInfo_Location {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSourceCodeInfo_Location();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag === 8) {
            message.path.push(reader.int32());

            continue;
          }

          if (tag === 10) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.path.push(reader.int32());
            }

            continue;
          }

          break;
        case 2:
          if (tag === 16) {
            message.span.push(reader.int32());

            continue;
          }

          if (tag === 18) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.span.push(reader.int32());
            }

            continue;
          }

          break;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.leadingComments = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.trailingComments = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.leadingDetachedComments.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SourceCodeInfo_Location {
    return {
      path: globalThis.Array.isArray(object?.path) ? object.path.map((e: any) => globalThis.Number(e)) : [],
      span: globalThis.Array.isArray(object?.span) ? object.span.map((e: any) => globalThis.Number(e)) : [],
      leadingComments: isSet(object.leadingComments) ? globalThis.String(object.leadingComments) : "",
      trailingComments: isSet(object.trailingComments) ? globalThis.String(object.trailingComments) : "",
      leadingDetachedComments: globalThis.Array.isArray(object?.leadingDetachedComments)
        ? object.leadingDetachedComments.map((e: any) => globalThis.String(e))
        : [],
    };
  },

  toJSON(message: SourceCodeInfo_Location): unknown {
    const obj: any = {};
    if (message.path?.length) {
      obj.path = message.path.map((e) => Math.round(e));
    }
    if (message.span?.length) {
      obj.span = message.span.map((e) => Math.round(e));
    }
    if (message.leadingComments !== "") {
      obj.leadingComments = message.leadingComments;
    }
    if (message.trailingComments !== "") {
      obj.trailingComments = message.trailingComments;
    }
    if (message.leadingDetachedComments?.length) {
      obj.leadingDetachedComments = message.leadingDetachedComments;
    }
    return obj;
  },

  create(base?: DeepPartial<SourceCodeInfo_Location>): SourceCodeInfo_Location {
    return SourceCodeInfo_Location.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SourceCodeInfo_Location>): SourceCodeInfo_Location {
    const message = createBaseSourceCodeInfo_Location();
    message.path = object.path?.map((e) => e) || [];
    message.span = object.span?.map((e) => e) || [];
    message.leadingComments = object.leadingComments ?? "";
    message.trailingComments = object.trailingComments ?? "";
    message.leadingDetachedComments = object.leadingDetachedComments?.map((e) => e) || [];
    return message;
  },
};

function createBaseGeneratedCodeInfo(): GeneratedCodeInfo {
  return { annotation: [] };
}

export const GeneratedCodeInfo = {
  encode(message: GeneratedCodeInfo, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.annotation) {
      GeneratedCodeInfo_Annotation.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GeneratedCodeInfo {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGeneratedCodeInfo();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.annotation.push(GeneratedCodeInfo_Annotation.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GeneratedCodeInfo {
    return {
      annotation: globalThis.Array.isArray(object?.annotation)
        ? object.annotation.map((e: any) => GeneratedCodeInfo_Annotation.fromJSON(e))
        : [],
    };
  },

  toJSON(message: GeneratedCodeInfo): unknown {
    const obj: any = {};
    if (message.annotation?.length) {
      obj.annotation = message.annotation.map((e) => GeneratedCodeInfo_Annotation.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<GeneratedCodeInfo>): GeneratedCodeInfo {
    return GeneratedCodeInfo.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GeneratedCodeInfo>): GeneratedCodeInfo {
    const message = createBaseGeneratedCodeInfo();
    message.annotation = object.annotation?.map((e) => GeneratedCodeInfo_Annotation.fromPartial(e)) || [];
    return message;
  },
};

function createBaseGeneratedCodeInfo_Annotation(): GeneratedCodeInfo_Annotation {
  return { path: [], sourceFile: "", begin: 0, end: 0, semantic: GeneratedCodeInfo_Annotation_Semantic.NONE };
}

export const GeneratedCodeInfo_Annotation = {
  encode(message: GeneratedCodeInfo_Annotation, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    writer.uint32(10).fork();
    for (const v of message.path) {
      writer.int32(v);
    }
    writer.ldelim();
    if (message.sourceFile !== "") {
      writer.uint32(18).string(message.sourceFile);
    }
    if (message.begin !== 0) {
      writer.uint32(24).int32(message.begin);
    }
    if (message.end !== 0) {
      writer.uint32(32).int32(message.end);
    }
    if (message.semantic !== GeneratedCodeInfo_Annotation_Semantic.NONE) {
      writer.uint32(40).int32(generatedCodeInfo_Annotation_SemanticToNumber(message.semantic));
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GeneratedCodeInfo_Annotation {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGeneratedCodeInfo_Annotation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag === 8) {
            message.path.push(reader.int32());

            continue;
          }

          if (tag === 10) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.path.push(reader.int32());
            }

            continue;
          }

          break;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.sourceFile = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.begin = reader.int32();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.end = reader.int32();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.semantic = generatedCodeInfo_Annotation_SemanticFromJSON(reader.int32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GeneratedCodeInfo_Annotation {
    return {
      path: globalThis.Array.isArray(object?.path) ? object.path.map((e: any) => globalThis.Number(e)) : [],
      sourceFile: isSet(object.sourceFile) ? globalThis.String(object.sourceFile) : "",
      begin: isSet(object.begin) ? globalThis.Number(object.begin) : 0,
      end: isSet(object.end) ? globalThis.Number(object.end) : 0,
      semantic: isSet(object.semantic)
        ? generatedCodeInfo_Annotation_SemanticFromJSON(object.semantic)
        : GeneratedCodeInfo_Annotation_Semantic.NONE,
    };
  },

  toJSON(message: GeneratedCodeInfo_Annotation): unknown {
    const obj: any = {};
    if (message.path?.length) {
      obj.path = message.path.map((e) => Math.round(e));
    }
    if (message.sourceFile !== "") {
      obj.sourceFile = message.sourceFile;
    }
    if (message.begin !== 0) {
      obj.begin = Math.round(message.begin);
    }
    if (message.end !== 0) {
      obj.end = Math.round(message.end);
    }
    if (message.semantic !== GeneratedCodeInfo_Annotation_Semantic.NONE) {
      obj.semantic = generatedCodeInfo_Annotation_SemanticToJSON(message.semantic);
    }
    return obj;
  },

  create(base?: DeepPartial<GeneratedCodeInfo_Annotation>): GeneratedCodeInfo_Annotation {
    return GeneratedCodeInfo_Annotation.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GeneratedCodeInfo_Annotation>): GeneratedCodeInfo_Annotation {
    const message = createBaseGeneratedCodeInfo_Annotation();
    message.path = object.path?.map((e) => e) || [];
    message.sourceFile = object.sourceFile ?? "";
    message.begin = object.begin ?? 0;
    message.end = object.end ?? 0;
    message.semantic = object.semantic ?? GeneratedCodeInfo_Annotation_Semantic.NONE;
    return message;
  },
};

function bytesFromBase64(b64: string): Uint8Array {
  if (globalThis.Buffer) {
    return Uint8Array.from(globalThis.Buffer.from(b64, "base64"));
  } else {
    const bin = globalThis.atob(b64);
    const arr = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; ++i) {
      arr[i] = bin.charCodeAt(i);
    }
    return arr;
  }
}

function base64FromBytes(arr: Uint8Array): string {
  if (globalThis.Buffer) {
    return globalThis.Buffer.from(arr).toString("base64");
  } else {
    const bin: string[] = [];
    arr.forEach((byte) => {
      bin.push(globalThis.String.fromCharCode(byte));
    });
    return globalThis.btoa(bin.join(""));
  }
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Long ? string | number | Long : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
