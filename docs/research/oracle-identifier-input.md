# Oracle SID / service name input

Research date: September 17, 2026. Five independent agents researched first-party product documentation and proposed alternatives. This is design judgment, not usability-test evidence. Only the DBeaver reviewer inspected a current documentation screenshot; other observations are documented behavior unless stated otherwise.

## Decision

Use two aligned rows: **Connect using** with visible **Service name / SID** choices, followed by an input labeled **Service name** or **SID**. Use Bytebase's shared radio-based SegmentedControl in the control column. Keep ordinary input sizing and the existing responsive label/control layout.

The extra row buys an explicit visible input label, full input width, and a consistent noninteractive label column. A dropdown is compact but hides one of only two choices. Putting the selector beside the input reduces room for long service names. Putting it in the label column makes that column do two jobs and increases wrapping with translations.

## Five independent proposals

| Research lens | Verified evidence | Proposed design | Assessment |
| --- | --- | --- | --- |
| Oracle tools | Classic SQL Developer documents SID and Service name choices; its older tutorial defaults SID to xe. Current extension docs list a Type and Service Name in Basic connection settings. | Database identifier label, radios above one input. | Good compact runner-up; the active input lacks its own visible label. |
| JetBrains DataGrip | Connection type offers SID, Service name, TNS, Long, URL only, with mode-specific fields. | Connect using row, then a normally labeled input. | Best layout for Bytebase; use the existing segmented primitive for the visible binary choice. |
| DBeaver | Official screenshot shows a Database input followed by a Service Name dropdown. | Database identifier row with dropdown before input. | Most compact; hides the alternative and divides input width. |
| Navicat / DbVisualizer | Navicat documents a corresponding radio selection for Service Name or SID. DbVisualizer documents structured fields and Test Connection, but does not establish this exact selector. | Visible mode selector and input together in control column, stacking when narrow. | Reasonable density option, but variable wrapping and reduced input width. |
| Cloud connectors | Airbyte explicitly offers Service Name/System ID and recommends Service Name. AWS DMS uses a combined SID/Service name label. Azure documents distinct SID/service semantics. | Connect using row with segmented choice, then active labeled input. | Winner with DataGrip proposal: clear semantics and local component consistency. |

Sources:

- [Oracle SQL Developer tutorial](https://docs.oracle.com/en/database/oracle/oracle-database/19/tdddg/connecting-oracle-database-sql-developer.html)
- [SQL Developer for VS Code 26.2](https://docs.oracle.com/en/database/oracle/sql-developer-vscode/26.2/sqdnx/connecting-your-database.html)
- [DataGrip Oracle connections](https://www.jetbrains.com/help/datagrip/oracle.html)
- [DBeaver Oracle docs](https://dbeaver.com/docs/dbeaver/Oracle/) and [connection screenshot](https://dbeaver.com/docs/dbeaver/images/database/oracle/oracle-connection-main.png)
- [Navicat 17 Linux manual, General Settings / Oracle, printed page 42](https://www.navicat.com/manual/pdf_manual/en/navicat_17/linux_manual/navicat_en.pdf)
- [DbVisualizer connection basics](https://www.dbvis.com/docs/ug/getting-started/creating-a-connection-basics/)
- [Airbyte Oracle connection types](https://docs.airbyte.com/integrations/sources/oracle#step-3-configure-connection-type-and-schemas)
- [AWS DMS Oracle endpoint example](https://docs.aws.amazon.com/dms/latest/sbs/oracle-s3-data-lake-step-3.html)
- [Azure Data Factory Oracle connector](https://learn.microsoft.com/en-us/azure/data-factory/connector-oracle)

## Behavior

- Default new connections to Service name, with an empty value. Preserve the saved mode on edit. This recommendation is supported by Airbyte's guidance and [Oracle service/PDB semantics](https://docs.oracle.com/en/database/oracle/oracle-database/26/multi/administering-a-cdb-with-sql-plus.html); it is not a universal competitor default.
- Keep explicit mode state and separate SID/service-name drafts. Switching restores the selected draft. Clearing an input never changes mode. Examples belong in placeholders, not stored values.
- Submit only the active field; update both serialized fields atomically when switching. Keep inactive drafts outside the submitted data source.
- Give the input an associated visible label. Name the selection group Connect using; preserve standard radio keyboard behavior and focus. Follow the [WAI-ARIA radio pattern](https://www.w3.org/WAI/ARIA/apg/patterns/radio/).
- Validate the active value with the existing connection requirements and adjacent, mode-specific errors. Do not invent identifier regexes. Test Connection remains available for server validation.
- Stack labels above controls at narrow form widths. Avoid permanently displayed definitions; optional help can explain that a service name and SID are distinct identifiers. Do not label SID deprecated.

## Findings before implementation

Before implementation, OracleSIDServiceNameInput in [DataSourceForm.tsx](../../frontend/src/components/instance/DataSourceForm.tsx) derived its mode from SID truthiness. Clearing SID therefore changes mode; selecting SID inserts XE. Switching also clears values instead of preserving independent drafts. The two callbacks each spread the same dataSource snapshot, so mode changes should become one atomic update. Validation currently stores the shared Oracle requiredness error under serviceName; the redesign must retain error wiring or deliberately update it.

The [frontend UX contract](../agents/frontend-ux.md) already calls for choices before dependent fields, aligned label/control columns, accessible names, and preserved drafts. The subsequent implementation extracts [OracleConnectionFields](../../frontend/src/components/instance/OracleConnectionFields.tsx), adds explicit mode and draft state, and uses one atomic identifier update. Focused tests cover clearing, switching, reverting, disabled controls, and validation. The existing shared Oracle requiredness message remains associated with the active input.
