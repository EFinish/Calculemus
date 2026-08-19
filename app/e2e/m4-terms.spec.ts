import { test, expect, type Page } from "@playwright/test";
import { F_RED, addArgument, addStatement, headerBadge, openWithBallUniverse, rowOf, setAssert } from "./helpers";

// The M4 dogfood from DESIGN §8: Barbara — all men are mortal… — validates
// from quantifier structure alone. Plus: term-level contradictions the
// engine couldn't see before, scenario editing, and named-form badges.

async function openEmpty(page: Page) {
  await page.goto("/");
  await page.evaluate(() => localStorage.clear());
  await page.reload();
  await expect(headerBadge(page)).toHaveText("consistent", { timeout: 20_000 });
}

test("Barbara validates from quantifier structure alone", async ({ page }) => {
  await openEmpty(page);
  // Three bare statements — no formulas, no conditionals anywhere.
  await addStatement(page, "men", "mortal");
  await addStatement(page, "greeks", "men");
  await addStatement(page, "greeks", "mortal");
  await addArgument(
    page,
    "Mortality of Greeks",
    ["all of men is mortal", "all of greeks is men"],
    "all of greeks is mortal",
  );
  const row = rowOf(page, "Mortality of Greeks");
  await expect(row.getByText("valid", { exact: true })).toBeVisible();
  await expect(row.getByText("Barbara", { exact: true })).toBeVisible();

  await row.click();
  const inspector = page.locator(".card", { hasText: "Inspector" });
  await expect(inspector.getByText("No countermodel exists")).toBeVisible();
  await expect(inspector.getByText("a recognized form")).toBeVisible();
});

test("term-level contradiction: all-red vs some-not-red", async ({ page }) => {
  await openEmpty(page);
  await addStatement(page, "the ball", "red"); // all of the ball is red
  // some of the ball is not red
  await page.getByLabel("Quantifier").selectOption("SOME");
  await page.getByLabel("Qualifier").selectOption("IS_NOT");
  await addStatement(page, "the ball", "red");

  await setAssert(page, "all of the ball is red");
  await setAssert(page, "some of the ball is not red");
  await expect(headerBadge(page)).toHaveText("contradictory");
  const verdicts = page.locator(".card", { hasText: "Verdicts" });
  await expect(verdicts.getByRole("button", { name: "unassert" })).toHaveCount(2);
});

test("modus ponens badge on the ball argument", async ({ page }) => {
  await openWithBallUniverse(page);
  await addArgument(
    page,
    "Is it time to play?",
    [F_RED, "all of the ball is red"],
    "all of the time to play is now",
  );
  await expect(
    rowOf(page, "Is it time to play?").getByText("modus ponens", { exact: true }),
  ).toBeVisible();
});

test("scenarios: edits stay in the counterfactual, base untouched", async ({ page }) => {
  await openWithBallUniverse(page);

  page.once("dialog", (d) => d.accept("blue world"));
  await page.getByRole("button", { name: "+ scenario" }).click();
  await expect(page.getByLabel("Scenario")).toHaveValue("blue world");

  // Toggling inside the scenario writes to its toggles — contradiction here…
  await setAssert(page, "all of the ball is blue");
  await expect(headerBadge(page)).toHaveText("contradictory");

  // …but the base universe never felt it.
  await page.getByLabel("Scenario").selectOption("");
  await expect(headerBadge(page)).toHaveText("consistent");
  await expect(
    rowOf(page, "all of the ball is blue").getByRole("checkbox"),
  ).not.toBeChecked();

  // Back in the scenario, the toggle is remembered — and it persists.
  await page.getByLabel("Scenario").selectOption("blue world");
  await expect(headerBadge(page)).toHaveText("contradictory");
  await page.reload();
  await expect(headerBadge(page)).toHaveText("consistent", { timeout: 20_000 });
  await page.getByLabel("Scenario").selectOption("blue world");
  await expect(headerBadge(page)).toHaveText("contradictory");
});
