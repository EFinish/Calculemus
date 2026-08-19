import { test, expect, type Page } from "@playwright/test";

// The M1 dogfood test from DESIGN.md §8, automated: build a universe in the
// browser, watch the engine derive verdicts live, reload the tab, and find
// everything still there. This is the test no previous version of this tool
// could pass.

const F_RED = "(all of the ball is red IMPLIES all of the time to play is now)";
const F_BLUE = "(all of the ball is blue IMPLIES NOT (all of the time to play is now))";

async function addStatement(page: Page, subject: string, predicate: string) {
  await page.getByLabel("Subject").fill(subject);
  await page.getByLabel("Predicate").fill(predicate);
  await page.getByRole("button", { name: "Add statement" }).click();
}

async function addFormula(page: Page, op: string, terms: string[]) {
  await page.getByLabel("Connective").selectOption(op);
  for (let i = 0; i < terms.length; i++) {
    await page.getByLabel(`Term ${i + 1}`).selectOption({ label: terms[i] });
  }
  await page.getByRole("button", { name: "Add formula" }).click();
}

// Statements are listed before formulas, so for a bare-statement text the
// first matching row is the statement even when a formula's rendering
// contains the same words.
function rowOf(page: Page, text: string) {
  return page.locator(".row", { hasText: text }).first();
}

async function setAssert(page: Page, text: string, on = true) {
  const box = rowOf(page, text).getByRole("checkbox");
  if (on) await box.check();
  else await box.uncheck();
}

function headerBadge(page: Page) {
  return page.locator("header .badge").first();
}

test("M1 dogfood: compose, derive, contradict, reload, persist", async ({ page }) => {
  await page.goto("/");
  await page.evaluate(() => localStorage.clear());
  await page.reload();

  // Engine boots via WASM; an empty universe is consistent.
  await expect(headerBadge(page)).toHaveText("consistent", { timeout: 20_000 });

  await page.getByLabel("Universe title").fill("Playtime");

  // Compose the ball universe through the guided composer — no raw syntax.
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

  // Assert red→play, blue→¬play, red. The engine must derive play (modus
  // ponens) and ¬blue (modus tollens) — no rule is coded anywhere.
  await setAssert(page, F_RED);
  await setAssert(page, F_BLUE);
  await setAssert(page, "all of the ball is red");

  await expect(headerBadge(page)).toHaveText("consistent");
  const verdicts = page.locator(".card", { hasText: "Verdicts" });
  await expect(
    verdicts.locator(".row", { hasText: "all of the time to play is now" }).filter({ hasText: "⊨ true" }),
  ).toBeVisible();
  await expect(
    verdicts.locator(".row", { hasText: "all of the ball is blue" }).filter({ hasText: "⊨ false" }),
  ).toBeVisible();
  // blue→¬play holds only because blue is forced false — labeled vacuous.
  await expect(rowOf(page, F_BLUE).getByText("vacuous", { exact: true })).toBeVisible();

  // Contradict on purpose: assert blue as well → diagnosis replaces derivation.
  await setAssert(page, "all of the ball is blue");
  await expect(headerBadge(page)).toHaveText("contradictory");
  await expect(page.getByText("cannot coexist")).toBeVisible();
  await setAssert(page, "all of the ball is blue", false);
  await expect(headerBadge(page)).toHaveText("consistent");

  // Build the modus ponens argument; validity is computed live.
  await page.getByLabel("Argument title").fill("Is it time to play?");
  const argCard = page.locator(".card", { hasText: "Arguments" });
  await argCard.locator(".premise", { hasText: F_RED }).getByRole("checkbox").check();
  await argCard
    .locator(".premise", { hasText: "all of the ball is red" })
    .first()
    .getByRole("checkbox")
    .check();
  await page.getByLabel("Conclusion").selectOption({ label: "all of the time to play is now" });
  await page.getByRole("button", { name: "Add argument" }).click();
  await expect(rowOf(page, "Is it time to play?").getByText("valid", { exact: true })).toBeVisible();

  // THE dogfood moment: close the tab, reopen — everything is still there.
  await page.reload();
  await expect(headerBadge(page)).toHaveText("consistent", { timeout: 20_000 });
  await expect(page.getByLabel("Universe title")).toHaveValue("Playtime");
  await expect(rowOf(page, F_BLUE)).toBeVisible();
  await expect(rowOf(page, "all of the ball is red").getByRole("checkbox")).toBeChecked();
  await expect(rowOf(page, "Is it time to play?").getByText("valid", { exact: true })).toBeVisible();
  await expect(
    verdicts.locator(".row", { hasText: "all of the time to play is now" }).filter({ hasText: "⊨ true" }),
  ).toBeVisible();
});
