import { test, expect } from "@playwright/test";
import { headerBadge, openWithBallUniverse, rowOf, setAssert } from "./helpers";

// The M5 dogfood from DESIGN §8: send a universe URL to a friend; they can
// explore your scenarios. Shared links are immutable read-only snapshots;
// the viewer's own saved universe is never touched.

test("share a universe, explore it read-only, copy to workspace", async ({ page }) => {
  await openWithBallUniverse(page);

  // A scenario for the recipient to explore.
  page.once("dialog", (d) => d.accept("blue too"));
  await page.getByRole("button", { name: "+ scenario" }).click();
  await setAssert(page, "all of the ball is blue");
  await expect(headerBadge(page)).toHaveText("contradictory");
  await page.getByLabel("Scenario").selectOption("");
  await expect(headerBadge(page)).toHaveText("consistent");

  // Publish. The link lands in a visible input (clipboard is best-effort).
  await page.getByRole("button", { name: "Share" }).click();
  const link = await page.getByLabel("Share link").inputValue();
  expect(link).toMatch(/\?u=[a-z0-9]{10}$/);

  // The friend opens the link (same browser here — which also proves the
  // viewer's own universe survives, checked at the end).
  await page.goto(link);
  await expect(page.getByText("shared · read-only")).toBeVisible();
  await expect(headerBadge(page)).toHaveText("consistent", { timeout: 20_000 });
  await expect(rowOf(page, "all of the ball is red")).toBeVisible();

  // Read-only: no composers, no edits.
  await expect(page.getByRole("button", { name: "Add statement" })).toHaveCount(0);
  await expect(
    rowOf(page, "all of the ball is red").getByRole("checkbox"),
  ).toBeDisabled();

  // …but the shared scenarios are explorable.
  await page.getByLabel("Scenario").selectOption("blue too");
  await expect(headerBadge(page)).toHaveText("contradictory");
  await page.getByLabel("Scenario").selectOption("");
  await expect(headerBadge(page)).toHaveText("consistent");

  // Adopt it: editing returns and it becomes the viewer's saved universe.
  await page.getByRole("button", { name: "Copy to my workspace" }).click();
  await expect(page.getByRole("button", { name: "Add statement" })).toBeVisible();
  await expect(page.getByText("shared · read-only")).toHaveCount(0);
  await page.reload();
  await expect(headerBadge(page)).toHaveText("consistent", { timeout: 20_000 });
  await expect(rowOf(page, "all of the ball is red")).toBeVisible();
});

test("viewing a shared link does not clobber the viewer's own universe", async ({ page }) => {
  await openWithBallUniverse(page);
  await page.getByLabel("Universe title").fill("My precious universe");

  await page.getByRole("button", { name: "Share" }).click();
  const link = await page.getByLabel("Share link").inputValue();

  await page.goto(link);
  await expect(page.getByText("shared · read-only")).toBeVisible();

  // Leaving the shared view restores the viewer's own saved universe.
  await page.goto("/");
  await expect(headerBadge(page)).toHaveText("consistent", { timeout: 20_000 });
  await expect(page.getByLabel("Universe title")).toHaveValue("My precious universe");
  await expect(page.getByText("shared · read-only")).toHaveCount(0);
});

test("stale share links fail gracefully", async ({ page }) => {
  await page.goto("/?u=zzzzzzzzzz");
  await expect(page.getByText("doesn't exist")).toBeVisible({ timeout: 20_000 });
});
