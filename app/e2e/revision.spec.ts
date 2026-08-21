import { test, expect } from "@playwright/test";
import { openWithBallUniverse, inspect, headerBadge } from "./helpers";

// Belief revision (DESIGN.md §12) + entails edges (§5), dogfooded through
// the ball universe: red + (red → now) force "now" true; denying it must
// cost exactly one of the two, and the canvas must draw the entailments.

test("entails edges draw on the canvas and filter off", async ({ page }) => {
  await openWithBallUniverse(page);
  // now ⊨ (red → now) and NOT(now) ⊨ (blue → ¬now); nothing else pairwise.
  const entails = page.locator(".vue-flow__edge.e-entails");
  await expect(entails).toHaveCount(2);
  await page.locator("label", { hasText: "entails" }).getByRole("checkbox").uncheck();
  await expect(entails).toHaveCount(0);
});

test("believe otherwise: minimal retractions offered and applied", async ({ page }) => {
  await openWithBallUniverse(page);

  await inspect(page, "all of the time to play is now", "⊨ true");
  const inspector = page.locator(".card", { hasText: "Inspector" });
  await expect(inspector.getByText("Believe otherwise")).toBeVisible();

  // Two minimal prices: give up "the ball is red", or give up the conditional.
  const options = inspector.locator(".retraction");
  await expect(options).toHaveCount(2);

  // Pay the statement price (the option without the conditional in it).
  await options
    .filter({ hasNotText: "IMPLIES" })
    .first()
    .getByRole("button", { name: "give up" })
    .click();

  // The forced truth is released, nothing broke, and the panel withdraws.
  await expect(inspector.locator(".badge", { hasText: "undetermined" })).toBeVisible();
  await expect(headerBadge(page)).toHaveText("consistent");
  await expect(inspector.getByText("Believe otherwise")).toHaveCount(0);
});
