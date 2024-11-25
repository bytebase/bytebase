import { useLocalStorage } from "@vueuse/core";
import type { ClientMiddleware } from "nice-grpc-web";

// eslint-disable-next-line @typescript-eslint/no-empty-object-type
export type SimulateLatencyOptions = {};

const simulateLatency = (minMS: number, maxMS: number, tags: string[] = []) => {
  if (maxMS < minMS) return;
  if (maxMS <= 0) return;
  const ms = Math.random() * (maxMS - minMS) + minMS;
  if (ms <= 0) return;
  const parts = ["[SimulateLatency]"];
  parts.push(...tags);
  parts.push(String(ms));
  console.debug(parts.join(" "));
  return new Promise((resolve) => setTimeout(resolve, ms));
};

type Config = {
  enabled: boolean;
  before: number[];
  after: number[];
};

const defaults = (): Config => ({
  enabled: false,
  before: [0, 0],
  after: [0, 0],
});

const config = useLocalStorage<Config>("bb.debug.simulate-latency", defaults, {
  serializer: {
    read(raw: string) {
      try {
        const config = JSON.parse(raw) as Config;
        if (!config) return defaults();
        if (typeof config !== "object") return defaults();
        if (typeof config.enabled !== "boolean") return defaults();
        if (!Array.isArray(config.before)) return defaults();
        if (config.before.length !== 2) return defaults();
        if (typeof config.before[0] !== "number") return defaults();
        if (typeof config.before[1] !== "number") return defaults();
        if (!Array.isArray(config.after)) return defaults();
        if (config.after.length !== 2) return defaults();
        if (typeof config.after[0] !== "number") return defaults();
        if (typeof config.after[1] !== "number") return defaults();

        return config;
      } catch {
        return defaults();
      }
    },
    write(config) {
      return JSON.stringify(config);
    },
  },
});

/**
 * Way to define a grpc-web middleware
 * ClientMiddleware<CallOptionsExt = {}, RequiredCallOptionsExt = {}>
 * See
 *   - https://github.com/deeplay-io/nice-grpc/blob/master/packages/nice-grpc-client-middleware-deadline/src/index.ts
 *   - https://github.com/deeplay-io/nice-grpc/tree/master/packages/nice-grpc-web#middleware
 *   as an example.
 */
export const simulateLatencyMiddleware: ClientMiddleware<SimulateLatencyOptions> =
  async function* (call, options) {
    const { enabled, before, after } = config.value;

    const apiPath = call.method.path;
    if (!call.responseStream) {
      if (enabled) {
        await simulateLatency(before[0], before[1], [apiPath, "[BEFORE]"]);
      }
      const response = yield* call.next(call.request, options);
      if (enabled) {
        await simulateLatency(after[0], after[1], [apiPath, "[AFTER]"]);
      }
      return response;
    } else {
      if (enabled) {
        await simulateLatency(before[0], before[1], [apiPath, "[BEFORE]"]);
      }

      for await (const response of call.next(call.request, options)) {
        if (enabled) {
          simulateLatency(after[0], after[1], [apiPath, "[AFTER]"]);
        }
        yield response;

        // Simulate latency before going to next streaming round-trip.
        if (enabled) {
          await simulateLatency(before[0], before[1], [apiPath, "[BEFORE]"]);
        }
      }

      return;
    }
  };
