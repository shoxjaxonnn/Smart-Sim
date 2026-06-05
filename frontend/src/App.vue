<script setup>
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
import TeacherPanel from './components/TeacherPanel.vue'
import StudentPanel from './components/StudentPanel.vue'

const path = ref(window.location.pathname || '/')

const isTeacher = computed(() => path.value.startsWith('/teacher'))

function syncPath() {
  path.value = window.location.pathname || '/'
}

function go(pathname) {
  if (window.location.pathname === pathname) return
  window.history.pushState({}, '', pathname)
  syncPath()
}

onMounted(() => {
  window.addEventListener('popstate', syncPath)
})

onBeforeUnmount(() => {
  window.removeEventListener('popstate', syncPath)
})
</script>

<template>
  <div class="shell">
    <header class="topbar">
      <div class="brand">
        <span class="logo">SS</span>
        <div>
          <div class="brand-name">Smart Sim</div>
          <div class="brand-sub">Uzbek AI simulation lab</div>
        </div>
      </div>
      <nav class="nav">
        <button class="nav-btn" :class="{ active: !isTeacher }" @click="go('/')">Student</button>
        <button class="nav-btn" :class="{ active: isTeacher }" @click="go('/teacher')">Teacher</button>
      </nav>
    </header>

    <main class="stage">
      <TeacherPanel v-if="isTeacher" />
      <StudentPanel v-else />
    </main>
  </div>
</template>

<style scoped>
.shell {
  height: 100%;
  display: flex;
  flex-direction: column;
  max-width: 1720px;
  margin: 0 auto;
  padding: 0 18px 18px;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 2px 12px;
}
.brand { display: flex; align-items: center; gap: 14px; }
.logo {
  width: 42px; height: 42px;
  display: grid; place-items: center;
  font-size: 13px;
  font-weight: 800;
  border-radius: 8px;
  background: var(--ink);
  color: var(--paper);
  border: 1px solid var(--ink);
  box-shadow: var(--shadow);
}
.brand-name { font-size: 18px; font-weight: 800; }
.brand-sub { font-size: 12px; color: var(--text-dim); margin-top: 2px; }

.nav {
  display: flex;
  gap: 10px;
  align-items: center;
}
.nav-btn {
  background: var(--panel);
  color: var(--text-dim);
  border: 1px solid var(--border);
  padding: 10px 15px;
  border-radius: 8px;
  font-weight: 700;
}
.nav-btn.active {
  color: var(--paper);
  background: var(--ink);
  border-color: var(--ink);
  box-shadow: var(--shadow);
}

.stage {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

@media (max-width: 760px) {
  .shell { padding: 0 14px 14px; }
  .topbar { align-items: stretch; flex-direction: column; }
  .nav { display: grid; grid-template-columns: 1fr 1fr; }
}
</style>
