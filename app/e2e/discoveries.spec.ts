import { test, expect, type Page } from "@playwright/test";
import { headerBadge, rowOf, setAssert } from "./helpers";

// Saturation: the engine proposes atomic statements the assertions force but
// nobody authored; the user adopts them with one click. Born from the
// question "wouldn't it be possible to calculate that Jeff the Lion is a
// mammal?" — yes, and now the app volunteers it.

async function openEmpty(page: Page) {
  await page.goto("/");
  await page.evaluate(() => localStorage.clear());
  await page.reload();
  await expect(headerBadge(page)).toHaveText("consistent", { timeout: 20_000 });
}

test("Jeff the Lion is discovered to be a mammal, and adopted", async ({ page }) => {
  await openEmpty(page);

  // all of lions is mammals
  await page.getByLabel("Subject").fill("lions");
  await page.getByLabel("Predicate").fill("mammals");
  await page.getByRole("button", { name: "Add statement" }).click();
  // the Jeff the Lion is lions
  await page.getByLabel("Quantifier", { exact: true }).selectOption("THE");
  await page.getByLabel("Subject").fill("Jeff the Lion");
  await page.getByLabel("Predicate").fill("lions");
  await page.getByRole("button", { name: "Add statement" }).click();

  await setAssert(page, "all of lions is mammals");
  await setAssert(page, "the Jeff the Lion is lions");

  const verdicts = page.locator(".card", { hasText: "Verdicts" });
  await expect(verdicts.getByText("Discoveries")).toBeVisible();
  const discovery = verdicts.locator(".row", { hasText: "the jeff the lion is mammals" });
  await expect(discovery).toBeVisible();

  // Adopt: it becomes a real statement carrying its ⊨ badge, and the
  // proposal disappears.
  await discovery.getByRole("button", { name: "+" }).click();
  await expect(
    rowOf(page, "the jeff the lion is mammals").getByText("⊨ true", { exact: true }),
  ).toBeVisible({ timeout: 15_000 });
  // …while the derived-truths list may now show the adopted statement, no
  // PROPOSAL row (one bearing an adopt button) remains for it.
  await expect(
    verdicts
      .locator(".row", { hasText: "the jeff the lion is mammals" })
      .filter({ has: page.getByRole("button", { name: "+" }) }),
  ).toHaveCount(0);
});

test("negative discoveries: the ball is not blue", async ({ page }) => {
  await openEmpty(page);

  await page.getByLabel("Quantifier", { exact: true }).selectOption("THE");
  await page.getByLabel("Subject").fill("ball");
  await page.getByLabel("Predicate").fill("red");
  await page.getByRole("button", { name: "Add statement" }).click();

  await page.getByLabel("Quantifier", { exact: true }).selectOption("NONE");
  await page.getByLabel("Subject").fill("red");
  await page.getByLabel("Predicate").fill("blue");
  await page.getByRole("button", { name: "Add statement" }).click();

  await setAssert(page, "the ball is red");
  await setAssert(page, "none of red is blue");

  const verdicts = page.locator(".card", { hasText: "Verdicts" });
  await expect(verdicts.locator(".row", { hasText: "the ball is not blue" })).toBeVisible();
});
