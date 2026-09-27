import Emittery from "emittery";
import type { AIContextEvents } from "../types";

/**
 * Module-level AI event bus. `AIContextProvider` exposes it as `events`,
 * and SQL Editor components outside the provider import it directly. Single
 * shared instance so emit/on from either side reaches the other.
 */
export const aiContextEvents: AIContextEvents = new Emittery();
