import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  DataSourceExternalSecretSchema,
  DataSourceSchema,
  DataSourceExternalSecret_SecretType as SecretType,
} from "@/types/proto-es/v1/instance_service_pb";
import type { EditDataSource } from "./common";
import {
  deactivateExternalSecret,
  invalidateSourceDrafts,
  type SourceDraftState,
} from "./data-source-drafts";

const initialState: SourceDraftState = {
  instanceName: "instances/production",
  resetEvent: 0,
};

describe("invalidateSourceDrafts", () => {
  test("keeps drafts while switching providers in the same edit", () => {
    const drafts = new Map([["admin", new Map([[1, "vault token"]])]]);

    const next = invalidateSourceDrafts(drafts, initialState, initialState);

    expect(next).toEqual(initialState);
    expect(drafts.get("admin")?.get(1)).toBe("vault token");
  });

  test("discards abandoned provider credentials after a form reset", () => {
    const drafts = new Map([["admin", new Map([[1, "vault token"]])]]);

    const next = invalidateSourceDrafts(drafts, initialState, {
      ...initialState,
      resetEvent: 1,
    });

    expect(next.resetEvent).toBe(1);
    expect(drafts).toHaveLength(0);
  });
});

describe("deactivateExternalSecret", () => {
  test.each(["", "database/password"])(
    "keeps the provider draft out of the active data source (%s)",
    (secretName) => {
      const secret = create(DataSourceExternalSecretSchema, {
        secretType: SecretType.VAULT_KV_V2,
        secretName,
      });
      const source: EditDataSource = {
        ...create(DataSourceSchema, { id: "admin", externalSecret: secret }),
        pendingCreate: true,
        updatedPassword: "",
        updatedMasterPassword: "",
        updatedToken: "",
      };
      const drafts = new Map<string, Map<number, typeof secret>>();
      const inactive = deactivateExternalSecret(source, drafts);

      expect(inactive.externalSecret).toBeUndefined();
      expect(source.externalSecret).toBe(secret);
      expect(drafts.get("admin")?.get(SecretType.VAULT_KV_V2)).toBe(secret);
      expect(
        deactivateExternalSecret(inactive, drafts).externalSecret
      ).toBeUndefined();
      expect(drafts.get("admin")?.get(SecretType.VAULT_KV_V2)).toBe(secret);
    }
  );
});
