import { describe, expect, test, vi } from "vitest";
import {
  type AgentDiagnosticContext,
  buildAgentDiagnosticPrompt,
  openAgentDiagnosticHelp,
} from "./diagnosticHelp";
import { openAgentWithPrompt } from "./instanceConnectionHelp";
import { type AgentSqlFixSource, registerAgentSqlFixTarget } from "./sqlFix";

const sqlFixMocks = vi.hoisted(() => ({
  setSource: vi.fn(),
  workspace: "workspaces/example",
  user: "users/member@example.com",
}));
vi.mock("../store/agent", () => ({
  useAgentStore: {
    getState: () => ({ setSqlFixSource: sqlFixMocks.setSource }),
  },
}));
vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      workspaceResourceName: () => sqlFixMocks.workspace,
      currentUser: { name: sqlFixMocks.user },
    }),
  },
}));

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
  test("marks the complete replacement separately from explanatory snippets", () => {
    expect(buildAgentDiagnosticPrompt(context())).toContain("sql replacement");
  });

  test.each([
    "user-resolve",
    "user-reject",
    "workspace-resolve",
    "workspace-reject",
  ])(
    "does not queue old evidence after an async %s scope change",
    async (scenario) => {
      const input = context();
      input.sqlFixSource = { sha256: "a".repeat(64) };
      let resolve!: (value: undefined) => void;
      let reject!: (reason: Error) => void;
      const unregister = registerAgentSqlFixTarget({
        capture: () =>
          new Promise<undefined>((done, fail) => {
            resolve = done;
            reject = fail;
          }),
        reveal: () => {},
        inspect: () => ({ status: "unavailable" }),
        apply: () => false,
      });
      const before = vi.mocked(openAgentWithPrompt).mock.calls.length;
      try {
        const pending = openAgentDiagnosticHelp(input);
        if (scenario.startsWith("user"))
          sqlFixMocks.user = "users/other@example.com";
        else sqlFixMocks.workspace = "workspaces/other";
        if (scenario.endsWith("reject"))
          reject(new Error("old request failed"));
        else resolve(undefined);
        await pending;
        expect(vi.mocked(openAgentWithPrompt).mock.calls.length).toBe(before);
      } finally {
        sqlFixMocks.user = "users/member@example.com";
        sqlFixMocks.workspace = "workspaces/example";
        unregister();
      }
    }
  );

  const sqlFixSource = (input = context()): AgentSqlFixSource => ({
    identity: JSON.stringify([
      "workspaces/example",
      "users/member@example.com",
    ]),
    pageKey: "/projects/example/plans/42",
    resource: input.resource,
    sourceId: input.sourceId,
    plan: "projects/example/plans/42",
    specId: "spec-1",
    sheet: "projects/example/sheets/9",
    targets: ["instances/prod/databases/example"],
    sha256: "a".repeat(64),
    sql: input.sql!.text,
  });

  test("registers the full trusted source before queueing its diagnostic run", () => {
    const input = context();
    input.sqlFixSource = { specId: "spec-1", sha256: "a".repeat(64) };
    const source = sqlFixSource(input);
    const unregister = registerAgentSqlFixTarget({
      capture: () => source,
      reveal: vi.fn(),
      inspect: () => ({ status: "ready", currentSql: source.sql }),
      apply: () => true,
    });
    try {
      openAgentDiagnosticHelp(input);
      const [prompt, , onChatCreated, options] =
        vi.mocked(openAgentWithPrompt).mock.lastCall!;
      expect(snapshotFromPrompt(prompt).sql.text).toBe(source.sql);
      expect(prompt).toContain("COMPLETE replacement");
      expect(options?.capability).toEqual({
        kind: "diagnostic",
        resource: { ...input.resource, label: input.resource.name },
      });
      expect(options).not.toHaveProperty("sqlFixSource");
      expect(onChatCreated).toBeTypeOf("function");
      onChatCreated!("trusted-chat");
      expect(sqlFixMocks.setSource).toHaveBeenLastCalledWith(
        "trusted-chat",
        source
      );
    } finally {
      unregister();
    }
  });

  test.each([
    { label: "SQL source", value: "Unavailable for this result" },
    {
      label: "Evidence",
      value:
        "Partial: original SQL snapshot unavailable; use the original revision identifiers, not current SQL.",
    },
  ])(
    "replaces obsolete unavailable-source evidence after capture: $label",
    (fact) => {
      const input = context();
      input.sqlFixSource = { sha256: "a".repeat(64) };
      input.sql = undefined;
      input.facts = [fact, { label: "Rule", value: "syntax" }];
      const source = { ...sqlFixSource(), sourceId: input.sourceId };
      const unregister = registerAgentSqlFixTarget({
        capture: () => source,
        reveal: vi.fn(),
        inspect: () => ({ status: "ready" }),
        apply: () => true,
      });
      try {
        openAgentDiagnosticHelp(input);
        const [prompt] = vi.mocked(openAgentWithPrompt).mock.lastCall!;
        const snapshot = snapshotFromPrompt(prompt);
        expect(snapshot.sql.text).toBe(source.sql);
        expect(snapshot.facts).toContainEqual({
          label: "Rule",
          value: "syntax",
        });
        expect(snapshot.facts).not.toContainEqual(fact);
        expect(snapshot.facts).toContainEqual({
          label: fact.label,
          value: "Exact checked SQL snapshot included",
        });
      } finally {
        unregister();
      }
    }
  );

  test.each(["redacted", "bounded-prompt", "oversized"])(
    "keeps %s SQL diagnosis-only",
    (reason) => {
      const input = context();
      input.sqlFixSource = { sha256: "a".repeat(64) };
      const source = sqlFixSource(input);
      if (reason === "redacted")
        source.sql = "ALTER USER app PASSWORD 'not-to-persist';";
      if (reason === "bounded-prompt") {
        input.detail = '"'.repeat(2000);
        source.sql = "SELECT 1;".repeat(200);
      }
      if (reason === "oversized") source.sql = "SELECT 1;".repeat(300);
      const unregister = registerAgentSqlFixTarget({
        capture: () => source,
        reveal: vi.fn(),
        inspect: () => ({ status: "ready" }),
        apply: () => true,
      });
      try {
        openAgentDiagnosticHelp(input);
        const [prompt, , onChatCreated] =
          vi.mocked(openAgentWithPrompt).mock.lastCall!;
        expect(prompt).not.toContain("not-to-persist");
        expect(onChatCreated).toBeUndefined();
      } finally {
        unregister();
      }
    }
  );

  test("awaits source capture before launching and safely falls back after failure", async () => {
    const input = context();
    input.sqlFixSource = { sha256: "a".repeat(64) };
    const source = sqlFixSource(input);
    let resolve: (value: AgentSqlFixSource) => void = () => {};
    const unregister = registerAgentSqlFixTarget({
      capture: () =>
        new Promise<AgentSqlFixSource>((done) => {
          resolve = done;
        }),
      reveal: vi.fn(),
      inspect: () => ({ status: "ready" }),
      apply: () => true,
    });
    const before = vi.mocked(openAgentWithPrompt).mock.calls.length;
    const pending = openAgentDiagnosticHelp(input);
    expect(vi.mocked(openAgentWithPrompt).mock.calls.length).toBe(before);
    resolve(source);
    await pending;
    expect(vi.mocked(openAgentWithPrompt).mock.calls.length).toBe(before + 1);
    unregister();
    const unregisterFailure = registerAgentSqlFixTarget({
      capture: () => Promise.reject(new Error("SQL unavailable")),
      reveal: vi.fn(),
      inspect: () => ({ status: "unavailable" }),
      apply: () => false,
    });
    try {
      await expect(openAgentDiagnosticHelp(input)).resolves.toBeUndefined();
      expect(vi.mocked(openAgentWithPrompt).mock.lastCall![2]).toBeUndefined();
    } finally {
      unregisterFailure();
    }
  });

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
    [
      "ALTER USER app IDENTIFIED BY 'new-example-password' REPLACE 'old-example-password'; SELECT 1;",
      "ALTER USER app IDENTIFIED BY [redacted] REPLACE [redacted]; SELECT 1;",
    ],
    [
      "ALTER USER app IDENTIFIED WITH caching_sha2_password BY 'new-example-password'\nREPLACE 'old-example-password'; SELECT 1;",
      "ALTER USER app IDENTIFIED WITH caching_sha2_password BY [redacted]\nREPLACE [redacted]; SELECT 1;",
    ],
    [
      "ALTER USER app IDENTIFIED BY RANDOM PASSWORD REPLACE 'old-example-password'; SELECT 1;",
      "ALTER USER app IDENTIFIED BY RANDOM PASSWORD REPLACE [redacted]; SELECT 1;",
    ],
    [
      "ALTER USER app IDENTIFIED WITH caching_sha2_password BY RANDOM PASSWORD REPLACE 'old-example-password'; SELECT 1;",
      "ALTER USER app IDENTIFIED WITH caching_sha2_password BY RANDOM PASSWORD REPLACE [redacted]; SELECT 1;",
    ],
    [
      'ALTER USER app IDENTIFIED BY "new-example-password" REPLACE "old-example-password"; SELECT 1;',
      "ALTER USER app IDENTIFIED BY [redacted] REPLACE [redacted]; SELECT 1;",
    ],
    [
      "SET PASSWORD FOR 'app'@'localhost' = 'new-example-password' REPLACE 'old-example-password'; SELECT 1;",
      "SET PASSWORD FOR 'app'@'localhost' = [redacted] REPLACE [redacted]; SELECT 1;",
    ],
    [
      "CREATE USER app IDENTIFIED VIA mysql_native_password USING PASSWORD('old-example-password'); SELECT 1;",
      "CREATE USER app IDENTIFIED VIA mysql_native_password USING PASSWORD([redacted]); SELECT 1;",
    ],
    [
      "CREATE USER app IDENTIFIED WITH 'ed25519' AS PASSWORD ('old-example-password'); SELECT 1;",
      "CREATE USER app IDENTIFIED WITH 'ed25519' AS PASSWORD ([redacted]); SELECT 1;",
    ],
    [
      "CREATE USER app IDENTIFIED BY PASSWORD '*0123456789ABCDEF0123456789ABCDEF01234567'; SELECT 1;",
      "CREATE USER app IDENTIFIED BY PASSWORD [redacted]; SELECT 1;",
    ],
    [
      "CREATE USER app IDENTIFIED VIA mysql_native_password USING '*0123456789ABCDEF0123456789ABCDEF01234567' OR ed25519 USING PASSWORD('old-example-password'); SELECT 1;",
      "CREATE USER app IDENTIFIED VIA mysql_native_password USING [redacted] OR ed25519 USING PASSWORD([redacted]); SELECT 1;",
    ],
    [
      "CREATE USER app IDENTIFIED VIA mysql_native_password USING '*0123456789ABCDEF0123456789ABCDEF01234567' OR ed25519 USING 'old-example-password'; SELECT 1;",
      "CREATE USER app IDENTIFIED VIA mysql_native_password USING [redacted] OR ed25519 USING [redacted]; SELECT 1;",
    ],
    [
      "CREATE USER app IDENTIFIED VIA mysql_native_password USING PASSWORD('new-example-password') OR unix_socket OR ed25519 AS 'old-example-password'; SELECT 1;",
      "CREATE USER app IDENTIFIED VIA mysql_native_password USING PASSWORD([redacted]) OR unix_socket OR ed25519 AS [redacted]; SELECT 1;",
    ],
    [
      "CREATE USER app IDENTIFIED VIA unix_socket OR ed25519 USING 'old-example-password'; SELECT 1;",
      "CREATE USER app IDENTIFIED VIA unix_socket OR ed25519 USING [redacted]; SELECT 1;",
    ],
  ])(
    "redacts complete account authentication credentials before launch: %s",
    (evidence, expected) => {
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
      expect(outbound).not.toContain("new-example-password");
      expect(outbound).not.toContain("old-example-password");
      expect(outbound).not.toContain("0123456789ABCDEF");
      expect(snapshotFromPrompt(prompt)).toMatchObject({
        summary: expected,
        detail: expected,
        sql: {
          source: "draft",
          text: expected,
          sheet: expected,
          sha256: input.sql!.sha256,
        },
        facts: [{ label: "evidence", value: expected }],
        truncated: [],
      });
    }
  );

  test.each([
    "CREATE TABLE accounts (password TEXT NOT NULL);",
    "ALTER TABLE accounts ADD password VARCHAR(255);",
    "SELECT password FROM accounts;",
    "SELECT password AS current_password FROM accounts;",
    "SELECT REPLACE('before', 'old', 'new') FROM accounts;",
    "CREATE USER app IDENTIFIED VIA unix_socket;",
    "ALTER USER app IDENTIFIED BY RANDOM PASSWORD;",
  ])(
    "preserves noncredential SQL involving password identifiers: %s",
    (sql) => {
      const input = context();
      input.sql!.text = sql;
      const snapshot = snapshotFromPrompt(buildAgentDiagnosticPrompt(input));
      expect(snapshot.sql.text).toBe(sql);
      expect(snapshot.sql.sha256).toBe(input.sql!.sha256);
      expect(snapshot.truncated).toEqual([]);
    }
  );

  test.each([
    ["SET PASSWORD FOR 'app'@'localhost' =", "'example-secret'"],
    [
      'set\npassword\tfor\n"app" @ "localhost" =',
      '"private first;second suffix-secret"',
    ],
    [
      "SET PASSWORD FOR `app``name`@`local``host`=",
      "'private first;second suffix-secret'",
    ],
    [
      "SET PASSWORD FOR app@localhost =",
      "'private first;second suffix-secret'",
    ],
    [
      "SET PASSWORD FOR 'app = login'@'localhost' =",
      "'private first;second suffix-secret'",
    ],
    [
      "SET PASSWORD FOR 'app''name'@'local''host' =",
      "'private ''first;second suffix-secret'",
    ],
    [
      String.raw`SET PASSWORD FOR 'app\'name'@'localhost' =`,
      String.raw`'private \'first;second suffix-secret'`,
    ],
    [
      "SET PASSWORD FOR CURRENT_USER() =",
      "'private first;second suffix-secret'",
    ],
    [
      "SET PASSWORD FOR CURRENT_USER () =",
      "'private first;second suffix-secret'",
    ],
    [
      "SET PASSWORD FOR CURRENT_USER( ) =",
      "'private first;second suffix-secret'",
    ],
    ["SET PASSWORD =", "'private first;second suffix-secret'"],
    [
      "SET PASSWORD FOR 'app'@'localhost' = PASSWORD(",
      "'private first;second suffix-secret')",
    ],
    ["SET PASSWORD = PASSWORD (", "'private first;second suffix-secret')"],
  ])("redacts SET PASSWORD values before launch: %s", (clause, credential) => {
    vi.mocked(openAgentWithPrompt).mockClear();
    const evidence = `${clause} ${credential}; SELECT 1;`;
    const input = context();
    input.summary = evidence;
    input.detail = evidence;
    input.sql!.text = evidence;
    input.sql!.sheet = evidence;
    input.facts = [
      { label: "evidence", value: evidence },
      { label: evidence, value: "visible" },
    ];

    openAgentDiagnosticHelp(input);

    const [prompt, references, , options] =
      vi.mocked(openAgentWithPrompt).mock.lastCall!;
    const outbound = JSON.stringify({ prompt, references, options });
    for (const secret of [
      "example-secret",
      "private",
      "first",
      "second",
      "suffix-secret",
    ]) {
      expect(outbound).not.toContain(secret);
    }
    const safeEvidence = `${clause} [redacted]${credential.endsWith(")") ? ")" : ""}; SELECT 1;`;
    expect(snapshotFromPrompt(prompt)).toMatchObject({
      summary: safeEvidence,
      detail: safeEvidence,
      sql: { text: safeEvidence, sheet: safeEvidence },
      facts: [
        { label: "evidence", value: safeEvidence },
        { label: safeEvidence, value: "[redacted]" },
      ],
    });
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
