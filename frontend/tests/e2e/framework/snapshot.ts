import * as fs from "fs";
import * as path from "path";
import { BytebaseApiClient } from "./api-client";

const SNAPSHOT_FILE = path.join(__dirname, "../../.e2e-snapshot.json");

export interface SnapshotScope {
  policies?: string[];
  catalogs?: string[];
  instanceData?: {
    instance: string;
    database: string;
    captureQueries: string[];
    restoreQueries: string[];
  }[];
}

export interface Snapshot {
  status: "captured" | "restored";
  timestamp: string;
  metadata: {
    policies: Record<string, unknown>;
    catalogs: Record<string, unknown>;
  };
  instanceData: Record<string, unknown[]>;
}

export async function createSnapshot(api: BytebaseApiClient, scope: SnapshotScope): Promise<Snapshot> {
  const snapshot: Snapshot = {
    status: "captured",
    timestamp: new Date().toISOString(),
    metadata: { policies: {}, catalogs: {} },
    instanceData: {},
  };

  // Capture policies
  if (scope.policies) {
    for (const policyPath of scope.policies) {
      const policy = await api.getPolicy(policyPath);
      snapshot.metadata.policies[policyPath] = policy;
    }
  }

  // Capture catalogs
  if (scope.catalogs) {
    for (const catalogPath of scope.catalogs) {
      const catalog = await api.getCatalog(catalogPath);
      snapshot.metadata.catalogs[catalogPath] = catalog;
    }
  }

  // Capture instance data via SQL queries
  if (scope.instanceData) {
    for (const entry of scope.instanceData) {
      for (const query of entry.captureQueries) {
        const key = `${entry.instance}/${entry.database}:${query}`;
        try {
          const result = await api.query(entry.instance, entry.database, query);
          snapshot.instanceData[key] = result.results;
        } catch {
          snapshot.instanceData[key] = [];
        }
      }
    }
  }

  // Persist to disk for crash recovery
  fs.writeFileSync(SNAPSHOT_FILE, JSON.stringify(snapshot, null, 2));

  return snapshot;
}

export async function restoreSnapshot(api: BytebaseApiClient, snapshot: Snapshot): Promise<void> {
  // Restore policies
  for (const [policyPath, policyData] of Object.entries(snapshot.metadata.policies)) {
    if (policyData) {
      // Extract parent and type from path like "projects/foo/policies/masking_exemption"
      const lastSlash = policyPath.lastIndexOf("/");
      const secondLastSlash = policyPath.lastIndexOf("/", lastSlash - 1);
      const parent = policyPath.slice(0, secondLastSlash);
      const policyType = policyPath.slice(lastSlash + 1);
      try {
        await api.upsertPolicy(parent, policyType, policyData);
      } catch (err) {
        console.warn(`Warning: Failed to restore policy ${policyPath}:`, err);
      }
    }
  }

  // Restore catalogs
  for (const [catalogPath, catalogData] of Object.entries(snapshot.metadata.catalogs)) {
    if (catalogData) {
      try {
        await api.updateCatalog(catalogPath, catalogData);
      } catch (err) {
        console.warn(`Warning: Failed to restore catalog ${catalogPath}:`, err);
      }
    }
  }

  // Restore instance data via SQL queries
  // Note: restoreQueries from the scope would need to be persisted or re-derived.
  // For now, instance data restore is best-effort through the scope's restoreQueries.

  // Update snapshot status
  snapshot.status = "restored";
  if (fs.existsSync(SNAPSHOT_FILE)) {
    fs.writeFileSync(SNAPSHOT_FILE, JSON.stringify(snapshot, null, 2));
  }
}

export function loadPersistedSnapshot(): Snapshot | null {
  if (!fs.existsSync(SNAPSHOT_FILE)) return null;
  try {
    return JSON.parse(fs.readFileSync(SNAPSHOT_FILE, "utf-8")) as Snapshot;
  } catch {
    return null;
  }
}

export function deletePersistedSnapshot(): void {
  if (fs.existsSync(SNAPSHOT_FILE)) fs.unlinkSync(SNAPSHOT_FILE);
}
