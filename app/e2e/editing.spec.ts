import { test, expect, type Page } from "@playwright/test";
import { F_RED, addArgument, headerBadge, openWithBallUniverse, rowOf } from "./helpers";

// Edit-in-place: ids never change on save, so every reference follows the
// edit and the derived web reflows — including verdicts flipping.

function editButton(page: Page, rowText: string) {
  return rowOf(page, rowText).getByTitle("Edit");
}

test("editing a statement reflows every verdict that touched it", async ({ page }) => {
  await page.goto("/");
  await page.evaluate(() => localStorage.clear());
  await page.reload();
  await expect(headerBadge(page)).toHaveText("consistent", { timeout: 20_000 });
  await page.getByRole("button", { name: "Example" }).click();

  // The Frege universe derives "throws some of red" and validates the argument.
  await expect(
    rowOf(page, "the boy throws some of red").getByText("⊨ true", { exact: true }),
  ).toBeVisible();
  await expect(rowOf(page, "The Frege step").getByText("valid", { exact: true })).toBeVisible();

  // Repaint the ball: red → blue.
  await editButton(page, "the ball is red").click();
  await expect(page.getByText("Editing “the ball is red”")).toBeVisible();
  await page.getByLabel("Predicate").fill("blue");
  await page.getByRole("button", { name: "Save statement" }).click();

  // Same id, new meaning, everywhere at once: the row, the argument's
  // premise list, the derivation, and the verdict.
  await expect(rowOf(page, "the ball is blue")).toBeVisible();
  await expect(rowOf(page, "The Frege step")).toContainText("the ball is blue");
  await expect(
    rowOf(page, "the boy throws some of red").getByText("⊨ true", { exact: true }),
  ).toHaveCount(0);
  await expect(rowOf(page, "The Frege step").getByText("invalid", { exact: true })).toBeVisible();
});

test("editing a formula updates everything that references it", async ({ page }) => {
  await openWithBallUniverse(page);

  // not_play's argument s_play → s_red; F_BLUE references not_play and must
  // re-render through it.
  await editButton(page, "NOT (all of the time to play is now)").click();
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

  await editButton(page, "Is it time to play?").click();
  await expect(page.getByText("Editing “Is it time to play?”")).toBeVisible();
  await page.getByLabel("Conclusion").selectOption({ label: "all of the ball is blue" });
  await page.getByRole("button", { name: "Save argument" }).click();

  await expect(
    rowOf(page, "Is it time to play?").getByText("invalid", { exact: true }),
  ).toBeVisible();
});
