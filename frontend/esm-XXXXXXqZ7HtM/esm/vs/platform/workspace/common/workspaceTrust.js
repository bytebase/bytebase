/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { localizeWithPath } from '../../../nls.js';
import { createDecorator } from '../../instantiation/common/instantiation.js';
export var WorkspaceTrustScope;
(function (WorkspaceTrustScope) {
    WorkspaceTrustScope[WorkspaceTrustScope["Local"] = 0] = "Local";
    WorkspaceTrustScope[WorkspaceTrustScope["Remote"] = 1] = "Remote";
})(WorkspaceTrustScope || (WorkspaceTrustScope = {}));
export function workspaceTrustToString(trustState) {
    if (trustState) {
        return localizeWithPath('vs/platform/workspace/common/workspaceTrust', 'trusted', "Trusted");
    }
    else {
        return localizeWithPath('vs/platform/workspace/common/workspaceTrust', 'untrusted', "Restricted Mode");
    }
}
export const IWorkspaceTrustEnablementService = createDecorator('workspaceTrustEnablementService');
export const IWorkspaceTrustManagementService = createDecorator('workspaceTrustManagementService');
export const IWorkspaceTrustRequestService = createDecorator('workspaceTrustRequestService');
