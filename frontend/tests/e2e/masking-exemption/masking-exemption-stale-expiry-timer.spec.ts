import { test, expect } from "@playwright/test";

const PROJECT_ID = "project-sample";

// BUG: isExpired and expiryLabel in ExemptionGrantSection use Date.now()
// inside useMemo but don't include it in the dependency array. This means:
// - A grant showing "expires in 2 days" will show that label forever
//   until the page is reloaded, even after it actually expires.
// - The isExpired flag won't flip from false to true when the grant expires.
//
// Location: ProjectMaskingExemptionPage.tsx lines 1098-1119
//   const isExpired = useMemo(
//     () => !!grant.expirationTimestamp && grant.expirationTimestamp <= Date.now(),
//     [grant.expirationTimestamp]  // <-- Date.now() not in deps
//   );
//   const expiryLabel = useMemo(() => {
//     if (!grant.expirationTimestamp) return "";
//     const msRemaining = grant.expirationTimestamp - Date.now();  // <-- stale
//     ...
//   }, [grant.expirationTimestamp, t]);  // <-- Date.now() not in deps
//
// This test uses Playwright's clock API to verify the expiry label updates
// correctly when time crosses the expiration boundary.
test.describe("BUG: Stale expiry timer in grant section", () => {
  test.fixme(
    "expiry label should update from active to expired when time passes expiration",
    async ({ page }) => {
      // To properly test this bug, we need:
      // 1. A grant with an expiration timestamp a few seconds in the future
      // 2. Playwright clock.install() to control time
      // 3. Advance time past the expiration
      // 4. Verify the label changes from "expires in X" to "(Expired)"
      //
      // The bug: because useMemo deps don't include Date.now(), the label
      // won't update even when we advance time. The test should FAIL
      // against the current code, proving the bug exists.
      //
      // Implementation blocked by: need API setup to create a grant with
      // a near-future expiration. The UI enforces min date = today start,
      // so we can't create a grant that expires in seconds via the UI.
      //
      // When this is implemented and the bug is fixed, the test will pass,
      // preventing re-introduction of stale memo dependencies.

      // Pseudo-implementation:
      // await page.clock.install({ time: new Date('2026-04-08T12:00:00Z') });
      //
      // // Create grant via API expiring at 12:00:05Z (5 seconds from now)
      // await createGrantViaAPI({
      //   member: 'admin@bytebase.com',
      //   expiration: '2026-04-08T12:00:05Z',
      // });
      //
      // await page.goto(`/projects/${PROJECT_ID}/masking-exemption`);
      // await page.getByText('admin@bytebase.com').click();
      //
      // // Should show active state
      // await expect(page.getByText('expires today')).toBeVisible();
      // await expect(page.getByText('(Expired)')).not.toBeVisible();
      //
      // // Advance time past expiration
      // await page.clock.fastForward(10000); // 10 seconds
      //
      // // BUG: The label should update to "(Expired)" but won't because
      // // useMemo dependencies don't include Date.now()
      // await expect(page.getByText('(Expired)')).toBeVisible();
    }
  );
});
