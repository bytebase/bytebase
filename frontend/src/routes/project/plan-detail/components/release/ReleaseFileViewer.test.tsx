import { create } from "@bufbuild/protobuf";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeAll, beforeEach, describe, expect, test, vi } from "vitest";
import i18n from "@/lib/i18n";
import { VCSType } from "@/types/proto-es/v1/common_pb";
import { Release_VCSSourceSchema } from "@/types/proto-es/v1/release_service_pb";
import {
  ReleaseFileViewer,
  ReleaseFileViewerProvider,
  type ReleaseFileViewerTarget,
  useOpenReleaseFile,
} from "./ReleaseFileViewer";

const mocks = vi.hoisted(() => ({
  fetchSheet: vi.fn(),
  editorProps: [] as Array<{ content: string; advices?: unknown[] }>,
  // What the editor drew: as in MonacoEditor, only advices that change after
  // the editor is ready.
  drawn: [] as unknown[][],
  revealed: [] as number[],
}));

vi.mock("@/stores/app", () => {
  const state = () => ({ fetchSheet: mocks.fetchSheet, notify: vi.fn() });
  const useAppStore = (selector?: (s: unknown) => unknown) =>
    selector ? selector(state()) : state();
  useAppStore.getState = state;
  return { useAppStore };
});

vi.mock("@/components/monaco", async () => {
  const React = await vi.importActual<typeof import("react")>("react");
  return {
    // Reports ready after mount, as the real editor does once Monaco loads.
    ReadonlyMonaco: (props: {
      content: string;
      advices?: unknown[];
      onReady?: (monaco: unknown, editor: unknown) => void;
    }) => {
      mocks.editorProps.push(props);
      const { onReady, advices } = props;
      const ready = React.useRef(false);
      React.useEffect(() => {
        if (ready.current) mocks.drawn.push(advices ?? []);
      }, [advices]);
      React.useEffect(() => {
        ready.current = true;
        onReady?.({}, { revealLineInCenter: (line: number) => mocks.revealed.push(line) });
        // biome-ignore lint/correctness/useExhaustiveDependencies: once per mount, as the editor loads once.
      }, []);
      return <pre data-testid="editor">{props.content}</pre>;
    },
  };
});

const STATEMENT = "SELECT 'é';\nALTER TABLE t ADD c INT;";
const sheetOf = (content: string) => {
  const bytes = new TextEncoder().encode(content);
  return { content: bytes, contentSize: BigInt(bytes.byteLength) };
};

const setViewport = (width: number) => {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: (query: string) => {
      const min = /min-width:\s*(\d+)px/.exec(query);
      const max = /max-width:\s*(\d+)px/.exec(query);
      const matches =
        (!min || width >= Number(min[1])) && (!max || width <= Number(max[1]));
      return {
        matches,
        media: query,
        addEventListener: () => {},
        removeEventListener: () => {},
      } as unknown as MediaQueryList;
    },
  });
};

const github = create(Release_VCSSourceSchema, {
  vcsType: VCSType.GITHUB,
  url: "https://github.com/example/app/commit/0a1b2c3d4e5f",
  commitSha: "0a1b2c3d4e5f",
});

const target = (overrides: Partial<ReleaseFileViewerTarget> = {}): ReleaseFileViewerTarget => ({
  file: { version: "002", path: "migrations/002_orders.sql", sheet: "projects/p/sheets/s2" },
  ...overrides,
});

const flush = async () => {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
};

beforeAll(async () => {
  await i18n.changeLanguage("en-US");
});

beforeEach(() => {
  mocks.editorProps.length = 0;
  mocks.drawn.length = 0;
  mocks.revealed.length = 0;
  mocks.fetchSheet.mockResolvedValue(sheetOf(STATEMENT));
  setViewport(1440);
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("ReleaseFileViewer", () => {
  test("opened from a failed section, it draws the error on the failed statement from its byte range", async () => {
    // "SELECT 'é';\n" is 13 bytes: é takes two.
    render(
      <ReleaseFileViewer
        target={target({
          context: { kind: "failed", error: "ERROR: column \"c\" already exists", range: { start: 13, end: 37 } },
          stageTitle: "Staging",
        })}
        vcsSource={github}
        onClose={() => {}}
      />
    );
    await flush();
    expect(screen.getByText("Failed on Staging")).toBeDefined();
    expect(mocks.editorProps.at(-1)?.advices).toEqual([
      {
        severity: "ERROR",
        message: 'ERROR: column "c" already exists',
        startLineNumber: 2,
        startColumn: 1,
        endLineNumber: 2,
        endColumn: 25,
      },
    ]);
  });

  test("with no command range it opens at the top with the error above the editor", async () => {
    render(
      <ReleaseFileViewer
        target={target({ context: { kind: "failed", error: "ORA-00942: table or view does not exist" } })}
        vcsSource={github}
        onClose={() => {}}
      />
    );
    await flush();
    expect(screen.getByTestId("release-file-error").textContent).toBe(
      "ORA-00942: table or view does not exist"
    );
    expect(mocks.editorProps.at(-1)?.advices).toEqual([]);
  });

  test("says what this database did with the version, and nothing when opened from Changes", async () => {
    const { rerender } = render(
      <ReleaseFileViewer target={target({ context: { kind: "applied" } })} onClose={() => {}} />
    );
    await flush();
    expect(screen.getByTestId("release-file-database-line").textContent).toBe("Applied by this run");
    rerender(
      <ReleaseFileViewer target={target({ context: { kind: "already-applied", run: true } })} onClose={() => {}} />
    );
    expect(screen.getByTestId("release-file-database-line").textContent).toBe(
      "Already applied; this run passed over it"
    );
    rerender(<ReleaseFileViewer target={target()} onClose={() => {}} />);
    expect(screen.queryByTestId("release-file-database-line")).toBeNull();
  });

  test.each([
    [VCSType.GITHUB, "https://github.com/example/app/commit/0a1b2c3d4e5f", "Open in GitHub at 0a1b2c3", "https://github.com/example/app/blob/0a1b2c3d4e5f/migrations/002_orders.sql"],
    [VCSType.GITLAB, "https://gitlab.example.com/group/app/-/commit/0a1b2c3d4e5f", "Open in GitLab at 0a1b2c3", "https://gitlab.example.com/group/app/-/blob/0a1b2c3d4e5f/migrations/002_orders.sql"],
    [VCSType.BITBUCKET, "https://bitbucket.org/example/app/commits/0a1b2c3d4e5f", "Open in Bitbucket at 0a1b2c3", "https://bitbucket.org/example/app/src/0a1b2c3d4e5f/migrations/002_orders.sql"],
    [VCSType.AZURE_DEVOPS, "https://dev.azure.com/example/shop/_git/app/commit/0a1b2c3d4e5f", "Open in Azure DevOps at 0a1b2c3", "https://dev.azure.com/example/shop/_git/app?path=/migrations/002_orders.sql&version=GC0a1b2c3d4e5f"],
  ])("links to the file at the commit on platform %s", async (vcsType, url, label, href) => {
    render(
      <ReleaseFileViewer
        target={target()}
        vcsSource={create(Release_VCSSourceSchema, { vcsType, url })}
        onClose={() => {}}
      />
    );
    await flush();
    const link = screen.getByRole("link", { name: label });
    expect(link.getAttribute("href")).toBe(href);
  });

  test("an unknown platform gets no link", async () => {
    render(
      <ReleaseFileViewer
        target={target()}
        vcsSource={create(Release_VCSSourceSchema, { url: "https://git.example.com/app/commit/77aa" })}
        onClose={() => {}}
      />
    );
    await flush();
    expect(screen.queryByRole("link")).toBeNull();
  });

  test.each([
    [1440, "side-panel"],
    [1024, "sheet"],
    [390, "sheet"],
  ])("at %i px it opens as a %s", async (width, layout) => {
    setViewport(width);
    render(<ReleaseFileViewer target={target()} onClose={() => {}} />);
    await flush();
    expect(screen.getByTestId("release-file-viewer").getAttribute("data-layout")).toBe(layout);
  });

  test("another file swaps the side panel's content", async () => {
    const { rerender } = render(<ReleaseFileViewer target={target()} onClose={() => {}} />);
    await flush();
    rerender(
      <ReleaseFileViewer
        target={target({ file: { version: "003", path: "migrations/003_index.sql", sheet: "projects/p/sheets/s3" } })}
        onClose={() => {}}
      />
    );
    await flush();
    expect(screen.getAllByTestId("release-file-viewer")).toHaveLength(1);
    expect(screen.getByTestId("release-file-viewer").textContent).toContain("migrations/003_index.sql");
    expect(mocks.fetchSheet).toHaveBeenLastCalledWith("projects/p/sheets/s3", true);
  });

  test("Esc closes it", async () => {
    const onClose = vi.fn();
    render(<ReleaseFileViewer target={target()} onClose={onClose} />);
    await flush();
    fireEvent.keyDown(screen.getByTestId("release-file-viewer"), { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });

  test("before any run, the already-applied line names no run", async () => {
    render(<ReleaseFileViewer target={target({ context: { kind: "already-applied", run: false } })} onClose={() => {}} />);
    await flush();
    expect(screen.getByTestId("release-file-database-line").textContent).toBe(
      "Already applied before this rollout"
    );
  });

  test("a sheet that cannot be loaded says so instead of loading forever", async () => {
    mocks.fetchSheet.mockRejectedValue(new Error("permission denied"));
    render(<ReleaseFileViewer target={target()} onClose={() => {}} />);
    await flush();
    expect(screen.getByTestId("release-file-statement-state").textContent).toBe(
      "The file's statement could not be loaded."
    );
    const { unmount } = render(
      <ReleaseFileViewer
        target={target({ file: { version: "009", path: "x.sql", sheet: "" } })}
        onClose={() => {}}
      />
    );
    await flush();
    expect(screen.getAllByTestId("release-file-statement-state").at(-1)?.textContent).toBe(
      "The file's statement could not be loaded."
    );
    unmount();
  });

  test("the page has one viewer: a file opened from anywhere swaps its content", async () => {
    const Opener = ({ path }: { path: string }) => {
      const open = useOpenReleaseFile();
      return (
        <button type="button" onClick={() => open?.({ file: { version: path, path, sheet: `projects/p/sheets/${path}` } })}>
          {`open ${path}`}
        </button>
      );
    };
    render(
      <ReleaseFileViewerProvider>
        <Opener path="a.sql" />
        <Opener path="b.sql" />
      </ReleaseFileViewerProvider>
    );
    fireEvent.click(screen.getByRole("button", { name: "open a.sql" }));
    await flush();
    fireEvent.click(screen.getByRole("button", { name: "open b.sql" }));
    await flush();
    expect(screen.getAllByTestId("release-file-viewer")).toHaveLength(1);
    expect(screen.getByTestId("release-file-viewer").textContent).toContain("b.sql");
  });

  test("each failed file opened in the page's viewer gets its diagnostic drawn, reopened ones too", async () => {
    const failedTarget = (path: string) => ({
      file: { version: path, path, sheet: `projects/p/sheets/${path}` },
      context: { kind: "failed" as const, error: `error in ${path}`, range: { start: 13, end: 37 } },
    });
    const Opener = ({ target: t }: { target: ReleaseFileViewerTarget }) => {
      const open = useOpenReleaseFile();
      return (
        <button type="button" onClick={() => open?.(t)}>
          {`open ${t.file.path}`}
        </button>
      );
    };
    render(
      <ReleaseFileViewerProvider>
        <Opener target={failedTarget("a.sql")} />
        <Opener target={failedTarget("b.sql")} />
      </ReleaseFileViewerProvider>
    );
    const drewErrorFor = (path: string) =>
      mocks.drawn.some((advices) =>
        advices.some((a) => (a as { message: string }).message === `error in ${path}`)
      );
    fireEvent.click(screen.getByRole("button", { name: "open a.sql" }));
    await flush();
    expect(drewErrorFor("a.sql")).toBe(true);
    fireEvent.click(screen.getByRole("button", { name: "open b.sql" }));
    await flush();
    expect(drewErrorFor("b.sql")).toBe(true);
    // Closed and reopened.
    fireEvent.keyDown(screen.getByTestId("release-file-viewer"), { key: "Escape" });
    await flush();
    mocks.drawn.length = 0;
    fireEvent.click(screen.getByRole("button", { name: "open a.sql" }));
    await flush();
    expect(drewErrorFor("a.sql")).toBe(true);
  });

  test("the same file opened again from its failed section scrolls to the failed statement", async () => {
    const file = { version: "a", path: "a.sql", sheet: "projects/p/sheets/a" };
    const Opener = ({ label, t }: { label: string; t: ReleaseFileViewerTarget }) => {
      const open = useOpenReleaseFile();
      return (
        <button type="button" onClick={() => open?.(t)}>
          {label}
        </button>
      );
    };
    render(
      <ReleaseFileViewerProvider>
        <Opener label="from changes" t={{ file }} />
        <Opener
          label="from failed"
          t={{ file, context: { kind: "failed", error: "boom", range: { start: 13, end: 37 } } }}
        />
      </ReleaseFileViewerProvider>
    );
    fireEvent.click(screen.getByRole("button", { name: "from changes" }));
    await flush();
    expect(mocks.revealed).toEqual([]);
    fireEvent.click(screen.getByRole("button", { name: "from failed" }));
    await flush();
    expect(mocks.revealed).toEqual([2]);
  });
});
