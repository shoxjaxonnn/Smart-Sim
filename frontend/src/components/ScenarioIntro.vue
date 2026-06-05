<script setup>
defineProps({
  scenarios: { type: Array, default: () => [] },
  offline: { type: Boolean, default: false },
})
const emit = defineEmits(['start'])

const features = [
  { icon: '🛡️', title: 'Anti-Hallucination', text: 'AI raqamlarni eslab qolmaydi — get_fact orqali faqat tasdiqlangan faktni oladi.' },
  { icon: '📊', title: 'Rubrika baholash', text: 'Javobing mezon bo\'yicha avtomatik baholanadi, har biri izohli.' },
  { icon: '🧩', title: 'Har qanday fan', text: 'Yangi fan = yangi senariy fayli. Kod yozilmaydi.' },
]
</script>

<template>
  <section class="intro">
    <div class="hero">
      <h1>An'anaviy test emas — <span class="grad">real simulyatsiya</span>.</h1>
      <p class="lead">
        Statik savollarga javob bermaysan. AI yuritadigan muhitda real muammoni hal qilasan,
        AI bilan muloqot qilasan, va rubrika bo'yicha baho olasan.
      </p>
    </div>

    <div class="features">
      <div v-for="f in features" :key="f.title" class="feature">
        <span class="feat-icon">{{ f.icon }}</span>
        <div class="feat-title">{{ f.title }}</div>
        <div class="feat-text">{{ f.text }}</div>
      </div>
    </div>

    <h2 class="section-title">Senariy tanla</h2>

    <div v-if="offline" class="empty">
      Backend ulanmagan. Senariylar yuklanmadi.
    </div>
    <div v-else-if="scenarios.length === 0" class="empty">
      Hali approved senariy yo'q. Teacher panelda birini yarating va approve qiling.
    </div>

    <div class="cards">
      <button
        v-for="s in scenarios"
        :key="s.id"
        class="card"
        @click="emit('start', s)"
      >
        <span class="card-subject">{{ s.subject }}</span>
        <span class="card-title">{{ s.title }}</span>
        <span class="card-go">Boshlash →</span>
      </button>
    </div>
  </section>
</template>

<style scoped>
.intro { overflow-y: auto; padding: 8px 4px 24px; }

.hero { max-width: 720px; margin: 14px 0 30px; }
h1 { font-size: 40px; line-height: 1.1; margin: 0 0 14px; letter-spacing: -.5px; }
.grad {
  background: linear-gradient(135deg, var(--accent), var(--accent-2));
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}
.lead { font-size: 17px; color: var(--text-dim); line-height: 1.6; margin: 0; }

.features {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 38px;
}
.feature {
  background: linear-gradient(160deg, var(--panel-2), var(--panel));
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 20px;
  box-shadow: var(--shadow);
}
.feat-icon { font-size: 26px; }
.feat-title { font-weight: 650; margin: 10px 0 6px; font-size: 16px; }
.feat-text { color: var(--text-dim); font-size: 14px; line-height: 1.55; }

.section-title { font-size: 20px; margin: 0 0 16px; }
.empty { color: var(--text-dim); padding: 8px 0; }

.cards {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}
.card {
  text-align: left;
  background: linear-gradient(160deg, var(--panel-2), var(--panel));
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 22px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  color: var(--text);
  box-shadow: var(--shadow);
}
.card:hover { border-color: var(--accent); filter: brightness(1.04); }
.card-subject {
  font-size: 12px;
  color: var(--accent);
  text-transform: uppercase;
  letter-spacing: .6px;
  font-weight: 600;
}
.card-title { font-size: 18px; font-weight: 650; }
.card-go { color: var(--text-dim); font-size: 14px; margin-top: 4px; }

@media (max-width: 760px) {
  h1 { font-size: 30px; }
  .features { grid-template-columns: 1fr; }
  .cards { grid-template-columns: 1fr; }
}
</style>
