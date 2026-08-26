import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import { MCPEngineEnforcement_ReadOnlyDepth } from "@/types/proto-es/v1/workspace_service_pb";
import { MCPMethodClass } from "@/types/proto-es/v1/annotation_pb";
import { MCPModeContentsSheet } from "./MCPModeContentsSheet";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

const REDSHIFT_NOTE =
  "Outside a datashare database the driver also opens the session read-only. " +
  "A datashare database cannot run a read-only transaction, so there it is " +
  "statement classification alone.";

const info = {
  workspace: "workspaces/ws",
  capability: MCPSetting_Capability.READ_ONLY,
  modes: [
    {
      capability: MCPSetting_Capability.READ_ONLY,
      servedClasses: [MCPMethodClass.READ],
    },
  ],
  methods: [
    {
      method: "/bytebase.v1.SQLService/Query",
      operationId: "bytebase.v1.SQLService.Query",
      class: MCPMethodClass.READ,
      permission: "bb.sql.select",
      authMethod: 0,
    },
    {
      method: "/bytebase.v1.ActuatorService/GetActuatorInfo",
      operationId: "bytebase.v1.ActuatorService.GetActuatorInfo",
      class: MCPMethodClass.READ,
      permission: "",
      authMethod: 0,
    },
  ],
  engines: [
    {
      engine: Engine.REDSHIFT,
      readOnlyDepth: MCPEngineEnforcement_ReadOnlyDepth.STATEMENT,
      masking: 0,
      note: REDSHIFT_NOTE,
    },
  ],
  ignoreMaskingExemptions: false,
  dataMaskingAvailable: true,
} as unknown as Parameters<typeof MCPModeContentsSheet>[0]["info"];

// The sheet portals, so the container is attached to the document.
const renderSheet = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => root.render(element));
  return () => {
    act(() => root.unmount());
    container.remove();
  };
};

afterEach(() => {
  document.body.innerHTML = "";
});

describe("MCPModeContentsSheet", () => {
  // Codex, #21236: engine.note is server-authored English (mcp_info.go
  // mcpEngineNote). Rendering it raw shows English in every other locale, which
  // is the rule at AGENTS.md:146-150. The server owns whether an engine has a
  // caveat; the words are the frontend's.
  test("translates the engine note instead of rendering the server's prose", () => {
    const unmount = renderSheet(
      <MCPModeContentsSheet
        open
        capability={MCPSetting_Capability.READ_ONLY}
        info={info}
        modeLabel="Read-only"
        ignoreMaskingExemptions={false}
        onClose={() => {}}
      />
    );

    expect(document.body.textContent).toContain(
      "settings.mcp.contents.notes.redshift"
    );
    expect(document.body.textContent).not.toContain("datashare");

    unmount();
  });

  // Codex, #21236: a newer backend can attach a note to an engine this bundle
  // has no wording for. Returning the server's sentence put English prose in
  // every locale; dropping it lost a caveat that is the only place a
  // per-engine floor is stated. It announces the caveat in the viewer's
  // language and names the remedy instead.
  test("an engine with no wording announces the caveat rather than quoting it", () => {
    const future = {
      ...info,
      engines: [
        {
          engine: Engine.SPANNER,
          readOnlyDepth: MCPEngineEnforcement_ReadOnlyDepth.STATEMENT,
          masking: 0,
          note: "Spanner does something this build has never heard of.",
        },
      ],
    } as unknown as Parameters<typeof MCPModeContentsSheet>[0]["info"];

    const unmount = renderSheet(
      <MCPModeContentsSheet
        open
        capability={MCPSetting_Capability.READ_ONLY}
        info={future}
        modeLabel="Read-only"
        ignoreMaskingExemptions={false}
        onClose={() => {}}
      />
    );

    expect(document.body.textContent).toContain(
      "settings.mcp.contents.notes.unavailable"
    );
    expect(document.body.textContent).not.toContain("never heard of");

    unmount();
  });

  test("the masking description follows the value it is given", () => {
    const unmount = renderSheet(
      <MCPModeContentsSheet
        open
        capability={MCPSetting_Capability.READ_ONLY}
        info={info}
        modeLabel="Read-only"
        ignoreMaskingExemptions
        onClose={() => {}}
      />
    );

    // Not info.ignoreMaskingExemptions, which is false above.
    expect(document.body.textContent).toContain(
      "settings.mcp.contents.masking.description-on"
    );
    expect(document.body.textContent).not.toContain(
      "settings.mcp.contents.masking.description-off"
    );

    unmount();
  });

  // Codex, #21236: the permission lived in a native title on a Badge, which is
  // a span — unfocusable, so a keyboard user could never read it, and it
  // appeared nowhere else. It is rendered now.
  test("renders each method's permission rather than hiding it in a tooltip", () => {
    const unmount = renderSheet(
      <MCPModeContentsSheet
        open
        capability={MCPSetting_Capability.READ_ONLY}
        info={info}
        modeLabel="Read-only"
        ignoreMaskingExemptions={false}
        onClose={() => {}}
      />
    );

    expect(document.body.textContent).toContain("bb.sql.select");
    // A method that authorizes in its handler says so, rather than blank.
    expect(document.body.textContent).toContain(
      "settings.mcp.contents.methods.handler"
    );
    // Nothing depends on hover: no title attribute carries the information.
    const titled = [...document.querySelectorAll("[title]")].filter((n) =>
      n.getAttribute("title")?.includes("bb.")
    );
    expect(titled).toHaveLength(0);

    unmount();
  });
});
