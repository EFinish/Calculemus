import { expect, type Page } from "@playwright/test";

export const F_RED = "(all of the ball is red IMPLIES all of the time to play is now)";
export const F_BLUE = "(all of the ball is blue IMPLIES NOT (all of the time to play is now))";

export async function addStatement(page: Page, subject: string, predicate: string) {
  await page.getByLabel("Subject").fill(subject);
  await page.getByLabel("Predicate").fill(predicate);
  await page.getByRole("button", { name: "Add statement" }).click();
}

export async function addFormula(page: Page, op: string, terms: string[]) {
  await page.getByLabel("Connective").selectOption(op);
  for (let i = 0; i < terms.length; i++) {
    await page.getByLabel(`Term ${i + 1}`).selectOption({ label: terms[i] });
  }
  await page.getByRole("button", { name: "Add formula" }).click();
}

export async function addArgument(
  page: Page,
  title: string,
  premises: string[],
  conclusion: string,
) {
  await page.getByLabel("Argument title").fill(title);
  const argCard = page.locator(".card", { hasText: "Arguments" });
  for (const p of premises) {
    await argCard.locator(".premise", { hasText: p }).first().getByRole("checkbox").check();
  }
  await page.getByLabel("Conclusion").selectOption({ label: conclusion });
  await page.getByRole("button", { name: "Add argument" }).click();
}

// Statements are listed before formulas, so for a bare-statement text the
// first matching row is the statement even when a formula's rendering
// contains the same words.
export function rowOf(page: Page, text: string) {
  return page.locator(".row", { hasText: text }).first();
}

export async function setAssert(page: Page, text: string, on = true) {
  const box = rowOf(page, text).getByRole("checkbox");
  if (on) await box.check();
  else await box.uncheck();
}

export function headerBadge(page: Page) {
  return page.locator("header .badge").first();
}

// Fresh page with the ball universe composed and red→play, blue→¬play, red
// asserted — the shared starting point for milestone specs.
export async function openWithBallUniverse(page: Page) {
  await page.goto("/");
  await page.evaluate(() => localStorage.clear());
  await page.reload();
  await expect(headerBadge(page)).toHaveText("consistent", { timeout: 20_000 });

  await addStatement(page, "the ball", "red");
  await addStatement(page, "the ball", "blue");
  await addStatement(page, "the time to play", "now");

  await addFormula(page, "NOT", ["all of the time to play is now"]);
  await addFormula(page, "IMPLIES", [
    "all of the ball is red",
    "all of the time to play is now",
  ]);
  await addFormula(page, "IMPLIES", [
    "all of the ball is blue",
    "NOT (all of the time to play is now)",
  ]);

  await setAssert(page, F_RED);
  await setAssert(page, F_BLUE);
  await setAssert(page, "all of the ball is red");
  await expect(headerBadge(page)).toHaveText("consistent");
}
