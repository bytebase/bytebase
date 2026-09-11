import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSource_AuthenticationType as Auth,
  DataSource_AWSCredentialSchema,
  DataSource_AzureCredentialSchema,
  DataSourceSchema,
  DataSourceExternalSecret_SecretType as SecretType,
} from "@/types/proto-es/v1/instance_service_pb";
import type { EditDataSource } from "./common";
import { validateDataSource } from "./validation";

const draft = (patch: Partial<EditDataSource> = {}): EditDataSource => ({
  ...create(DataSourceSchema, { host: "db.example.com" }),
  pendingCreate: true,
  updatedPassword: "",
  updatedMasterPassword: "",
  updatedToken: "",
  ...patch,
});
const validate = (ds: EditDataSource, engine = Engine.MYSQL) =>
  validateDataSource(ds, { engine, isSaaSMode: true });

describe("instance validation", () => {
  test.each(["sslCaPath", "sslCertPath", "sslKeyPath"] as const)(
    "rejects relative %s",
    (field) => {
      const errors = validate(
        draft({
          useSsl: true,
          sslCertPath: "/cert.pem",
          sslKeyPath: "/key.pem",
          [field]: "relative.pem",
        })
      );
      expect(errors[field]).toBe("absolute-path");
    }
  );
  test("requires both TLS identity fields and ignores material when TLS is off", () => {
    const ds = draft({ useSsl: true, sslCertPath: "/cert.pem" });
    expect(validate(ds).sslKeyPath).toBe("tls-pair");
    expect(validate({ ...ds, useSsl: false })).toEqual({});
  });
  test("keeps stored TLS material until its group is replaced", () => {
    const ds = draft({
      pendingCreate: false,
      useSsl: true,
      sslCertSet: true,
      sslKeySet: true,
    });
    expect(validate(ds)).toEqual({});
    expect(
      validate({
        ...ds,
        sslCertPath: "/cert.pem",
        updateSsl: { clientCert: true },
      }).sslKeyPath
    ).toBe("tls-pair");
    expect(
      validate({
        ...ds,
        sslCertPath: "/cert.pem",
        sslKeyPath: "/key.pem",
        updateSsl: { clientCert: true },
      })
    ).toEqual({});
  });
  test("catches malformed PEM and conflicting sources", () => {
    expect(validate(draft({ useSsl: true, sslCa: "not PEM" })).sslCa).toBe(
      "pem-certificate"
    );
    expect(
      validate(draft({ useSsl: true, sslCa: "not PEM", sslCaPath: "/ca.pem" }))
        .sslCaPath
    ).toBe("tls-source-conflict");
  });
  test.each(["allowAllFiles", " ALLOWALLFILES "])(
    "rejects unsafe MySQL parameter %s",
    (key) => {
      const ds = draft({ extraConnectionParameters: { [key]: "false" } });
      expect(validate(ds)[`extraConnectionParameters.${key}`]).toBe(
        "forbidden-parameter"
      );
      expect(validate(ds, Engine.POSTGRES)).toEqual({});
    }
  );
  test("AWS region does not short circuit credential or TLS validation", () => {
    const ds = draft({
      authenticationType: Auth.AWS_RDS_IAM,
      region: "us-east-1",
      iamExtension: {
        case: "awsCredential",
        value: create(DataSource_AWSCredentialSchema),
      },
      useSsl: true,
      sslCaPath: "relative",
    });
    const errors = validate(ds);
    expect(errors["iamExtension.accessKeyId"]).toBe("aws-credential");
    expect(errors.sslCaPath).toBe("absolute-path");
  });
  test("Azure Key Vault requires its URL and secret name", () => {
    const ds = draft(
      create(DataSourceSchema, {
        host: "db.example.com",
        externalSecret: { secretType: SecretType.AZURE_KEY_VAULT },
      })
    );
    expect(validate(ds)).toMatchObject({
      "externalSecret.url": "required",
      "externalSecret.secretName": "required",
    });
  });
  test("stored redacted credentials pass, replacements require complete credentials", () => {
    const stored = create(DataSourceSchema, {
      host: "db.example.com",
      authenticationType: Auth.AZURE_IAM,
      iamExtension: {
        case: "azureCredential",
        value: { tenantId: "tenant", clientId: "client" },
      },
    });
    const ds = draft({ ...stored, pendingCreate: false });
    expect(
      validateDataSource(ds, {
        engine: Engine.POSTGRES,
        isSaaSMode: true,
        stored,
      })
    ).toEqual({});
    const replacement = draft({
      ...stored,
      pendingCreate: false,
      iamExtension: {
        case: "azureCredential",
        value: create(DataSource_AzureCredentialSchema, {
          tenantId: "other",
          clientId: "client",
        }),
      },
    });
    expect(
      validateDataSource(replacement, {
        engine: Engine.POSTGRES,
        isSaaSMode: true,
        stored,
      })["iamExtension.clientSecret"]
    ).toBe("required");
  });
  test("BigQuery validates credentials without requiring a Cloud SQL host", () => {
    const ds = draft(
      create(DataSourceSchema, {
        host: "",
        projectId: "valid-project",
        authenticationType: Auth.GOOGLE_CLOUD_SQL_IAM,
        iamExtension: { case: "gcpCredential", value: {} },
      })
    );
    const errors = validate(ds, Engine.BIGQUERY);
    expect(errors.host).toBeUndefined();
    expect(errors["iamExtension.content"]).toBe("required");
  });
  test("self-hosted DynamoDB can use the default credential chain without a region or host", () => {
    const ds = draft({ host: "", authenticationType: Auth.AWS_RDS_IAM });
    expect(
      validateDataSource(ds, { engine: Engine.DYNAMODB, isSaaSMode: false })
    ).toEqual({});
  });
});
