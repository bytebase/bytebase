/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { toErrorMessage } from '../../../base/common/errorMessage.js';
import { Emitter } from '../../../base/common/event.js';
import { hash } from '../../../base/common/hash.js';
import { Disposable } from '../../../base/common/lifecycle.js';
import { ResourceMap } from '../../../base/common/map.js';
import { isWindows } from '../../../base/common/platform.js';
import { joinPath } from '../../../base/common/resources.js';
import { isNumber, isString } from '../../../base/common/types.js';
import { URI } from '../../../base/common/uri.js';
import { RawContextKey } from '../../contextkey/common/contextkey.js';
import { createDecorator } from '../../instantiation/common/instantiation.js';
export const ILogService = createDecorator('logService');
export const ILoggerService = createDecorator('loggerService');
function now() {
    return new Date().toISOString();
}
export function isLogLevel(thing) {
    return isNumber(thing);
}
export var LogLevel;
(function (LogLevel) {
    LogLevel[LogLevel["Off"] = 0] = "Off";
    LogLevel[LogLevel["Trace"] = 1] = "Trace";
    LogLevel[LogLevel["Debug"] = 2] = "Debug";
    LogLevel[LogLevel["Info"] = 3] = "Info";
    LogLevel[LogLevel["Warning"] = 4] = "Warning";
    LogLevel[LogLevel["Error"] = 5] = "Error";
})(LogLevel || (LogLevel = {}));
export const DEFAULT_LOG_LEVEL = LogLevel.Info;
export function log(logger, level, message) {
    switch (level) {
        case LogLevel.Trace:
            logger.trace(message);
            break;
        case LogLevel.Debug:
            logger.debug(message);
            break;
        case LogLevel.Info:
            logger.info(message);
            break;
        case LogLevel.Warning:
            logger.warn(message);
            break;
        case LogLevel.Error:
            logger.error(message);
            break;
        case LogLevel.Off: /* do nothing */ break;
        default: throw new Error(`Invalid log level ${level}`);
    }
}
function format(args, verbose = false) {
    let result = '';
    for (let i = 0; i < args.length; i++) {
        let a = args[i];
        if (a instanceof Error) {
            a = toErrorMessage(a, verbose);
        }
        if (typeof a === 'object') {
            try {
                a = JSON.stringify(a);
            }
            catch (e) { }
        }
        result += (i > 0 ? ' ' : '') + a;
    }
    return result;
}
export class AbstractLogger extends Disposable {
    constructor() {
        super(...arguments);
        this.level = DEFAULT_LOG_LEVEL;
        this._onDidChangeLogLevel = this._register(new Emitter());
        this.onDidChangeLogLevel = this._onDidChangeLogLevel.event;
    }
    setLevel(level) {
        if (this.level !== level) {
            this.level = level;
            this._onDidChangeLogLevel.fire(this.level);
        }
    }
    getLevel() {
        return this.level;
    }
    checkLogLevel(level) {
        return this.level !== LogLevel.Off && this.level <= level;
    }
}
export class AbstractMessageLogger extends AbstractLogger {
    constructor(logAlways) {
        super();
        this.logAlways = logAlways;
    }
    checkLogLevel(level) {
        return this.logAlways || super.checkLogLevel(level);
    }
    trace(message, ...args) {
        if (this.checkLogLevel(LogLevel.Trace)) {
            this.log(LogLevel.Trace, format([message, ...args], true));
        }
    }
    debug(message, ...args) {
        if (this.checkLogLevel(LogLevel.Debug)) {
            this.log(LogLevel.Debug, format([message, ...args]));
        }
    }
    info(message, ...args) {
        if (this.checkLogLevel(LogLevel.Info)) {
            this.log(LogLevel.Info, format([message, ...args]));
        }
    }
    warn(message, ...args) {
        if (this.checkLogLevel(LogLevel.Warning)) {
            this.log(LogLevel.Warning, format([message, ...args]));
        }
    }
    error(message, ...args) {
        if (this.checkLogLevel(LogLevel.Error)) {
            if (message instanceof Error) {
                const array = Array.prototype.slice.call(arguments);
                array[0] = message.stack;
                this.log(LogLevel.Error, format(array));
            }
            else {
                this.log(LogLevel.Error, format([message, ...args]));
            }
        }
    }
    flush() { }
}
export class ConsoleMainLogger extends AbstractLogger {
    constructor(logLevel = DEFAULT_LOG_LEVEL) {
        super();
        this.setLevel(logLevel);
        this.useColors = !isWindows;
    }
    trace(message, ...args) {
        if (this.checkLogLevel(LogLevel.Trace)) {
            if (this.useColors) {
                console.log(`\x1b[90m[main ${now()}]\x1b[0m`, message, ...args);
            }
            else {
                console.log(`[main ${now()}]`, message, ...args);
            }
        }
    }
    debug(message, ...args) {
        if (this.checkLogLevel(LogLevel.Debug)) {
            if (this.useColors) {
                console.log(`\x1b[90m[main ${now()}]\x1b[0m`, message, ...args);
            }
            else {
                console.log(`[main ${now()}]`, message, ...args);
            }
        }
    }
    info(message, ...args) {
        if (this.checkLogLevel(LogLevel.Info)) {
            if (this.useColors) {
                console.log(`\x1b[90m[main ${now()}]\x1b[0m`, message, ...args);
            }
            else {
                console.log(`[main ${now()}]`, message, ...args);
            }
        }
    }
    warn(message, ...args) {
        if (this.checkLogLevel(LogLevel.Warning)) {
            if (this.useColors) {
                console.warn(`\x1b[93m[main ${now()}]\x1b[0m`, message, ...args);
            }
            else {
                console.warn(`[main ${now()}]`, message, ...args);
            }
        }
    }
    error(message, ...args) {
        if (this.checkLogLevel(LogLevel.Error)) {
            if (this.useColors) {
                console.error(`\x1b[91m[main ${now()}]\x1b[0m`, message, ...args);
            }
            else {
                console.error(`[main ${now()}]`, message, ...args);
            }
        }
    }
    dispose() {
        // noop
    }
    flush() {
        // noop
    }
}
export class ConsoleLogger extends AbstractLogger {
    constructor(logLevel = DEFAULT_LOG_LEVEL, useColors = true) {
        super();
        this.useColors = useColors;
        this.setLevel(logLevel);
    }
    trace(message, ...args) {
        if (this.checkLogLevel(LogLevel.Trace)) {
            if (this.useColors) {
                console.log('%cTRACE', 'color: #888', message, ...args);
            }
            else {
                console.log(message, ...args);
            }
        }
    }
    debug(message, ...args) {
        if (this.checkLogLevel(LogLevel.Debug)) {
            if (this.useColors) {
                console.log('%cDEBUG', 'background: #eee; color: #888', message, ...args);
            }
            else {
                console.log(message, ...args);
            }
        }
    }
    info(message, ...args) {
        if (this.checkLogLevel(LogLevel.Info)) {
            if (this.useColors) {
                console.log('%c INFO', 'color: #33f', message, ...args);
            }
            else {
                console.log(message, ...args);
            }
        }
    }
    warn(message, ...args) {
        if (this.checkLogLevel(LogLevel.Warning)) {
            if (this.useColors) {
                console.log('%c WARN', 'color: #993', message, ...args);
            }
            else {
                console.log(message, ...args);
            }
        }
    }
    error(message, ...args) {
        if (this.checkLogLevel(LogLevel.Error)) {
            if (this.useColors) {
                console.log('%c  ERR', 'color: #f33', message, ...args);
            }
            else {
                console.error(message, ...args);
            }
        }
    }
    dispose() {
        // noop
    }
    flush() {
        // noop
    }
}
export class AdapterLogger extends AbstractLogger {
    constructor(adapter, logLevel = DEFAULT_LOG_LEVEL) {
        super();
        this.adapter = adapter;
        this.setLevel(logLevel);
    }
    trace(message, ...args) {
        if (this.checkLogLevel(LogLevel.Trace)) {
            this.adapter.log(LogLevel.Trace, [this.extractMessage(message), ...args]);
        }
    }
    debug(message, ...args) {
        if (this.checkLogLevel(LogLevel.Debug)) {
            this.adapter.log(LogLevel.Debug, [this.extractMessage(message), ...args]);
        }
    }
    info(message, ...args) {
        if (this.checkLogLevel(LogLevel.Info)) {
            this.adapter.log(LogLevel.Info, [this.extractMessage(message), ...args]);
        }
    }
    warn(message, ...args) {
        if (this.checkLogLevel(LogLevel.Warning)) {
            this.adapter.log(LogLevel.Warning, [this.extractMessage(message), ...args]);
        }
    }
    error(message, ...args) {
        if (this.checkLogLevel(LogLevel.Error)) {
            this.adapter.log(LogLevel.Error, [this.extractMessage(message), ...args]);
        }
    }
    extractMessage(msg) {
        if (typeof msg === 'string') {
            return msg;
        }
        return toErrorMessage(msg, this.checkLogLevel(LogLevel.Trace));
    }
    dispose() {
        // noop
    }
    flush() {
        // noop
    }
}
export class MultiplexLogger extends AbstractLogger {
    constructor(loggers) {
        super();
        this.loggers = loggers;
        if (loggers.length) {
            this.setLevel(loggers[0].getLevel());
        }
    }
    setLevel(level) {
        for (const logger of this.loggers) {
            logger.setLevel(level);
        }
        super.setLevel(level);
    }
    trace(message, ...args) {
        for (const logger of this.loggers) {
            logger.trace(message, ...args);
        }
    }
    debug(message, ...args) {
        for (const logger of this.loggers) {
            logger.debug(message, ...args);
        }
    }
    info(message, ...args) {
        for (const logger of this.loggers) {
            logger.info(message, ...args);
        }
    }
    warn(message, ...args) {
        for (const logger of this.loggers) {
            logger.warn(message, ...args);
        }
    }
    error(message, ...args) {
        for (const logger of this.loggers) {
            logger.error(message, ...args);
        }
    }
    flush() {
        for (const logger of this.loggers) {
            logger.flush();
        }
    }
    dispose() {
        for (const logger of this.loggers) {
            logger.dispose();
        }
    }
}
export class AbstractLoggerService extends Disposable {
    constructor(logLevel, logsHome, loggerResources) {
        super();
        this.logLevel = logLevel;
        this.logsHome = logsHome;
        this._loggers = new ResourceMap();
        this._onDidChangeLoggers = this._register(new Emitter);
        this.onDidChangeLoggers = this._onDidChangeLoggers.event;
        this._onDidChangeLogLevel = this._register(new Emitter);
        this.onDidChangeLogLevel = this._onDidChangeLogLevel.event;
        this._onDidChangeVisibility = this._register(new Emitter);
        this.onDidChangeVisibility = this._onDidChangeVisibility.event;
        if (loggerResources) {
            for (const loggerResource of loggerResources) {
                this._loggers.set(loggerResource.resource, { logger: undefined, info: loggerResource });
            }
        }
    }
    getLoggerEntry(resourceOrId) {
        if (isString(resourceOrId)) {
            return [...this._loggers.values()].find(logger => logger.info.id === resourceOrId);
        }
        return this._loggers.get(resourceOrId);
    }
    getLogger(resourceOrId) {
        return this.getLoggerEntry(resourceOrId)?.logger;
    }
    createLogger(idOrResource, options) {
        const resource = this.toResource(idOrResource);
        const id = isString(idOrResource) ? idOrResource : (options?.id ?? hash(resource.toString()).toString(16));
        let logger = this._loggers.get(resource)?.logger;
        const logLevel = options?.logLevel === 'always' ? LogLevel.Trace : options?.logLevel;
        if (!logger) {
            logger = this.doCreateLogger(resource, logLevel ?? this.getLogLevel(resource) ?? this.logLevel, { ...options, id });
        }
        const loggerEntry = {
            logger,
            info: { resource, id, logLevel, name: options?.name, hidden: options?.hidden, extensionId: options?.extensionId, when: options?.when }
        };
        this.registerLogger(loggerEntry.info);
        // TODO: @sandy081 Remove this once registerLogger can take ILogger
        this._loggers.set(resource, loggerEntry);
        return logger;
    }
    toResource(idOrResource) {
        return isString(idOrResource) ? joinPath(this.logsHome, `${idOrResource}.log`) : idOrResource;
    }
    setLogLevel(arg1, arg2) {
        if (URI.isUri(arg1)) {
            const resource = arg1;
            const logLevel = arg2;
            const logger = this._loggers.get(resource);
            if (logger && logLevel !== logger.info.logLevel) {
                logger.info.logLevel = logLevel === this.logLevel ? undefined : logLevel;
                logger.logger?.setLevel(logLevel);
                this._loggers.set(logger.info.resource, logger);
                this._onDidChangeLogLevel.fire([resource, logLevel]);
            }
        }
        else {
            this.logLevel = arg1;
            for (const [resource, logger] of this._loggers.entries()) {
                if (this._loggers.get(resource)?.info.logLevel === undefined) {
                    logger.logger?.setLevel(this.logLevel);
                }
            }
            this._onDidChangeLogLevel.fire(this.logLevel);
        }
    }
    setVisibility(resourceOrId, visibility) {
        const logger = this.getLoggerEntry(resourceOrId);
        if (logger && visibility !== !logger.info.hidden) {
            logger.info.hidden = !visibility;
            this._loggers.set(logger.info.resource, logger);
            this._onDidChangeVisibility.fire([logger.info.resource, visibility]);
        }
    }
    getLogLevel(resource) {
        let logLevel;
        if (resource) {
            logLevel = this._loggers.get(resource)?.info.logLevel;
        }
        return logLevel ?? this.logLevel;
    }
    registerLogger(resource) {
        const existing = this._loggers.get(resource.resource);
        if (existing) {
            if (existing.info.hidden !== resource.hidden) {
                this.setVisibility(resource.resource, !resource.hidden);
            }
        }
        else {
            this._loggers.set(resource.resource, { info: resource, logger: undefined });
            this._onDidChangeLoggers.fire({ added: [resource], removed: [] });
        }
    }
    deregisterLogger(resource) {
        const existing = this._loggers.get(resource);
        if (existing) {
            if (existing.logger) {
                existing.logger.dispose();
            }
            this._loggers.delete(resource);
            this._onDidChangeLoggers.fire({ added: [], removed: [existing.info] });
        }
    }
    *getRegisteredLoggers() {
        for (const entry of this._loggers.values()) {
            yield entry.info;
        }
    }
    getRegisteredLogger(resource) {
        return this._loggers.get(resource)?.info;
    }
    dispose() {
        this._loggers.forEach(logger => logger.logger?.dispose());
        this._loggers.clear();
        super.dispose();
    }
}
export class NullLogger {
    constructor() {
        this.onDidChangeLogLevel = new Emitter().event;
    }
    setLevel(level) { }
    getLevel() { return LogLevel.Info; }
    trace(message, ...args) { }
    debug(message, ...args) { }
    info(message, ...args) { }
    warn(message, ...args) { }
    error(message, ...args) { }
    critical(message, ...args) { }
    dispose() { }
    flush() { }
}
export class NullLogService extends NullLogger {
}
export function getLogLevel(environmentService) {
    if (environmentService.verbose) {
        return LogLevel.Trace;
    }
    if (typeof environmentService.logLevel === 'string') {
        const logLevel = parseLogLevel(environmentService.logLevel.toLowerCase());
        if (logLevel !== undefined) {
            return logLevel;
        }
    }
    return DEFAULT_LOG_LEVEL;
}
export function LogLevelToString(logLevel) {
    switch (logLevel) {
        case LogLevel.Trace: return 'trace';
        case LogLevel.Debug: return 'debug';
        case LogLevel.Info: return 'info';
        case LogLevel.Warning: return 'warn';
        case LogLevel.Error: return 'error';
        case LogLevel.Off: return 'off';
    }
}
export function parseLogLevel(logLevel) {
    switch (logLevel) {
        case 'trace':
            return LogLevel.Trace;
        case 'debug':
            return LogLevel.Debug;
        case 'info':
            return LogLevel.Info;
        case 'warn':
            return LogLevel.Warning;
        case 'error':
            return LogLevel.Error;
        case 'critical':
            return LogLevel.Error;
        case 'off':
            return LogLevel.Off;
    }
    return undefined;
}
// Contexts
export const CONTEXT_LOG_LEVEL = new RawContextKey('logLevel', LogLevelToString(LogLevel.Info));
