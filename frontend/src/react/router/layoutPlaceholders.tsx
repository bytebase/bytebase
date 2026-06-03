// TEMPORARY layout placeholders for the react-router route table.
//
// The real React layouts (Splash / Body / Dashboard) do not exist yet. Each
// vue-router layout route (`component: SplashLayout | BodyLayout |
// DashboardLayout`) is translated to a react-router layout route whose
// `element` is one of these placeholders. A placeholder renders only an
// `<Outlet/>`, so child routes still mount. A later migration phase swaps
// these for the real layout components.
import { Outlet } from "react-router-dom";

export const SplashLayoutPlaceholder = () => <Outlet />;

export const BodyLayoutPlaceholder = () => <Outlet />;

export const DashboardLayoutPlaceholder = () => <Outlet />;

// The Vue route-shell components (`SettingRouteShell`, `ProjectRouteShell`,
// `InstanceRouteShell`, `IssuesRouteShell`) render an empty teleport target
// for the Vue ReactRouteShellBridge and require props supplied by that bridge;
// they do not render an `<Outlet/>` and so cannot host react-router children
// yet. Each shell-parent route uses this placeholder so its child leaf routes
// still render. A later phase replaces these with `<Outlet/>`-aware React
// shells that own breadcrumbs / permission guards / document title.
export const RouteShellOutletPlaceholder = () => <Outlet />;
