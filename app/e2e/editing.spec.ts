import { test, expect } from "@playwright/test";
import { F_RED, addArgument, headerBadge, openEdit, openWithBallUniverse, rowOf } from "./helpers";

// Edit-in-place: ids never change on save, so every reference follows the
// edit and the derived web reflows — including verdicts flipping.

test("editing a statement reflows every verdict that touched it", async ({ page }) => {
  await page.goto("/");
  await page.evaluate(() => localStorage.clear());
  await page.reload();
  await expect(headerBadge(page)).toHaveText("consistent", { timeout: 20_000 });
  await page.getByRole("button", { name: "Example" }).click();

  // The example derives "throws some of red" from the ball being red.
  await expect(
    rowOf(page, "the child throws some of red").getByText("⊨ true", { exact: true }),
  ).toBeVisible();

  // Repaint the ball: red → green ("blue" already exists in this universe).
  await openEdit(page, "the ball is red", "Editing “the ball is red”");
  await page.getByLabel("Predicate").fill("green");
  await page.getByRole("button", { name: "Save statement" }).click();

  // Same id, new meaning, everywhere at once: the row, the formulas that
  // reference it, and the derivation that depended on redness.
  await expect(rowOf(page, "the ball is green")).toBeVisible();
  await expect(
    rowOf(page, "(the ball is green IMPLIES the ball is not blue)"),
  ).toBeVisible();
  await expect(
    rowOf(page, "the child throws some of red").getByText("⊨ true", { exact: true }),
  ).toHaveCount(0);
});

test("editing a formula updates everything that references it", async ({ page }) => {
  await openWithBallUniverse(page);

  // not_play's argument s_play → s_red; F_BLUE references not_play and must
  // re-render through it.
  await openEdit(page, "NOT (all of the time to play is now)", "Editing formula");
  // The formula being edited is excluded from its own term choices (cycle guard).
  const formulaCard = page.locator(".card", { hasText: "Formulas" });
  await expect(
    formulaCard.locator("option", { hasText: "NOT (all of the time to play is now)" }),
  ).toHaveCount(0);
  await page.getByLabel("Term 1").selectOption({ label: "all of the ball is red" });
  await page.getByRole("button", { name: "Save formula" }).click();

  await expect(rowOf(page, "NOT (all of the ball is red)")).toBeVisible();
  await expect(
    rowOf(page, "(all of the ball is blue IMPLIES NOT (all of the ball is red))"),
  ).toBeVisible();
});

test("editing an argument recomputes its verdict", async ({ page }) => {
  await openWithBallUniverse(page);
  await addArgument(
    page,
    "Is it time to play?",
    [F_RED, "all of the ball is red"],
    "all of the time to play is now",
  );
  await expect(
    rowOf(page, "Is it time to play?").getByText("valid", { exact: true }),
  ).toBeVisible();

  await openEdit(page, "Is it time to play?", "Editing “Is it time to play?”");
  await page.getByLabel("Conclusion").selectOption({ label: "all of the ball is blue" });
  await page.getByRole("button", { name: "Save argument" }).click();

  await expect(
    rowOf(page, "Is it time to play?").getByText("invalid", { exact: true }),
  ).toBeVisible();
});
