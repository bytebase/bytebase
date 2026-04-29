// The heavy SchemaDiagram surface (Canvas, Navigator, ER, FK lines,
// auto-layout, ...) was migrated to React on 2026-04-29. The remaining
// Vue surfaces here only consume the small `SchemaDiagramIcon` SVG;
// every other consumer mounts the React diagram via `ReactPageMount`
// (page="SchemaDiagramPage").
import SchemaDiagramIcon from "./SchemaDiagramIcon.vue";

export { SchemaDiagramIcon };
