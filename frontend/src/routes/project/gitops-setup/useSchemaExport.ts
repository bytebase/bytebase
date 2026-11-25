import { create } from "@bufbuild/protobuf";
import { ConnectError, createContextValues } from "@connectrpc/connect";
import JSZip from "jszip";
import { useEffect, useState } from "react";
import { databaseServiceClientConnect } from "@/api";
import { skipErrorNotificationContextKey } from "@/api/context-key";
import {
  GetDatabaseSDLSchemaRequest_SDLFormat,
  GetDatabaseSDLSchemaRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";

export interface SchemaExport {
  // The ZIP as the server returned it, which Download hands over unchanged.
  bytes: Uint8Array;
  zip: JSZip;
  paths: string[];
}

export type SchemaExportState =
  | { status: "loading" }
  | { status: "failed"; message: string }
  | { status: "ready"; result: SchemaExport };

// The multi-file schema export of one database. The page holds it, so the
// steps can leave and come back without exporting again.
export function useSchemaExport(
  databaseName: string | undefined
): SchemaExportState | undefined {
  const [held, setHeld] = useState<{
    databaseName: string;
    state: SchemaExportState;
  }>();

  useEffect(() => {
    if (!databaseName) return;
    const controller = new AbortController();
    const hold = (state: SchemaExportState) => {
      if (!controller.signal.aborted) setHeld({ databaseName, state });
    };
    void (async () => {
      try {
        const response =
          await databaseServiceClientConnect.getDatabaseSDLSchema(
            create(GetDatabaseSDLSchemaRequestSchema, {
              name: `${databaseName}/sdlSchema`,
              format: GetDatabaseSDLSchemaRequest_SDLFormat.MULTI_FILE,
            }),
            {
              signal: controller.signal,
              contextValues: createContextValues().set(
                skipErrorNotificationContextKey,
                true
              ),
            }
          );
        if (controller.signal.aborted) return;
        const zip = await JSZip.loadAsync(response.schema);
        const paths = Object.values(zip.files)
          .filter((file) => !file.dir)
          .map((file) => file.name)
          .sort((a, b) => a.localeCompare(b));
        hold({
          status: "ready",
          result: { bytes: response.schema, zip, paths },
        });
      } catch (error) {
        hold({
          status: "failed",
          message: ConnectError.from(error).rawMessage,
        });
      }
    })();
    return () => controller.abort();
  }, [databaseName]);

  if (!databaseName) return undefined;
  // What is held belongs to another database until this one's export lands.
  return held?.databaseName === databaseName
    ? held.state
    : { status: "loading" };
}
