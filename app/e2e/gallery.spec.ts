import { test, expect, type Page } from "@playwright/test";
import { headerBadge, rowOf } from "./helpers";

// The Try-me gallery: each universe demonstrates a different engine layer
// with a famous argument or gotcha.

async function openGallery(page: Page, name: string) {
  await page.goto("/");
  await page.evaluate(() => localStorage.clear());
  await page.reload();
  await expect(headerBadge(page)).toHaveText("consistent", { timeout: 20_000 });
  await page.getByLabel("Try me").selectOption(name);
  await expect(page.getByLabel("Universe title")).toHaveValue(name);
}

test("Socrates: the syllogism derives, the converse trap fails", async ({ page }) => {
  await openGallery(page, "Socrates is mortal");
  await expect(
    rowOf(page, "Socrates is mortal").getByText("⊨ true", { exact: true }),
  ).toBeVisible();
  await expect(
    rowOf(page, "Barbara's little brother").getByText("valid", { exact: true }),
  ).toBeVisible();
  await expect(
    rowOf(page, "The converse trap").getByText("invalid", { exact: true }),
  ).toBeVisible();
});

test("Carroll's crocodile validates across four terms", async ({ page }) => {
  await openGallery(page, "Carroll's crocodile");
  await expect(
    rowOf(page, "no baby can manage a crocodile").getByText("⊨ true", { exact: true }),
  ).toBeVisible();
  await expect(
    rowOf(page, "Carroll's crocodile (Symbolic Logic, 1896)").getByText("valid", { exact: true }),
  ).toBeVisible();
});

test("ponens and tollens carry form badges; the impostors are invalid", async ({ page }) => {
  await openGallery(page, "Ponens and impostors");
  await expect(
    rowOf(page, "the real thing").getByText("modus ponens", { exact: true }),
  ).toBeVisible();
  await expect(
    rowOf(page, "also real").getByText("modus tollens", { exact: true }),
  ).toBeVisible();
  await expect(
    rowOf(page, "Affirming the consequent").getByText("invalid", { exact: true }),
  ).toBeVisible();
  await expect(
    rowOf(page, "Denying the antecedent").getByText("invalid", { exact: true }),
  ).toBeVisible();
});

test("the unicorn problem: Darapti fails until a unicorn exists", async ({ page }) => {
  await openGallery(page, "The unicorn problem");
  await expect(
    rowOf(page, "Darapti").getByText("invalid", { exact: true }),
  ).toBeVisible();
  await expect(
    rowOf(page, "Grant one unicorn").getByText("valid", { exact: true }),
  ).toBeVisible();
});

test("Russell's barber: a one-sentence contradiction, and explosion", async ({ page }) => {
  await openGallery(page, "Russell's barber");
  // The paradox's engine: p ↔ ¬p, asserted. The universe is contradictory
  // with a minimal core of exactly ONE assertion.
  await expect(headerBadge(page)).toHaveText("contradictory");
  const verdicts = page.locator(".card", { hasText: "Verdicts" });
  await expect(verdicts.getByRole("button", { name: "unassert" })).toHaveCount(1);
  // Ex falso quodlibet: from the impossible barber, the moon is cheese.
  await expect(
    rowOf(page, "Ex falso quodlibet").getByText("valid", { exact: true }),
  ).toBeVisible();
  // Dissolve the paradox: unassert the rule — no such barber exists.
  await verdicts.getByRole("button", { name: "unassert" }).click();
  await expect(headerBadge(page)).toHaveText("consistent");
});
