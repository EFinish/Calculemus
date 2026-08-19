<script setup lang="ts">
// The M3 canvas: the universe as a visible web. Nodes are statements,
// formulas, and arguments; every edge is derived by the engine (shares /
// contradicts / chains) — the user's only spatial contribution is layout,
// which persists in the universe document.
import { computed, reactive } from "vue";
import { VueFlow, MarkerType, Position } from "@vue-flow/core";
import type { NodeDragEvent, NodeMouseEvent, Node, Edge as FlowEdge } from "@vue-flow/core";
import { Background } from "@vue-flow/background";
import { universe, verdicts, selected, select, isAsserted, setLayout, clearLayout } from "../store";
import { renderRef } from "../render";

const filters = reactive({ shares: true, contradicts: true, chains: true });

function truthClass(id: string): string {
  if (verdicts.value?.unsatCore?.includes(id)) return "conflict";
  if (isAsserted(id)) return "asserted";
  if (verdicts.value?.entailedTrue?.includes(id)) return "entailed-true";
  if (verdicts.value?.entailedFalse?.includes(id)) return "entailed-false";
  return "undetermined";
}

// Hand-placed position wins; otherwise a simple three-column auto layout
// (statements | formulas | arguments).
function positionOf(id: string, column: number, index: number) {
  const saved = universe.layout?.[id];
  return saved ? { x: saved.x, y: saved.y } : { x: 40 + column * 300, y: 40 + index * 92 };
}

const nodes = computed<Node[]>(() => {
  const out: Node[] = [];
  universe.statements.forEach((s, i) =>
    out.push({
      id: s.id,
      label: s.text,
      position: positionOf(s.id, 0, i),
      class: `cnode n-stmt ${truthClass(s.id)}${selected.value === s.id ? " picked" : ""}`,
      sourcePosition: Position.Right,
      targetPosition: Position.Left,
      connectable: false,
    }),
  );
  (universe.formulas ?? []).forEach((f, i) =>
    out.push({
      id: f.id,
      label: renderRef(universe, f.id),
      position: positionOf(f.id, 1, i),
      class: `cnode n-form ${truthClass(f.id)}${selected.value === f.id ? " picked" : ""}`,
      sourcePosition: Position.Right,
      targetPosition: Position.Left,
      connectable: false,
    }),
  );
  (universe.arguments ?? []).forEach((a, i) => {
    const av = verdicts.value?.arguments?.find((v) => v.id === a.id);
    out.push({
      id: a.id,
      label: `${a.title}${av ? (av.valid ? " ✓" : " ✗") : ""}`,
      position: positionOf(a.id, 2, i),
      class: `cnode n-arg ${av ? (av.valid ? "valid" : "invalid") : ""}${selected.value === a.id ? " picked" : ""}`,
      sourcePosition: Position.Right,
      targetPosition: Position.Left,
      connectable: false,
    });
  });
  return out;
});

const edges = computed<FlowEdge[]>(() =>
  (verdicts.value?.edges ?? [])
    .filter((e) => filters[e.type])
    .map((e, i) => {
      const id = `${e.type}:${e.from}->${e.to}#${i}`;
      switch (e.type) {
        case "chains":
          return {
            id,
            source: e.from,
            target: e.to,
            animated: true,
            markerEnd: MarkerType.ArrowClosed,
            class: "e-chains",
          };
        case "contradicts":
          return { id, source: e.from, target: e.to, class: "e-contradicts" };
        default:
          return { id, source: e.from, target: e.to, class: "e-shares" };
      }
    }),
);

function onNodeClick({ node }: NodeMouseEvent) {
  select(node.id);
}

function onNodeDragStop({ node }: NodeDragEvent) {
  setLayout(node.id, node.position.x, node.position.y);
}
</script>

<template>
  <section class="card canvas-card">
    <div class="toolbar">
      <h2>Canvas</h2>
      <label class="small"><input v-model="filters.shares" type="checkbox" /> <span class="sw shares"></span>shares</label>
      <label class="small"><input v-model="filters.contradicts" type="checkbox" /> <span class="sw contradicts"></span>contradicts</label>
      <label class="small"><input v-model="filters.chains" type="checkbox" /> <span class="sw chains"></span>chains</label>
      <span class="spacer"></span>
      <button class="small" title="Forget hand-placed positions" @click="clearLayout">Auto-arrange</button>
    </div>
    <div class="flow-wrap">
      <VueFlow
        :nodes="nodes"
        :edges="edges"
        fit-view-on-init
        :min-zoom="0.2"
        :max-zoom="1.25"
        @node-click="onNodeClick"
        @node-drag-stop="onNodeDragStop"
      >
        <Background />
      </VueFlow>
    </div>
    <p v-if="nodes.length === 0" class="muted small">
      The web draws itself as you compose — nothing here to arrange yet.
    </p>
  </section>
</template>

<style scoped>
.canvas-card {
  display: flex;
  flex-direction: column;
  min-height: 70vh;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 0.9rem;
  margin-bottom: 0.6rem;
}
.toolbar h2 {
  margin: 0;
}
.toolbar label {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  color: var(--muted);
}
.spacer {
  flex: 1;
}
.sw {
  display: inline-block;
  width: 14px;
  height: 3px;
  border-radius: 2px;
}
.sw.shares {
  background: var(--muted);
}
.sw.contradicts {
  background: var(--false);
}
.sw.chains {
  background: var(--accent);
}
.flow-wrap {
  /* Explicit height, not min-height: VueFlow sizes itself with height:100%,
     which resolves to 0 against a min-height-only parent (Firefox strictly),
     leaving the hit-test layers with no box. */
  height: clamp(420px, 62vh, 900px);
  border: 1px solid var(--rule);
  border-radius: 6px;
  overflow: hidden;
  background: var(--ground);
}
</style>

<!-- Unscoped: vue-flow renders nodes/edges outside this component's scope. -->
<style>
@import "@vue-flow/core/dist/style.css";
@import "@vue-flow/core/dist/theme-default.css";

.vue-flow__node.cnode {
  font-size: 0.78rem;
  line-height: 1.3;
  max-width: 220px;
  padding: 0.45rem 0.6rem;
  border-radius: 6px;
  border: 1.5px solid var(--rule);
  background: var(--surface);
  color: var(--ink);
  cursor: pointer;
}
.vue-flow__node.cnode.picked {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}
.vue-flow__node.cnode.asserted {
  border-color: var(--accent);
  box-shadow: inset 3px 0 0 var(--accent);
}
.vue-flow__node.cnode.entailed-true {
  border-color: var(--true);
  background: var(--true-soft);
}
.vue-flow__node.cnode.entailed-false {
  border-color: var(--false);
  background: var(--false-soft);
}
.vue-flow__node.cnode.conflict {
  border-color: var(--false);
  box-shadow: 0 0 0 3px var(--false-soft);
}
.vue-flow__node.cnode.n-arg {
  border-style: double;
  border-width: 3px;
  font-weight: 600;
}
.vue-flow__node.cnode.n-arg.valid {
  border-color: var(--true);
}
.vue-flow__node.cnode.n-arg.invalid {
  border-color: var(--false);
}
.vue-flow__edge.e-shares .vue-flow__edge-path {
  stroke: var(--muted);
  stroke-dasharray: 4 4;
  opacity: 0.55;
}
.vue-flow__edge.e-contradicts .vue-flow__edge-path {
  stroke: var(--false);
  stroke-width: 2;
}
.vue-flow__edge.e-chains .vue-flow__edge-path {
  stroke: var(--accent);
  stroke-width: 2.5;
}
</style>
