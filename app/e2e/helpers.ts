import { expect, type Locator, type Page } from "@playwright/test";

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
  // Enforce the EXACT premise set: a mis-aimed click can check a neighboring
  // row (seen in the wild: a stray premise made a modus ponens argument stay
  // valid but lose its form badge). Sweep every box; check wanted, uncheck
  // strays. Premise labels render the exact ref text, so equality is safe.
  const labels = argCard.locator(".premise");
  for (let i = 0, n = await labels.count(); i < n; i++) {
    const label = labels.nth(i);
    const text = ((await label.textContent()) ?? "").trim();
    await ensureChecked(label.getByRole("checkbox"), premises.includes(text));
  }
  await page.getByLabel("Conclusion").selectOption({ label: conclusion });
  // Verify-and-retry: under parallel-suite load, async verdict badges can
  // reflow the library between Playwright's hit-test and its dispatch, so a
  // coordinate click occasionally lands beside the button.
  await expect(async () => {
    if ((await argCard.locator(".row", { hasText: title }).count()) === 0) {
      await page.getByRole("button", { name: "Add argument" }).click();
    }
    expect(await argCard.locator(".row", { hasText: title }).count()).toBeGreaterThan(0);
  }).toPass({ timeout: 15_000 });
}

// Statements are listed before formulas, so for a bare-statement text the
// first matching row is the statement even when a formula's rendering
// contains the same words.
export function rowOf(page: Page, text: string) {
  return page.locator(".row", { hasText: text }).first();
}

export async function setAssert(page: Page, text: string, on = true) {
  await ensureChecked(rowOf(page, text).getByRole("checkbox"), on);
}

// Controlled checkboxes re-render when async verdicts arrive; a coordinate
// click can race that re-render and get snapped back. Verify the outcome and
// retry instead of trusting a single click.
async function ensureChecked(box: Locator, want: boolean) {
  await expect(async () => {
    if ((await box.isChecked()) !== want) await box.click({ force: true });
    expect(await box.isChecked()).toBe(want);
  }).toPass({ timeout: 15_000 });
}

// Click a library row until the inspector shows the expected text. Row
// clicks toggle selection and can race re-renders, so verify the reaction.
export async function inspect(page: Page, rowText: string, expectText: string) {
  const inspector = page.locator(".card", { hasText: "Inspector" });
  await expect(async () => {
    if ((await inspector.getByText(expectText).count()) === 0) {
      await rowOf(page, rowText).click({ position: { x: 8, y: 8 } });
    }
    expect(await inspector.getByText(expectText).count()).toBeGreaterThan(0);
  }).toPass({ timeout: 15_000 });
}

// Click a row's edit button until its composer enters edit mode.
export async function openEdit(page: Page, rowText: string, noteText: string) {
  await expect(async () => {
    if ((await page.getByText(noteText).count()) === 0) {
      await rowOf(page, rowText).getByTitle("Edit").click();
    }
    expect(await page.getByText(noteText).count()).toBeGreaterThan(0);
  }).toPass({ timeout: 15_000 });
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
