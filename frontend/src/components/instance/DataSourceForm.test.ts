import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";

const source = () =>
  readFileSync(
    join(process.cwd(), "src/components/instance/DataSourceForm.tsx"),
    "utf-8"
  );

describe("DataSourceForm password sources", () => {
  test("keeps a draft per instance data source and secret provider", () => {
    const form = source();

    expect(form).toContain("sourceDraftStateRef");
    expect(form).toContain("invalidateSourceDrafts");
    expect(form).toContain("dataSourceResetEvent");
    expect(form).toContain("sourceDraftsRef.current.get(dataSource.id)");
    expect(form).toContain("drafts.set(");
    expect(form).toContain("const existingDraft = drafts.get(secretType)");
    expect(form).toContain("ds.externalSecret = existingDraft");
  });

  test("exports Redis Sentinel fields for the connection mode panel", () => {
    expect(source()).toContain("export function RedisSentinelFields");
  });

  test("does not initialize a provider from another provider's identifiers", () => {
    const form = source();

    expect(form).not.toContain("ds.externalSecret?.secretName ??");
    expect(form).not.toContain("ds.externalSecret?.passwordKeyName ??");
    expect(form).toContain('secretName: ""');
    expect(form).toContain('passwordKeyName: ""');
  });

  test("keeps the password source selector beside the Bytebase password input", () => {
    const form = source();
    const sourceIndex = form.indexOf("{passwordSourceControl}");
    const inputIndex = form.indexOf('type="password"', sourceIndex);

    expect(sourceIndex).toBeGreaterThan(0);
    expect(inputIndex).toBeGreaterThan(sourceIndex);
  });

  test("places the Bytebase password storage hint below the password title", () => {
    const form = source();
    const plainPasswordIndex = form.indexOf("/* Plain password */");
    const storedHintIndex = form.indexOf(
      "instance.password-source.stored-in-bytebase",
      plainPasswordIndex
    );
    const passwordInputIndex = form.indexOf(
      'type="password"',
      plainPasswordIndex
    );

    expect(plainPasswordIndex).toBeGreaterThan(0);
    expect(storedHintIndex).toBeGreaterThan(plainPasswordIndex);
    expect(storedHintIndex).toBeLessThan(passwordInputIndex);
    expect(form.slice(storedHintIndex - 100, storedHintIndex)).toContain(
      "description={"
    );
  });

  test("places external secret documentation below the source title", () => {
    const form = source();
    const externalSecretFieldIndex = form.indexOf(
      "/* External secret fields */"
    );
    const passwordSourceControlIndex = form.indexOf(
      "{passwordSourceControl}",
      externalSecretFieldIndex
    );
    const learnMoreIndex = form.indexOf(
      "<LearnMoreLink",
      externalSecretFieldIndex
    );

    expect(externalSecretFieldIndex).toBeGreaterThan(0);
    expect(learnMoreIndex).toBeGreaterThan(externalSecretFieldIndex);
    expect(learnMoreIndex).toBeLessThan(passwordSourceControlIndex);
    expect(form.slice(learnMoreIndex - 100, learnMoreIndex)).toContain(
      "description={"
    );
  });
});

describe("DataSourceForm connection information buttons", () => {
  test("uses named shared buttons for connection information panels", () => {
    const form = source();

    expect(form).not.toContain("<button");
    expect(form).toContain('aria-label={t("instance.authentication")}');
    expect(form).toContain(
      'aria-label={t("data-source.ssl.connection-security")}'
    );
    expect(form).toContain('aria-label={t("data-source.ssh-connection")}');
    expect(form).toContain('onOpenInfoPanel("authentication")');
    expect(form).toContain('onOpenInfoPanel("ssl")');
    expect(form).toContain('onOpenInfoPanel("ssh")');
    expect(form.match(/className="-ml-1 w-6 shrink-0 p-0"/g)).toHaveLength(3);
  });
});

describe("DataSourceForm extra parameters", () => {
  test("uses the same action width for add and remove rows", () => {
    const extraParameters = source().slice(
      source().indexOf("/* Extra connection parameters */")
    );

    expect(extraParameters.match(/className="w-24 shrink-0"/g)).toHaveLength(2);
  });
});

describe("DataSourceForm layout", () => {
  test("uses a 24px rhythm between connection fields", () => {
    expect(source()).toContain(
      'className="grid grid-cols-1 gap-y-6 gap-x-4 border-none sm:grid-cols-3"'
    );
  });
});
