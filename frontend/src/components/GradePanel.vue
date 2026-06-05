<script setup>
import { computed } from 'vue'

const props = defineProps({
  scenario: { type: Object, required: true },
  grade: { type: Object, required: true },
})
defineEmits(['restart'])

const pct = computed(() => {
  if (!props.grade.max_score) return 0
  return Math.round((props.grade.total_score / props.grade.max_score) * 100)
})

const verdict = computed(() => {
  const p = pct.value
  if (p >= 80) return { label: 'A\'lo', color: 'var(--good)' }
  if (p >= 50) return { label: 'Yaxshi', color: 'var(--warn)' }
  return { label: 'Ishlash kerak', color: 'var(--bad)' }
})

function barColor(score, max) {
  const p = max ? score / max : 0
  if (p >= 0.8) return 'var(--good)'
  if (p >= 0.4) return 'var(--warn)'
  return 'var(--bad)'
}
</script>

<template>
  <section class="grade-wrap">
    <div class="summary">
      <div class="ring" :style="{ '--p': pct, '--c': verdict.color }">
        <div class="ring-inner">
          <div class="ring-score">{{ grade.total_score }}<span>/{{ grade.max_score }}</span></div>
          <div class="ring-pct">{{ pct }}%</div>
        </div>
      </div>
      <div class="summary-text">
        <span class="verdict" :style="{ color: verdict.color }">{{ verdict.label }}</span>
        <h2>{{ scenario.title }}</h2>
        <p>Rubrika bo'yicha avtomatik baholandi. Har bir mezon izoh bilan.</p>
        <button class="btn-ghost" @click="$emit('restart')">← Yangi simulyatsiya</button>
      </div>
    </div>

    <div class="criteria">
      <div v-for="(c, i) in grade.criteria" :key="i" class="crit">
        <div class="crit-head">
          <span class="crit-name">{{ c.name }}</span>
          <span class="crit-score">{{ c.score }} / {{ c.max }}</span>
        </div>
        <div class="track">
          <div
            class="fill"
            :style="{ width: (c.max ? (c.score / c.max * 100) : 0) + '%', background: barColor(c.score, c.max) }"
          ></div>
        </div>
        <p class="crit-just">{{ c.justification }}</p>
      </div>
    </div>
  </section>
</template>

<style scoped>
.grade-wrap { overflow-y: auto; padding: 12px 4px 32px; }

.summary {
  display: flex;
  gap: 32px;
  align-items: center;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 32px;
  box-shadow: var(--shadow);
  margin-bottom: 24px;
}
.ring {
  flex: none;
  width: 160px; height: 160px;
  border-radius: 50%;
  background: conic-gradient(var(--c) calc(var(--p) * 1%), var(--border) 0);
  display: grid; place-items: center;
  transition: all 0.3s ease;
  box-shadow: 0 4px 10px rgba(15, 23, 42, 0.04);
}
.ring-inner {
  width: 124px; height: 124px;
  border-radius: 50%;
  background: var(--panel);
  display: grid; place-items: center;
  text-align: center;
}
.ring-score { font-size: 32px; font-weight: 800; color: var(--ink); }
.ring-score span { font-size: 16px; color: var(--text-dim); font-weight: 500; }
.ring-pct { font-size: 13px; color: var(--text-dim); font-weight: 600; margin-top: 2px; }

.summary-text h2 { margin: 6px 0 8px; font-size: 26px; color: var(--ink); font-weight: 800; }
.summary-text p { color: var(--text-dim); margin: 0 0 20px; line-height: 1.6; font-size: 14.5px; }
.verdict { font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: 1px; }

.criteria { display: flex; flex-direction: column; gap: 16px; }
.crit {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 20px 24px;
  box-shadow: var(--shadow);
  transition: transform 0.2s ease;
}
.crit:hover {
  transform: translateY(-1px);
}
.crit-head { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 14px; }
.crit-name { font-weight: 700; font-size: 16px; color: var(--ink); }
.crit-score { font-variant-numeric: tabular-nums; color: var(--ink); font-weight: 700; font-size: 15px; }
.track {
  height: 8px;
  background: var(--bg-soft);
  border-radius: 999px;
  overflow: hidden;
  border: 1px solid var(--border);
}
.fill { height: 100%; border-radius: 999px; transition: width .8s cubic-bezier(.2,.8,.2,1); }
.crit-just { margin: 14px 0 0; color: var(--text-dim); font-size: 14.5px; line-height: 1.6; }

@media (max-width: 760px) {
  .summary { flex-direction: column; text-align: center; gap: 24px; padding: 24px; }
}
</style>
