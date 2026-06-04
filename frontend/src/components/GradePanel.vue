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
.grade-wrap { overflow-y: auto; padding: 8px 4px 24px; }

.summary {
  display: flex;
  gap: 32px;
  align-items: center;
  background: linear-gradient(160deg, var(--panel-2), var(--panel));
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 28px;
  box-shadow: var(--shadow);
  margin-bottom: 22px;
}
.ring {
  flex: none;
  width: 150px; height: 150px;
  border-radius: 50%;
  background: conic-gradient(var(--c) calc(var(--p) * 1%), var(--border) 0);
  display: grid; place-items: center;
}
.ring-inner {
  width: 118px; height: 118px;
  border-radius: 50%;
  background: var(--panel);
  display: grid; place-items: center;
  text-align: center;
}
.ring-score { font-size: 30px; font-weight: 750; }
.ring-score span { font-size: 16px; color: var(--text-dim); font-weight: 500; }
.ring-pct { font-size: 13px; color: var(--text-dim); }

.summary-text h2 { margin: 6px 0 8px; font-size: 24px; }
.summary-text p { color: var(--text-dim); margin: 0 0 16px; line-height: 1.5; }
.verdict { font-size: 13px; font-weight: 700; text-transform: uppercase; letter-spacing: .6px; }

.criteria { display: flex; flex-direction: column; gap: 14px; }
.crit {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 18px 20px;
}
.crit-head { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 12px; }
.crit-name { font-weight: 600; font-size: 15.5px; }
.crit-score { font-variant-numeric: tabular-nums; color: var(--text-dim); font-weight: 600; }
.track {
  height: 9px;
  background: var(--bg-soft);
  border-radius: 999px;
  overflow: hidden;
  border: 1px solid var(--border);
}
.fill { height: 100%; border-radius: 999px; transition: width .6s cubic-bezier(.2,.8,.2,1); }
.crit-just { margin: 12px 0 0; color: var(--text-dim); font-size: 14px; line-height: 1.55; }

@media (max-width: 760px) {
  .summary { flex-direction: column; text-align: center; gap: 20px; }
}
</style>
