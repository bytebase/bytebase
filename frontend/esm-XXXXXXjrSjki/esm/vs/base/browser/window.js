/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
export function ensureCodeWindow(targetWindow, fallbackWindowId) {
    const codeWindow = targetWindow;
    if (typeof codeWindow.vscodeWindowId !== 'number') {
        Object.defineProperty(codeWindow, 'vscodeWindowId', {
            get: () => fallbackWindowId
        });
    }
}
// eslint-disable-next-line no-restricted-globals
export const mainWindow = window;
/**
 * @deprecated to support multi-window scenarios, use `DOM.mainWindow`
 * if you target the main global window or use helpers such as `DOM.getWindow()`
 * or `DOM.getActiveWindow()` to obtain the correct window for the context you are in.
 */
export const $window = mainWindow;
export function isAuxiliaryWindow(obj) {
    if (obj === mainWindow) {
        return false;
    }
    const candidate = obj;
    return typeof candidate?.vscodeWindowId === 'number';
}
