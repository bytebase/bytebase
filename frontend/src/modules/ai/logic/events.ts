import Emittery from "emittery";
import type { AIContextEvents } from "../types";

/**
 * Module-level AI event bus. Imported by both the Vue ProvideAIContext
 * (which wires it into the Vue-injected context) and React consumers that
 * can't participate in Vue's provide/inject. Single shared instance so
 * emit/on from either side reaches the other.
 */
export const aiContextEvents: AIContextEvents = new Emittery();
