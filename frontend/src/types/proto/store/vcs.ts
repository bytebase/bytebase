/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface VCSConnector {
  /** The title or display name of the VCS connector. */
  title: string;
  /**
   * Full path from the corresponding VCS provider.
   * For GitLab, this is the project full path. e.g. group1/project-1
   */
  fullPath: string;
  /**
   * Web url from the corresponding VCS provider.
   * For GitLab, this is the project web url. e.g. https://gitlab.example.com/group1/project-1
   */
  webUrl: string;
  /** Branch to listen to. */
  branch: string;
  /** Base working directory we are interested. */
  baseDirectory: string;
  /**
   * Repository id from the corresponding VCS provider.
   * For GitLab, this is the project id. e.g. 123
   */
  externalId: string;
  /**
   * Push webhook id from the corresponding VCS provider.
   * For GitLab, this is the project webhook id. e.g. 123
   */
  externalWebhookId: string;
  /** For GitLab, webhook request contains this in the 'X-Gitlab-Token" header and we compare it with the one stored in db to validate it sends to the expected endpoint. */
  webhookSecretToken: string;
  /**
   * Apply changes to the database group. Optional, if not set, will apply changes to all databases in the project.
   * Format: projects/{project}/databaseGroups/{databaseGroup}
   */
  databaseGroup: string;
}

function createBaseVCSConnector(): VCSConnector {
  return {
    title: "",
    fullPath: "",
    webUrl: "",
    branch: "",
    baseDirectory: "",
    externalId: "",
    externalWebhookId: "",
    webhookSecretToken: "",
    databaseGroup: "",
  };
}

export const VCSConnector = {
  encode(message: VCSConnector, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.title !== "") {
      writer.uint32(10).string(message.title);
    }
    if (message.fullPath !== "") {
      writer.uint32(18).string(message.fullPath);
    }
    if (message.webUrl !== "") {
      writer.uint32(26).string(message.webUrl);
    }
    if (message.branch !== "") {
      writer.uint32(34).string(message.branch);
    }
    if (message.baseDirectory !== "") {
      writer.uint32(42).string(message.baseDirectory);
    }
    if (message.externalId !== "") {
      writer.uint32(50).string(message.externalId);
    }
    if (message.externalWebhookId !== "") {
      writer.uint32(58).string(message.externalWebhookId);
    }
    if (message.webhookSecretToken !== "") {
      writer.uint32(66).string(message.webhookSecretToken);
    }
    if (message.databaseGroup !== "") {
      writer.uint32(74).string(message.databaseGroup);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): VCSConnector {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseVCSConnector();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.title = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.fullPath = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.webUrl = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.branch = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.baseDirectory = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.externalId = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.externalWebhookId = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.webhookSecretToken = reader.string();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.databaseGroup = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): VCSConnector {
    return {
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      fullPath: isSet(object.fullPath) ? globalThis.String(object.fullPath) : "",
      webUrl: isSet(object.webUrl) ? globalThis.String(object.webUrl) : "",
      branch: isSet(object.branch) ? globalThis.String(object.branch) : "",
      baseDirectory: isSet(object.baseDirectory) ? globalThis.String(object.baseDirectory) : "",
      externalId: isSet(object.externalId) ? globalThis.String(object.externalId) : "",
      externalWebhookId: isSet(object.externalWebhookId) ? globalThis.String(object.externalWebhookId) : "",
      webhookSecretToken: isSet(object.webhookSecretToken) ? globalThis.String(object.webhookSecretToken) : "",
      databaseGroup: isSet(object.databaseGroup) ? globalThis.String(object.databaseGroup) : "",
    };
  },

  toJSON(message: VCSConnector): unknown {
    const obj: any = {};
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.fullPath !== "") {
      obj.fullPath = message.fullPath;
    }
    if (message.webUrl !== "") {
      obj.webUrl = message.webUrl;
    }
    if (message.branch !== "") {
      obj.branch = message.branch;
    }
    if (message.baseDirectory !== "") {
      obj.baseDirectory = message.baseDirectory;
    }
    if (message.externalId !== "") {
      obj.externalId = message.externalId;
    }
    if (message.externalWebhookId !== "") {
      obj.externalWebhookId = message.externalWebhookId;
    }
    if (message.webhookSecretToken !== "") {
      obj.webhookSecretToken = message.webhookSecretToken;
    }
    if (message.databaseGroup !== "") {
      obj.databaseGroup = message.databaseGroup;
    }
    return obj;
  },

  create(base?: DeepPartial<VCSConnector>): VCSConnector {
    return VCSConnector.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<VCSConnector>): VCSConnector {
    const message = createBaseVCSConnector();
    message.title = object.title ?? "";
    message.fullPath = object.fullPath ?? "";
    message.webUrl = object.webUrl ?? "";
    message.branch = object.branch ?? "";
    message.baseDirectory = object.baseDirectory ?? "";
    message.externalId = object.externalId ?? "";
    message.externalWebhookId = object.externalWebhookId ?? "";
    message.webhookSecretToken = object.webhookSecretToken ?? "";
    message.databaseGroup = object.databaseGroup ?? "";
    return message;
  },
};

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
