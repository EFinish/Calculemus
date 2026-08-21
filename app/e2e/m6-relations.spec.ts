import { test, expect, type Page } from "@playwright/test";
import { addArgument, headerBadge, inspect, rowOf, setAssert } from "./helpers";

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

test("the Example button loads the child-and-ball universe", async ({ page }) => {
  await openEmpty(page);
  await page.getByRole("button", { name: "Example" }).click();
  await expect(page.getByLabel("Universe title")).toHaveValue("The child and the ball");
  await expect(headerBadge(page)).toHaveText("consistent");
  await expect(page.getByText("worlds ≤ 6 things")).toBeVisible();

  // Derived, not asserted: the relational payoff…
  await expect(
    rowOf(page, "the child throws some of red").getByText("⊨ true", { exact: true }),
  ).toBeVisible();
  // …and the IS_NOT twin linking automatically to its positive statement.
  await expect(
    rowOf(page, "the ball is not red").getByText("⊨ false", { exact: true }),
  ).toBeVisible();
  // The blue→not-red conditional is vacuous (blue is forced false).
  await expect(
    rowOf(page, "(the ball is blue IMPLIES the ball is not red)").getByText("vacuous", { exact: true }),
  ).toBeVisible();
  await expect(rowOf(page, "test 1").getByText("valid", { exact: true })).toBeVisible();
  await expect(rowOf(page, "test 2").getByText("valid", { exact: true })).toBeVisible();

  // Repaint the ball blue → the exclusion pair fights: diagnosed core.
  await setAssert(page, "the ball is blue");
  await expect(headerBadge(page)).toHaveText("contradictory");
  await setAssert(page, "the ball is blue", false);
  await expect(headerBadge(page)).toHaveText("consistent");

  // Replacing a non-empty universe asks first.
  page.once("dialog", (d) => d.accept());
  await page.getByRole("button", { name: "Example" }).click();
  await expect(page.getByLabel("Universe title")).toHaveValue("The child and the ball");
});

test("'is' typed as a verb is refused and redirected to the copula", async ({ page }) => {
  await openEmpty(page);
  await page.getByLabel("Quantifier", { exact: true }).selectOption("THE");
  await page.getByLabel("Subject").fill("ball");
  await page.getByLabel("Qualifier").selectOption("DOES");
  await page.getByLabel("Verb").fill("is");
  await page.getByLabel("Predicate").fill("red");
  await expect(page.getByText("uninterpreted relation")).toBeVisible();
  await expect(page.getByRole("button", { name: "Add statement" })).toBeDisabled();

  // A real verb clears the warning; the copula path works as ever.
  await page.getByLabel("Verb").fill("throw");
  await expect(page.getByText("uninterpreted relation")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Add statement" })).toBeEnabled();
  await page.getByLabel("Qualifier").selectOption("IS");
  await page.getByRole("button", { name: "Add statement" }).click();
  await expect(page.locator(".row", { hasText: "the ball is red" }).first()).toBeVisible();
});

test("already-conjugated verbs are not double-conjugated", async ({ page }) => {
  await openEmpty(page);
  await page.getByLabel("Quantifier", { exact: true }).selectOption("THE");
  await page.getByLabel("Subject").fill("child");
  await page.getByLabel("Qualifier").selectOption("DOES");
  await page.getByLabel("Verb").fill("throws");
  await page.getByLabel("Object quantifier").selectOption("SOME");
  await page.getByLabel("Predicate").fill("object");
  await expect(page.getByText("“the child throws some of object”")).toBeVisible();
});
