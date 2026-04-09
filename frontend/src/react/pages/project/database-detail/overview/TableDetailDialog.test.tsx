import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { DataClassificationSetting_DataClassificationConfig } from "@/types/proto-es/v1/setting_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
  })),
}));

let TableDetailDialog: typeof import("./TableDetailDialog").TableDetailDialog;

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/components/ui/dialog", () => ({
  Dialog: ({ open, children }: { open: boolean; children: React.ReactNode }) =>
    open ? <div data-testid="dialog-root">{children}</div> : null,
  DialogContent: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  DialogTitle: ({ children }: { children: React.ReactNode }) => (
    <h1>{children}</h1>
  ),
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) => (
    <input {...props} />
  ),
}));

const renderIntoContainer = (element: ReturnType<typeof createElement>) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

const flush = async () => {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
};

beforeEach(async () => {
  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
  });

  vi.resetModules();
  ({ TableDetailDialog } = await import("./TableDetailDialog"));
});

describe("TableDetailDialog", () => {
  test("restores the legacy table detail sections for columns and indexes", async () => {
    const classificationConfig = {
      id: "classification-config",
      levels: [{ level: 1, title: "L1" }],
      classification: {
        PII: {
          id: "PII",
          title: "PII",
          level: 1,
        },
      },
    } as unknown as DataClassificationSetting_DataClassificationConfig;

    const { container, render, unmount } = renderIntoContainer(
      createElement(TableDetailDialog, {
        open: true,
        onOpenChange: vi.fn(),
        table: {
          name: '"public"."audit"',
          classification: "PII",
          classificationConfig,
          columns: [
            {
              name: "id",
              semanticType: "",
              classification: "",
              type: "integer",
              defaultValue: "nextval('public.audit_id_seq'::regclass)",
              nullable: false,
              collation: "",
              comment: "",
            },
            {
              name: "query",
              semanticType: "SQL",
              classification: "PII",
              type: "text",
              defaultValue: "No default",
              nullable: true,
              collation: "en_US",
              comment: "query text",
            },
          ],
          rowCount: "0",
          dataSize: "8 KB",
          indexSize: "32 KB",
          collation: "en_US",
          indexes: [
            {
              name: "audit_pkey",
              expressions: ["id"],
              unique: true,
              visible: true,
              comment: "",
            },
          ],
          showColumnClassification: true,
          showColumnCollation: true,
          showCollation: true,
          showIndexComment: true,
          showIndexes: true,
          showIndexSize: true,
          showIndexVisible: false,
          showSemanticType: true,
        },
      })
    );

    render();
    await flush();

    expect(container.textContent).toContain("database.classification.self");
    expect(container.textContent).toContain("PII");
    expect(container.textContent).toContain(
      "settings.sensitive-data.semantic-types.table.semantic-type"
    );
    expect(container.textContent).toContain("common.default");
    expect(container.textContent).toContain("database.nullable");
    expect(container.textContent).toContain("db.collation");
    expect(container.textContent).toContain("database.indexes");
    expect(container.textContent).toContain("audit_pkey");
    expect(container.textContent).toContain("database.expression");
    expect(container.textContent).toContain("database.unique");
    expect(container.textContent).toContain(
      "nextval('public.audit_id_seq'::regclass)"
    );
    expect(container.textContent).toContain("query text");

    unmount();
  });
});
