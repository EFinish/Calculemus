import { test, expect, type Page } from "@playwright/test";
import { addArgument, headerBadge, rowOf, setAssert } from "./helpers";

// The M6 dogfood (DESIGN §11): the boy/ball/red example that motivated the
// Frege step, run through the UI. No conditionals anywhere — the connection
// between the object of one statement and the subject of another is now
// carried by the individual "the ball" itself.

async function openEmpty(page: Page) {
  await page.goto("/");
  await page.evaluate(() => localStorage.clear());
  await page.reload();
  await expect(headerBadge(page)).toHaveText("consistent", { timeout: 20_000 });
}

async function addRelational(
  page: Page,
  subjectMode: string,
  subject: string,
  verbMode: string,
  verb: string,
  objectMode: string,
  object: string,
) {
  await page.getByLabel("Quantifier", { exact: true }).selectOption(subjectMode);
  await page.getByLabel("Subject").fill(subject);
  await page.getByLabel("Qualifier").selectOption(verbMode);
  if (verb) await page.getByLabel("Verb").fill(verb);
  if (verbMode.startsWith("DOES")) {
    await page.getByLabel("Object quantifier").selectOption(objectMode);
  }
  await page.getByLabel("Predicate").fill(object);
  await page.getByRole("button", { name: "Add statement" }).click();
}

test("the Frege step: object of one statement meets subject of another", async ({ page }) => {
  await openEmpty(page);

  await addRelational(page, "THE", "boy", "DOES", "throw", "THE", "ball");
  await addRelational(page, "THE", "ball", "IS", "", "", "red");
  await addRelational(page, "THE", "boy", "DOES", "throw", "SOME", "red");

  // Assert the two facts; the third must be DERIVED — the inference the
  // Boolean edition needed a hand-written conditional for.
  await setAssert(page, "the boy throws the ball");
  await setAssert(page, "the ball is red");
  await expect(headerBadge(page)).toHaveText("consistent");
  await expect(
    rowOf(page, "the boy throws some of red").getByText("⊨ true", { exact: true }),
  ).toBeVisible();

  // Relational semantics are bounded and say so: 2 individuals + 4 witnesses.
  await expect(page.getByText("worlds ≤ 6 things")).toBeVisible();

  // The argument, valid with no formulas in the universe at all.
  await addArgument(
    page,
    "The Frege step",
    ["the boy throws the ball", "the ball is red"],
    "the boy throws some of red",
  );
  await expect(
    rowOf(page, "The Frege step").getByText("valid", { exact: true }),
  ).toBeVisible();
});

test("relational contradiction is diagnosed", async ({ page }) => {
  await openEmpty(page);

  await addRelational(page, "THE", "boy", "DOES", "throw", "THE", "ball");
  await addRelational(page, "THE", "ball", "IS", "", "", "red");
  await addRelational(page, "THE", "boy", "DOES", "throw", "NONE", "red");

  await setAssert(page, "the boy throws the ball");
  await setAssert(page, "the ball is red");
  await setAssert(page, "the boy throws none of red");

  await expect(headerBadge(page)).toHaveText("contradictory");
  const verdicts = page.locator(".card", { hasText: "Verdicts" });
  await expect(verdicts.getByRole("button", { name: "unassert" })).toHaveCount(3);
});

test("quantified verbs: all men throw some ball, socrates is a man", async ({ page }) => {
  await openEmpty(page);

  await addRelational(page, "ALL", "men", "DOES", "throw", "SOME", "ball");
  await addRelational(page, "THE", "socrates", "IS", "", "", "men");
  await addRelational(page, "THE", "socrates", "DOES", "throw", "SOME", "ball");

  await addArgument(
    page,
    "Socrates plays too",
    ["all of men throws some of ball", "the socrates is men"],
    "the socrates throws some of ball",
  );
  await expect(
    rowOf(page, "Socrates plays too").getByText("valid", { exact: true }),
  ).toBeVisible();
});

test("the Example button loads the Frege universe ready to explore", async ({ page }) => {
  await openEmpty(page);
  await page.getByRole("button", { name: "Example" }).click();
  await expect(page.getByLabel("Universe title")).toHaveValue("The Frege step");
  await expect(headerBadge(page)).toHaveText("consistent");
  await expect(
    rowOf(page, "the boy throws some of red").getByText("⊨ true", { exact: true }),
  ).toBeVisible();
  await expect(
    rowOf(page, "The Frege step").getByText("valid", { exact: true }),
  ).toBeVisible();
  await expect(page.getByText("worlds ≤ 6 things")).toBeVisible();

  // The bundled counterfactual: assert throws-none-red → contradiction.
  await page.getByLabel("Scenario").selectOption("throws nothing red");
  await expect(headerBadge(page)).toHaveText("contradictory");

  // Replacing a non-empty universe asks first.
  page.once("dialog", (d) => d.accept());
  await page.getByRole("button", { name: "Example" }).click();
  await expect(page.getByLabel("Universe title")).toHaveValue("The Frege step");
});
