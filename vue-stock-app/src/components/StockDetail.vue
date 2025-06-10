<template>
  <div v-if="stock" class="stock-detail-container">
    <h1 class="detail-header">Details for {{ stock.ticker }}</h1>
    <ul class="detail-list">
      <li><span class="label">Company:</span><span class="value">{{ stock.company }}</span></li>
      <li><span class="label">Brokerage:</span><span class="value">{{ stock.brokerage }}</span></li>
      <li><span class="label">Action:</span><span class="value">{{ stock.action }}</span></li>
      <li><span class="label">Rating From:</span><span class="value">{{ stock.rating_from }}</span></li>
      <li><span class="label">Rating To:</span><span class="value">{{ stock.rating_to }}</span></li>
      <li><span class="label">Target From:</span><span class="value">{{ stock.target_from }}</span></li>
      <li><span class="label">Target To:</span><span class="value">{{ stock.target_to }}</span></li>
      <li><span class="label">Date:</span>
          <span class="value">{{ new Date(stock.time).toLocaleString() }}</span>
      </li>
    </ul>
  </div>
  <div v-else class="loading">Loading…</div>
</template>


<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute }     from 'vue-router'

interface StockDetail {
  ticker: string
  brokerage :string
  action: string
  rating_from: number
  rating_to: number
  target_from :number
  target_from: number
  time: string
  // …etc
}

const route = useRoute()
const stock  = ref<StockDetail| null>(null)

// grab ticker from the URL
const ticker = String(route.params.ticker)

async function fetchDetail() {
  const res  = await fetch(`http://localhost:8081/stocks/${ticker}`)
  const body = await res.json()
  // assume body is the detail object
  stock.value = body
}

onMounted(fetchDetail)
</script>
