import { test, expect } from "@playwright/test";
import {
  F_RED,
  addArgument,
  headerBadge,
  openWithBallUniverse,
  rowOf,
  setAssert,
} from "./helpers";

// The M2 dogfood from DESIGN.md §8: deliberately assert a contradiction; the
// app names the exact statements at fault and stops pretending to derive —
// and now the diagnosis is actionable, and invalid arguments show their
// countermodel.

test("diagnosis names the fight and unassert resolves it", async ({ page }) => {
  await openWithBallUniverse(page);

  await setAssert(page, "all of the ball is blue");
  await expect(headerBadge(page)).toHaveText("contradictory");

  const verdicts = page.locator(".card", { hasText: "Verdicts" });
  await expect(page.getByText("cannot coexist")).toBeVisible();
  await expect(verdicts.getByRole("button", { name: "unassert" })).toHaveCount(4);
  // Derivation is suspended (explosion guard): the suspension note shows and
  // no derived-truth rows render.
  await expect(verdicts.getByText("suspended")).toBeVisible();
  await expect(verdicts.getByText("⊨ true")).toHaveCount(0);

  // Drop one member of the core from inside the diagnosis view — the bare
  // blue statement, not the F_BLUE conditional that also mentions it.
  await verdicts
    .locator(".row", { hasText: "all of the ball is blue" })
    .filter({ hasNotText: "IMPLIES" })
    .getByRole("button", { name: "unassert" })
    .click();
  await expect(headerBadge(page)).toHaveText("consistent");
  await expect(verdicts.getByText("forced by your assertions")).toBeVisible();
});

test("invalid argument shows its countermodel in the inspector", async ({ page }) => {
  await openWithBallUniverse(page);

  await addArgument(
    page,
    "Affirming the consequent",
    [F_RED, "all of the time to play is now"],
    "all of the ball is red",
  );
  const row = rowOf(page, "Affirming the consequent");
  await expect(row.getByText("invalid", { exact: true })).toBeVisible();

  await row.click();
  const inspector = page.locator(".card", { hasText: "Inspector" });
  await expect(inspector.getByText("every premise holds and the conclusion fails")).toBeVisible();
  const countermodel = inspector.locator(".countermodel");
  // The witness world: play true, red false (premises hold, conclusion fails).
  await expect(
    countermodel.locator("tr", { hasText: "all of the time to play is now" }).locator(".tv"),
  ).toHaveText("true");
  await expect(
    countermodel.locator("tr", { hasText: "all of the ball is red" }).locator(".tv"),
  ).toHaveText("false");
});

test("inspector explains truth states and vacuous conditionals", async ({ page }) => {
  await openWithBallUniverse(page);

  // Entailed-false statement: blue (modus tollens).
  await rowOf(page, "all of the ball is blue").click();
  const inspector = page.locator(".card", { hasText: "Inspector" });
  await expect(inspector.getByText("force this false")).toBeVisible();
  await expect(inspector.getByText("Used by")).toBeVisible();

  // Vacuous conditional: blue→¬play, explained.
  await rowOf(page, "(all of the ball is blue IMPLIES").click();
  await expect(inspector.getByText("holds only vacuously")).toBeVisible();

  // Valid argument: no-countermodel explanation, and its chains listed.
  await addArgument(
    page,
    "Is it time to play?",
    [F_RED, "all of the ball is red"],
    "all of the time to play is now",
  );
  await addArgument(
    page,
    "Addition",
    ["all of the time to play is now"],
    "all of the time to play is now",
  );
  await rowOf(page, "Is it time to play?").click();
  await expect(inspector.getByText("No countermodel exists")).toBeVisible();
  await expect(inspector.getByText("feeds")).toBeVisible();
});
