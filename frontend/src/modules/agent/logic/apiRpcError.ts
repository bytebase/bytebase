import { fromJsonString, type JsonValue } from "@bufbuild/protobuf";
import { StructSchema } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";
import {
  codeFromString,
  codeToHttpStatus,
  errorFromJson,
} from "@connectrpc/connect/protocol-connect";

export function parseAgentRpcError(
  body: string,
  status?: number
): ConnectError | undefined {
  try {
    // Decode before JSON.parse can collapse a duplicate RPC code field.
    const codeField = fromJsonString(StructSchema, body).fields.code?.kind;
    if (codeField?.case !== "stringValue") return undefined;
    const code = codeFromString(codeField.value);
    if (
      code === undefined ||
      !Object.hasOwn(Code, code) ||
      (status !== undefined && codeToHttpStatus(code) !== status)
    )
      return undefined;
    const json = JSON.parse(body) as Record<string, JsonValue>;
    if (json.details !== undefined && !Array.isArray(json.details))
      return undefined;
    return errorFromJson(
      json,
      undefined,
      new ConnectError("Invalid RPC error", code)
    );
  } catch {
    return undefined;
  }
}
