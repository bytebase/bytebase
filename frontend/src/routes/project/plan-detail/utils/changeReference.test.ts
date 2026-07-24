import type { TFunction } from "i18next";
import { describe, expect, it } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type { Environment } from "@/types/v1/environment";
import {
  derivePlanChangeReference,
  type PlanChangeReferenceResources,
  splitPlanChangeReferenceLabel,
} from "./changeReference";

const t = ((key: string, options?: Record<string, string | number>): string => {
  if (key === "plan.spec.change-reference.accessible-label") {
    return `Change ${options?.index}: ${options?.label}`;
  }
  if (key === "plan.spec.change-reference.database-count") {
    return `${options?.count} databases`;
  }
  if (key === "plan.spec.change-reference.environment-overflow") {
    return `${options?.environment} +${options?.count} environments`;
  }
  if (key === "plan.spec.change-reference.new-change") {
    return "New change";
  }
  return key;
}) as TFunction;

const DB_ORDERS = "instances/prod/databases/orders";
const DB_CUSTOMERS = "instances/staging/databases/customers";
const DB_AUDIT = "instances/dev/databases/audit";

const resources = ({
  databases = [],
  environments = [],
}: {
  databases?: Database[];
  environments?: Environment[];
} = {}): PlanChangeReferenceResources => ({
  databaseGroupsByName: {},
  databasesByName: Object.fromEntries(
    databases.map((database) => [database.name, database])
  ),
  environmentList: environments,
});

const changeSpec = (id: string, targets: string[]): Plan_Spec =>
  ({
    id,
    config: {
      case: "changeDatabaseConfig",
      value: { targets },
    },
  }) as Plan_Spec;

describe("derivePlanChangeReference", () => {
  it("uses the database name for one database", () => {
    const spec = changeSpec("spec-1", [DB_ORDERS]);
    const reference = derivePlanChangeReference({
      index: 0,
      resources: resources(),
      siblings: [spec],
      spec,
      t,
    });

    expect(reference).toMatchObject({
      accessibleLabel: "Change 1: orders",
      icon: "database",
      index: 1,
      label: "orders",
      showIndex: false,
      showIndexWithCountLabel: false,
      specId: "spec-1",
    });
    expect(reference.countLabel).toBeUndefined();
  });

  it("provides a count and deployment-ordered environment fallback", () => {
    const spec = changeSpec("spec-1", [DB_ORDERS, DB_CUSTOMERS, DB_AUDIT]);
    const reference = derivePlanChangeReference({
      index: 0,
      resources: resources({
        databases: [
          {
            name: DB_ORDERS,
            effectiveEnvironment: "environments/prod",
          } as Database,
          {
            name: DB_CUSTOMERS,
            effectiveEnvironment: "environments/staging",
          } as Database,
          {
            name: DB_AUDIT,
            effectiveEnvironment: "environments/dev",
          } as Database,
        ],
        environments: [
          {
            name: "environments/dev",
            order: 0,
            title: "Development",
          } as Environment,
          {
            name: "environments/staging",
            order: 1,
            title: "Staging",
          } as Environment,
          {
            name: "environments/prod",
            order: 2,
            title: "Production",
          } as Environment,
        ],
      }),
      siblings: [spec],
      spec,
      t,
    });

    expect(reference.label).toBe("orders, customers, audit");
    expect(reference.countLabel).toBe(
      "3 databases · Production +2 environments"
    );
  });

  it("uses the database group resource ID without requiring hydration", () => {
    const spec = changeSpec("spec-1", [
      "projects/acme/databaseGroups/tenant-prod",
    ]);
    const reference = derivePlanChangeReference({
      index: 0,
      resources: resources(),
      siblings: [spec],
      spec,
      t,
    });

    expect(reference).toMatchObject({
      icon: "database-group",
      label: "tenant-prod",
    });
  });

  it("supports create-database and export specs", () => {
    const createSpec = {
      id: "spec-create",
      config: {
        case: "createDatabaseConfig",
        value: {
          database: "reporting",
          target: "instances/prod",
        },
      },
    } as Plan_Spec;
    const exportSpec = {
      id: "spec-export",
      config: {
        case: "exportDataConfig",
        value: { targets: [DB_ORDERS] },
      },
    } as Plan_Spec;

    expect(
      derivePlanChangeReference({
        index: 0,
        resources: resources(),
        siblings: [createSpec],
        spec: createSpec,
        t,
      })
    ).toMatchObject({
      icon: "create-database",
      label: "reporting",
    });
    expect(
      derivePlanChangeReference({
        index: 0,
        resources: resources(),
        siblings: [exportSpec],
        spec: exportSpec,
        t,
      })
    ).toMatchObject({
      icon: "export",
      label: "orders",
    });
  });

  it("keeps release-backed changes target-derived", () => {
    const spec = {
      id: "spec-release",
      config: {
        case: "changeDatabaseConfig",
        value: {
          release: "projects/acme/releases/release-1",
          targets: [DB_ORDERS],
        },
      },
    } as Plan_Spec;

    expect(
      derivePlanChangeReference({
        index: 0,
        resources: resources(),
        siblings: [spec],
        spec,
        t,
      })
    ).toMatchObject({
      icon: "database",
      label: "orders",
    });
  });

  it("uses New change for an empty draft", () => {
    const spec = changeSpec("spec-1", []);
    const reference = derivePlanChangeReference({
      index: 0,
      resources: resources(),
      siblings: [spec],
      spec,
      t,
    });

    expect(reference.label).toBe("New change");
  });

  it("falls back to readable resource text and then a short spec ID", () => {
    const malformedTarget = changeSpec("spec-target", [
      "legacy/targets/orphaned",
    ]);
    const unknownConfig = {
      id: "2f2894b5-50e8-4d7b-9c58-5f5f08109460",
      config: { case: undefined, value: undefined },
    } as unknown as Plan_Spec;

    expect(
      derivePlanChangeReference({
        index: 0,
        resources: resources(),
        siblings: [malformedTarget],
        spec: malformedTarget,
        t,
      }).label
    ).toBe("orphaned");
    expect(
      derivePlanChangeReference({
        index: 0,
        resources: resources(),
        siblings: [unknownConfig],
        spec: unknownConfig,
        t,
      }).label
    ).toBe("2f2894b5");
  });

  it("keeps a usable count when resource enrichment is unavailable", () => {
    const spec = changeSpec("spec-1", [DB_ORDERS, DB_CUSTOMERS]);
    const reference = derivePlanChangeReference({
      index: 0,
      resources: resources(),
      siblings: [spec],
      spec,
      t,
    });

    expect(reference.label).toBe("orders, customers");
    expect(reference.countLabel).toBe("2 databases");
  });

  it("adds an environment qualifier when sibling labels collide", () => {
    const prodSpec = changeSpec("spec-prod", [
      "instances/prod/databases/orders",
    ]);
    const stagingSpec = changeSpec("spec-staging", [
      "instances/staging/databases/orders",
    ]);
    const reference = derivePlanChangeReference({
      index: 0,
      resources: resources({
        databases: [
          {
            name: "instances/prod/databases/orders",
            effectiveEnvironment: "environments/prod",
          } as Database,
          {
            name: "instances/staging/databases/orders",
            effectiveEnvironment: "environments/staging",
          } as Database,
        ],
        environments: [
          {
            name: "environments/staging",
            order: 0,
            title: "Staging",
          } as Environment,
          {
            name: "environments/prod",
            order: 1,
            title: "Production",
          } as Environment,
        ],
      }),
      siblings: [prodSpec, stagingSpec],
      spec: prodSpec,
      t,
    });

    expect(reference.label).toBe("orders · Production");
    expect(reference.showIndex).toBe(false);
    expect(reference.showIndexWithCountLabel).toBe(false);
  });

  it("uses the instance when duplicate labels share an environment", () => {
    const primarySpec = changeSpec("spec-primary", [
      "instances/primary/databases/orders",
    ]);
    const replicaSpec = changeSpec("spec-replica", [
      "instances/replica/databases/orders",
    ]);
    const reference = derivePlanChangeReference({
      index: 0,
      resources: resources({
        databases: [
          {
            name: "instances/primary/databases/orders",
            effectiveEnvironment: "environments/prod",
            instanceResource: { title: "Primary" },
          } as Database,
          {
            name: "instances/replica/databases/orders",
            effectiveEnvironment: "environments/prod",
            instanceResource: { title: "Replica" },
          } as Database,
        ],
        environments: [
          {
            name: "environments/prod",
            order: 0,
            title: "Production",
          } as Environment,
        ],
      }),
      siblings: [primarySpec, replicaSpec],
      spec: primarySpec,
      t,
    });

    expect(reference.label).toBe("orders · Primary");
    expect(reference.showIndex).toBe(false);
    expect(reference.showIndexWithCountLabel).toBe(false);
  });

  it("keeps identical-target siblings distinct through their indexes", () => {
    const first = changeSpec("spec-1", [DB_ORDERS]);
    const second = changeSpec("spec-2", [DB_ORDERS]);

    expect(
      derivePlanChangeReference({
        index: 0,
        resources: resources(),
        siblings: [first, second],
        spec: first,
        t,
      })
    ).toMatchObject({
      accessibleLabel: "Change 1: orders",
      label: "orders",
      showIndex: true,
      showIndexWithCountLabel: false,
    });
    expect(
      derivePlanChangeReference({
        index: 1,
        resources: resources(),
        siblings: [first, second],
        spec: second,
        t,
      })
    ).toMatchObject({
      accessibleLabel: "Change 2: orders",
      label: "orders",
      showIndex: true,
      showIndexWithCountLabel: false,
    });
  });

  it("shows indexes when different long labels share a count fallback", () => {
    const first = changeSpec("spec-1", [DB_ORDERS, DB_CUSTOMERS]);
    const second = changeSpec("spec-2", [
      DB_AUDIT,
      "instances/prod/databases/payments",
    ]);

    expect(
      derivePlanChangeReference({
        index: 0,
        resources: resources(),
        siblings: [first, second],
        spec: first,
        t,
      })
    ).toMatchObject({
      showIndex: false,
      showIndexWithCountLabel: true,
    });
    expect(
      derivePlanChangeReference({
        index: 1,
        resources: resources(),
        siblings: [first, second],
        spec: second,
        t,
      })
    ).toMatchObject({
      showIndex: false,
      showIndexWithCountLabel: true,
    });
  });

  it("does not show indexes when icons distinguish equal titles", () => {
    const change = changeSpec("spec-change", [DB_ORDERS]);
    const dataExport = {
      id: "spec-export",
      config: {
        case: "exportDataConfig",
        value: { targets: [DB_ORDERS] },
      },
    } as Plan_Spec;

    expect(
      derivePlanChangeReference({
        index: 0,
        resources: resources(),
        siblings: [change, dataExport],
        spec: change,
        t,
      }).showIndex
    ).toBe(false);
    expect(
      derivePlanChangeReference({
        index: 1,
        resources: resources(),
        siblings: [change, dataExport],
        spec: dataExport,
        t,
      }).showIndex
    ).toBe(false);
  });
});

describe("splitPlanChangeReferenceLabel", () => {
  it("splits without breaking a grapheme cluster", () => {
    const text = "customer_orders_👨‍👩‍👧‍👦_production_archive";
    const split = splitPlanChangeReferenceLabel(text);

    expect(split.prefix + split.suffix).toBe(text);
    expect(split.suffix).toBe("on_archive");
  });
});
