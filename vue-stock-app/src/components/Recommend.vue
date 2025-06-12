<template>
  <div class="recommend-container">
    <button class="back-btn" @click="goBack">← Go back</button>
    <h1 class="recommend-title">Top 10 </h1>

    <div v-if="isLoading" class="loading">Loading recomendations..</div>
    <div v-else-if="error" class="error">Error: {{ error }}</div>
    <table v-else class="stock-table">
      <thead>
        <tr>
          <th>Ticker</th>
          <th>Company</th>
          <th>Brokerage</th>
          <th>Rating</th>
          <th>Target</th>
          <th>Current Price</th>
          <th>Upside(times)</th>
          <th>Score(times)</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="rec in recs" :key="rec.ticker">
          <td>{{ rec.ticker }}</td>
          <td>{{ rec.company }}</td>
          <td>{{ rec.brokerage }}</td>
          <td>{{ rec.rating_from }} → {{ rec.rating_to }}</td>
          <td>{{ rec.target_from }} → {{ rec.target_to }}</td>
          <td>{{ rec.current_price }}</td>
          <td>{{ rec.upside_pct.toFixed(1) }}</td>
          <td>{{ rec.composite.toFixed(2) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'

interface RecResult {
  ticker: string
  company: string
  brokerage: string
  rating_from: string
  rating_to: string
  target_from: number
  target_to: number
  current_price: number
  upside_pct: number
  composite: number
}

const router = useRouter()
function goBack() {
  router.back()
}

const recs = ref<RecResult[]>([])
const isLoading = ref(true)
const error = ref<string | null>(null)

async function fetchRecs() {
  isLoading.value = true
  error.value = null
  try {
    const res = await fetch('http://localhost:8081/recommend')
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    recs.value = await res.json()
  } catch (e: any) {
    error.value = e.message
  } finally {
    isLoading.value = false
  }
}

onMounted(fetchRecs)
</script>

