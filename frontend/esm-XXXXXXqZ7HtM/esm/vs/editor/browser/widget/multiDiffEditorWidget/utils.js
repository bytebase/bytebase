/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { ActionRunner } from '../../../../base/common/actions.js';
export class ActionRunnerWithContext extends ActionRunner {
    constructor(_getContext) {
        super();
        this._getContext = _getContext;
    }
    runAction(action, _context) {
        return super.runAction(action, this._getContext());
    }
}
