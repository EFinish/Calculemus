import { test, expect } from "@playwright/test";
import { F_RED, addArgument, openWithBallUniverse } from "./helpers";

// The M3 dogfood from DESIGN.md §8: chain two arguments and watch the chains
// edge appear without drawing it. Plus: truth states styled on nodes, edge
// filters, canvas↔inspector selection sync, and layout that survives reload.

test("the web draws itself: chains edge appears, nodes carry truth state", async ({ page }) => {
  await openWithBallUniverse(page);
  // a1 concludes play; a2 takes play as a premise — a chain nobody draws.
  await addArgument(
    page,
    "Is it time to play?",
    [F_RED, "all of the ball is red"],
    "all of the time to play is now",
  );
  await addArgument(
    page,
    "Addition",
    ["all of the time to play is now"],
    "all of the time to play is now",
  );

  await expect(page.locator(".vue-flow__edge.e-chains").first()).toBeVisible();
  // Truth state is encoded in the node itself: play forced true, blue forced
  // false, red asserted (ids follow creation order: s1 red, s2 blue, s3 play).
  await expect(page.locator('.vue-flow__node.entailed-true[data-id="s3"]')).toBeVisible();
  await expect(page.locator('.vue-flow__node.entailed-false[data-id="s2"]')).toBeVisible();
  await expect(page.locator('.vue-flow__node.asserted[data-id="s1"]')).toBeVisible();
  // Argument nodes carry their computed verdict.
  await expect(page.locator('.vue-flow__node.n-arg.valid[data-id="a1"]')).toBeVisible();
});

test("edge filters and canvas↔inspector selection sync", async ({ page }) => {
  await openWithBallUniverse(page);

  await expect(page.locator(".vue-flow__edge.e-shares").first()).toBeVisible();
  await page.getByRole("checkbox", { name: "shares" }).uncheck();
  await expect(page.locator(".vue-flow__edge.e-shares")).toHaveCount(0);
  await page.getByRole("checkbox", { name: "shares" }).check();
  await expect(page.locator(".vue-flow__edge.e-shares").first()).toBeVisible();

  // Clicking a node selects it in the inspector.
  await page.locator('.vue-flow__node[data-id="s2"]').click();
  const inspector = page.locator(".card", { hasText: "Inspector" });
  await expect(inspector.getByText("all of the ball is blue", { exact: true })).toBeVisible();
  await expect(inspector.getByText("force this false")).toBeVisible();
});

test("dragged layout persists in the document and survives reload", async ({ page }) => {
  await openWithBallUniverse(page);

  const node = page.locator('.vue-flow__node[data-id="s1"]');
  const box = (await node.boundingBox())!;
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await page.mouse.down();
  await page.mouse.move(box.x + 180, box.y + 120, { steps: 8 });
  await page.mouse.up();

  const stored = () =>
    page.evaluate(() => {
      const u = JSON.parse(localStorage.getItem("calculemus.v2.universe")!);
      return u.layout?.s1 ?? null;
    });
  const pos = await stored();
  expect(pos).not.toBeNull();

  await page.reload();
  await expect(page.locator('.vue-flow__node[data-id="s1"]')).toBeVisible({ timeout: 20_000 });
  expect(await stored()).toEqual(pos);
});
