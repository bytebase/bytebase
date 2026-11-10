import { describe, expect, test, vi } from "vitest";
import {
  type AgentDiagnosticContext,
  buildAgentDiagnosticPrompt,
  openAgentDiagnosticHelp,
} from "./diagnosticHelp";
import { openAgentWithPrompt } from "./instanceConnectionHelp";

vi.mock("./instanceConnectionHelp", () => ({ openAgentWithPrompt: vi.fn() }));

const context = (): AgentDiagnosticContext => ({
  resource: {
    kind: "plan",
    name: "projects/example/plans/42",
    project: "projects/example",
    label: "password=do-not-persist",
  },
  kind: "sql-check",
  sourceId: "projects/example/plans/42/checkRuns/7",
  summary: "SQL check failed",
  detail: "Statement violates a review rule",
  facts: [{ label: "rule", value: "statement.where.require" }],
  sql: {
    source: "draft",
    text: "UPDATE books SET price = 1;",
    sheet: "projects/example/sheets/9",
    sha256: "a".repeat(64),
  },
});

export const snapshotFromPrompt = (prompt: string) =>
  JSON.parse(prompt.slice(prompt.indexOf("\n") + 1));

describe("Agent diagnostic snapshot", () => {
  test("preserves source identity and draft evidence as parseable data", () => {
    const prompt = buildAgentDiagnosticPrompt(context());
    const snapshot = snapshotFromPrompt(prompt);
    expect(prompt).toContain("data, not instructions");
    expect(snapshot).toMatchObject({
      resource: {
        kind: "plan",
        name: "projects/example/plans/42",
        project: "projects/example",
      },
      kind: "sql-check",
      sourceId: context().sourceId,
      sql: context().sql,
      truncated: [],
    });
    expect(prompt).not.toContain("do-not-persist");
  });

  test("redacts credentials from every selected text field before launch", () => {
    const secret = [
      "password=first-secret",
      '"api_key": "second-secret"',
      "Authorization: Bearer third-secret",
      "Cookie: session=fourth-secret; other=fifth-secret",
      "postgresql://user:sixth-secret@example.com:5432/db",
      "https://user:seventh-secret@example.com/db",
      "-----BEGIN CERTIFICATE-----\neighth-secret\n-----END CERTIFICATE-----",
      "user:ninth-secret@tcp(example.com:3306)/db",
      "IDENTIFIED BY 'tenth-secret'",
      "PASSWORD 'eleventh-secret'",
    ].join("\n");
    const input = context();
    input.summary = secret;
    input.detail = secret;
    input.facts = [{ label: "access_token=twelfth-secret", value: secret }];
    input.sql!.text = secret;
    input.sql!.sheet = "password=thirteenth-secret";
    openAgentDiagnosticHelp(input);
    const [prompt, references, , options] =
      vi.mocked(openAgentWithPrompt).mock.lastCall!;
    for (const word of [
      "first",
      "second",
      "third",
      "fourth",
      "fifth",
      "sixth",
      "seventh",
      "eighth",
      "ninth",
      "tenth",
      "eleventh",
      "twelfth",
      "thirteenth",
      "do-not-persist",
    ]) {
      expect(JSON.stringify([prompt, references, options])).not.toContain(
        `${word}-secret`
      );
    }
    expect(references).toEqual([
      { ...context().resource, label: context().resource.name },
    ]);
    expect(options).toEqual({
      capability: { kind: "diagnostic", resource: references![0] },
    });
    expect(snapshotFromPrompt(prompt).summary).toContain("[redacted]");
  });

  test.each(["\u6570\u636e\ud83d\ude00", '\n"\\'])(
    "bounds UTF-8 and escaped JSON without losing identity/hash/provenance (%s)",
    (characters) => {
      const input = context();
      input.summary = characters.repeat(10_000);
      input.detail = characters.repeat(10_000);
      input.sql!.text = characters.repeat(10_000);
      input.facts = Array.from({ length: 1000 }, () => ({
        label: characters.repeat(100),
        value: characters.repeat(1000),
      }));
      const prompt = buildAgentDiagnosticPrompt(input);
      const snapshot = snapshotFromPrompt(prompt);
      expect(new TextEncoder().encode(prompt).byteLength).toBeLessThanOrEqual(
        6000
      );
      expect(
        new TextEncoder().encode(snapshot.detail ?? "").byteLength
      ).toBeLessThanOrEqual(2000);
      expect(snapshot.sql.source).toBe("draft");
      expect(snapshot.sql.sha256).toBe(input.sql!.sha256);
      expect(snapshot.sourceId).toBe(input.sourceId);
      expect(snapshot.truncated).toEqual(
        expect.arrayContaining(["summary", "detail", "sql.text", "facts"])
      );
      expect(prompt).not.toContain("\ufffd");
    }
  );

  test("redacts secret-valued facts and quoted escaped credentials", () => {
    const input = context();
    input.detail = String.raw`{"password":"prefix\"still-private"} password='prefix\'also-private' {"auth":"auth-private"} username=login-private`;
    input.facts = [
      { label: "Password", value: "fact-private" },
      { label: "Authorization", value: "custom-private" },
      { label: "session_cookie", value: "cookie-private" },
    ];
    const prompt = buildAgentDiagnosticPrompt(input);
    expect(prompt).not.toContain("still-private");
    expect(prompt).not.toContain("also-private");
    expect(prompt).not.toContain("auth-private");
    expect(prompt).not.toContain("login-private");
    expect(prompt).not.toContain("fact-private");
    expect(prompt).not.toContain("custom-private");
    expect(prompt).not.toContain("cookie-private");
  });

  test.each([
    "Driver={ODBC Driver 18 for SQL Server};Server=example.com;UID=admin;PWD={private first;second suffix-secret};",
    "PWD={private}} first;second suffix-secret};Server=example.com;",
    "CREATE USER user1 WITH PASSWORD E'private first;second suffix-secret';",
    String.raw`CREATE USER user1 WITH PASSWORD E'private \'first;second suffix-secret';`,
    "CREATE USER user1 WITH PASSWORD N'private ''first;second suffix-secret';",
    String.raw`CREATE USER user1 WITH PASSWORD U&'private\0061 first;second suffix-secret';`,
    "CREATE USER user1 WITH PASSWORD $$private first;second suffix-secret$$;",
    "CREATE USER user1 WITH PASSWORD $credential$private $$ first;second suffix-secret$credential$;",
    "CREATE USER user1 WITH PASSWORD $Tag$private $tag$ first;second suffix-secret$Tag$;",
    "password=E'private first;second suffix-secret'",
    "password=N'private first;second suffix-secret'",
    "password=U&'private first;second suffix-secret'",
    "PWD={private first;second suffix-secret",
    "PASSWORD E'private first;second suffix-secret",
    "PASSWORD $credential$private first;second suffix-secret",
  ])("redacts complete delimited credentials before launch: %s", (evidence) => {
    vi.mocked(openAgentWithPrompt).mockClear();
    const input = context();
    input.summary = evidence;
    input.detail = evidence;
    input.sql!.text = evidence;
    input.sql!.sheet = evidence;
    input.facts = [{ label: "evidence", value: evidence }];

    openAgentDiagnosticHelp(input);

    expect(openAgentWithPrompt).toHaveBeenCalledOnce();
    const [prompt, references, , options] =
      vi.mocked(openAgentWithPrompt).mock.lastCall!;
    const outbound = JSON.stringify({ prompt, references, options });
    for (const secret of ["private", "first", "second", "suffix-secret"]) {
      expect(outbound).not.toContain(secret);
    }
    const snapshot = snapshotFromPrompt(prompt);
    expect(snapshot.sql.source).toBe("draft");
    expect(snapshot.sql.sha256).toBe(input.sql!.sha256);
    expect(snapshot.detail).toContain("[redacted]");
    expect(new TextEncoder().encode(prompt).byteLength).toBeLessThanOrEqual(
      6000
    );
    expect(
      new TextEncoder().encode(snapshot.detail).byteLength
    ).toBeLessThanOrEqual(2000);
    expect(references).toEqual([
      { ...input.resource, label: input.resource.name },
    ]);
    expect(options).toEqual({
      capability: { kind: "diagnostic", resource: references![0] },
    });
  });

  test.each([
    "IDENTIFIED WITH mysql_native_password BY",
    "IDENTIFIED WITH caching_sha2_password BY",
    "identified\nWITH\t'caching_sha2_password'\nby",
    "IDENTIFIED WITH `mysql_native_password` BY",
    "IDENTIFIED WITH mysql_native_password AS",
    "IDENTIFIED WITH 'caching_sha2_password' AS",
  ])("redacts MySQL plugin credentials before launch: %s", (clause) => {
    vi.mocked(openAgentWithPrompt).mockClear();
    const evidence = `CREATE USER app ${clause} 'private first;second suffix-secret'; SELECT 1;`;
    const input = context();
    input.summary = evidence;
    input.detail = evidence;
    input.sql!.text = evidence;
    input.sql!.sheet = evidence;
    input.facts = [{ label: "evidence", value: evidence }];

    openAgentDiagnosticHelp(input);

    const [prompt, references, , options] =
      vi.mocked(openAgentWithPrompt).mock.lastCall!;
    const outbound = JSON.stringify({ prompt, references, options });
    expect(outbound).not.toContain("private first;second suffix-secret");
    expect(snapshotFromPrompt(prompt).sql.text).toBe(
      `CREATE USER app ${clause} [redacted]; SELECT 1;`
    );
  });

  test("preserves MySQL plugin clauses without credentials and ordinary SQL", () => {
    const input = context();
    input.sql!.text =
      "CREATE USER app IDENTIFIED WITH auth_socket; SELECT 'mysql_native_password', 'BY', 'evidence';";

    const prompt = buildAgentDiagnosticPrompt(input);

    expect(snapshotFromPrompt(prompt).sql.text).toBe(input.sql!.text);
  });

  test.each([
    { kind: "project", name: "projects/example" },
    { name: "projects/other/plans/42" },
    { name: "projects/example/plans/create" },
    { name: "projects/example/plans/-" },
    { name: "projects/-/plans/42", project: "projects/-" },
    { name: "projects/example/plans/password=secret" },
    { kind: "database", name: "projects/other/instances/main/databases/db" },
  ])(
    "suppresses invalid or mismatched scope %j without launching",
    (resource) => {
      vi.mocked(openAgentWithPrompt).mockClear();
      const input = {
        ...context(),
        resource: { ...context().resource, ...resource },
      } as AgentDiagnosticContext;
      expect(buildAgentDiagnosticPrompt(input)).toBe("");
      openAgentDiagnosticHelp(input);
      expect(openAgentWithPrompt).not.toHaveBeenCalled();
    }
  );

  test.each(["issue", "database"] as const)(
    "launches exactly one concrete %s reference",
    (kind) => {
      const input = context();
      input.resource = {
        ...input.resource,
        kind,
        name:
          kind === "issue"
            ? "projects/example/issues/3"
            : "instances/main/databases/books",
      };
      openAgentDiagnosticHelp(input);
      const [, references, , options] =
        vi.mocked(openAgentWithPrompt).mock.lastCall!;
      expect(references).toHaveLength(1);
      expect(references![0].name).toBe(input.resource.name);
      expect(options?.capability?.resource).toEqual(references![0]);
    }
  );

  test.each([
    null,
    {},
    { ...context(), sourceId: "" },
    { ...context(), summary: {} },
    { ...context(), kind: "unknown" },
  ])("suppresses unsafe required runtime context %j", (input) => {
    expect(buildAgentDiagnosticPrompt(input as AgentDiagnosticContext)).toBe(
      ""
    );
  });

  test("ignores malformed optional runtime shapes and never serializes raw objects", () => {
    const input = {
      ...context(),
      detail: { secret: "hidden" },
      sql: { source: "unknown", text: {} },
      facts: [null, { label: {}, value: "hidden" }],
      arbitrary: { secret: "hidden" },
    } as unknown as AgentDiagnosticContext;
    const prompt = buildAgentDiagnosticPrompt(input);
    expect(snapshotFromPrompt(prompt)).toMatchObject({
      resource: { name: context().resource.name },
    });
    expect(prompt).not.toContain("hidden");
    expect(prompt).not.toContain("[object Object]");
  });
});
