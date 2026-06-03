// Route-shell placeholder for the react-router route table.
//
// The Vue route-shell components (`SettingRouteShell`, `ProjectRouteShell`,
// `InstanceRouteShell`, `IssuesRouteShell`) render an empty teleport target for
// the Vue ReactRouteShellBridge and require props supplied by that bridge; they
// do not render an `<Outlet/>` and so cannot host react-router children yet.
// Each shell-parent route uses this placeholder so its child leaf routes still
// render. A later phase replaces these with `<Outlet/>`-aware React shells that
// own breadcrumbs / permission guards / document title.
//
// (The Splash / Body / Dashboard layout placeholders that previously lived here
// have been replaced by the real layouts in `@/react/app/layouts/*`.)
import { Outlet } from "react-router-dom";

export const RouteShellOutletPlaceholder = () => <Outlet />;
