/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { isIOS, isLinux, isMacintosh, isMobile, isWeb, isWindows } from '../../../base/common/platform.js';
import { localizeWithPath } from '../../../nls.js';
import { RawContextKey } from './contextkey.js';
export const IsMacContext = new RawContextKey('isMac', isMacintosh, localizeWithPath('vs/platform/contextkey/common/contextkeys', 'isMac', "Whether the operating system is macOS"));
export const IsLinuxContext = new RawContextKey('isLinux', isLinux, localizeWithPath('vs/platform/contextkey/common/contextkeys', 'isLinux', "Whether the operating system is Linux"));
export const IsWindowsContext = new RawContextKey('isWindows', isWindows, localizeWithPath('vs/platform/contextkey/common/contextkeys', 'isWindows', "Whether the operating system is Windows"));
export const IsWebContext = new RawContextKey('isWeb', isWeb, localizeWithPath('vs/platform/contextkey/common/contextkeys', 'isWeb', "Whether the platform is a web browser"));
export const IsMacNativeContext = new RawContextKey('isMacNative', isMacintosh && !isWeb, localizeWithPath('vs/platform/contextkey/common/contextkeys', 'isMacNative', "Whether the operating system is macOS on a non-browser platform"));
export const IsIOSContext = new RawContextKey('isIOS', isIOS, localizeWithPath('vs/platform/contextkey/common/contextkeys', 'isIOS', "Whether the operating system is iOS"));
export const IsMobileContext = new RawContextKey('isMobile', isMobile, localizeWithPath('vs/platform/contextkey/common/contextkeys', 'isMobile', "Whether the platform is a mobile web browser"));
export const IsDevelopmentContext = new RawContextKey('isDevelopment', false, true);
export const ProductQualityContext = new RawContextKey('productQualityType', '', localizeWithPath('vs/platform/contextkey/common/contextkeys', 'productQualityType', "Quality type of VS Code"));
export const InputFocusedContextKey = 'inputFocus';
export const InputFocusedContext = new RawContextKey(InputFocusedContextKey, false, localizeWithPath('vs/platform/contextkey/common/contextkeys', 'inputFocus', "Whether keyboard focus is inside an input box"));
