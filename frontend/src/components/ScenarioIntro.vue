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
.intro { overflow-y: auto; padding: 12px 4px 32px; }

.hero { max-width: 800px; margin: 24px 0 36px; }
h1 { font-size: 42px; line-height: 1.15; margin: 0 0 16px; letter-spacing: -0.03em; font-weight: 800; }
.grad {
  background: linear-gradient(135deg, var(--accent), var(--accent-2));
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}
.lead { font-size: 18px; color: var(--text-dim); line-height: 1.6; margin: 0; font-weight: 400; }

.features {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
  margin-bottom: 44px;
}
.feature {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 24px;
  box-shadow: var(--shadow);
  transition: transform 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}
.feature:hover {
  transform: translateY(-2px);
}
.feat-icon { font-size: 28px; }
.feat-title { font-weight: 700; margin: 12px 0 8px; font-size: 17px; color: var(--ink); }
.feat-text { color: var(--text-dim); font-size: 14.5px; line-height: 1.6; }

.section-title { font-size: 22px; margin: 0 0 20px; font-weight: 700; color: var(--ink); }
.empty { color: var(--text-dim); padding: 12px 0; font-size: 14.5px; }

.cards {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
}
.card {
  text-align: left;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  color: var(--text);
  box-shadow: var(--shadow);
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}
.card:hover {
  border-color: var(--accent);
  transform: translateY(-3px);
  box-shadow: 0 12px 20px -8px rgba(15, 23, 42, 0.08);
}
.card-subject {
  font-size: 12px;
  color: var(--accent-2);
  text-transform: uppercase;
  letter-spacing: .8px;
  font-weight: 700;
}
.card-title { font-size: 20px; font-weight: 700; color: var(--ink); }
.card-go { color: var(--text-dim); font-size: 14.5px; margin-top: 6px; font-weight: 500; display: inline-flex; align-items: center; gap: 4px; transition: color 0.2s ease; }
.card:hover .card-go { color: var(--ink); }

@media (max-width: 760px) {
  h1 { font-size: 32px; }
  .features { grid-template-columns: 1fr; gap: 16px; }
  .cards { grid-template-columns: 1fr; gap: 16px; }
}
</style>
