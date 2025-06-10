
<template>
  <div class="app-container">
    <!-- Sidebar with filters -->
    <aside class="sidebar">
      <!-- Action filter -->
      <div class="filter-group">
        <label for="action">Action</label>
        <select id="action" v-model="action" @change="fetchStocks">
          <option value="">All</option>
          <option v-for="opt in actions" :key="opt" :value="opt">{{ opt }}</option>
        </select>
      </div>

      <!-- Rating filter -->
      <div class="filter-group">
        <label for="ratingFrom">Rating</label>
        <div class="rating-row">
          <select id="ratingFrom" v-model="ratingFrom" @change="fetchStocks">
            <option value="">From</option>
            <option v-for="opt in ratings" :key="opt" :value="opt">{{ opt }}</option>
          </select>
          <span class="arrow">→</span>
          <select id="ratingTo" v-model="ratingTo" @change="fetchStocks">
            <option value="">To</option>
            <option v-for="opt in ratings" :key="opt" :value="opt">{{ opt }}</option>
          </select>
        </div>
      </div>

      <!-- Target range -->
      <div class="filter-group">
        <label>Target Range</label>
        <div class="range-row">
          <input type="number" v-model.number="minTF" @input="fetchStocks" placeholder="Min" />
          <span class="arrow">→</span>
          <input type="number" v-model.number="maxTF" @input="fetchStocks" placeholder="Max" />
        </div>
      </div>

      <!-- Date range -->
      <div class="filter-group">
        <label>Date Range</label>
        <div class="range-row">
          <input type="date" v-model="dateFrom" @change="fetchStocks" />
          <span class="arrow">→</span>
          <input type="date" v-model="dateTo" @change="fetchStocks" />
        </div>
      </div>

      <!-- Sort controls -->
      <div class="filter-group">
        <label>Sort By / Order</label>
        <div class="sort-row">
          <select v-model="sortBy" @change="fetchStocks">
            <option v-for="col in sortableCols" :key="col" :value="col">{{ col }}</option>
          </select>
          <select v-model="order" @change="fetchStocks">
            <option value="ASC">Ascending</option>
            <option value="DESC">Descending</option>
          </select>
        </div>
      </div>
    </aside>

    <!-- Main content: Search + Table -->
    <main class="main-content bg-gray-900 p-4 rounded-lg">
      <!-- Search box above the table -->
      <div class="search-group">
        <input
          v-model="search"
          @input="fetchStocks"
          type="text"
          placeholder="Search ticker, company, action…"
        />
      </div>

      <!-- Data table -->
      <table class="stock-table w-full border-collapse bg-gray-800 text-gray-100">
        <thead>
          <tr class="bg-gray-700 text-gray-100">
            <th class="border border-gray-700 p-2">Ticker</th>
            <th class="border border-gray-700 p-2">Company</th>
            <th class="border border-gray-700 p-2" >Action</th>
            <th class="border border-gray-700 p-2" >Rating <br/> from → to</th>
            <th class="border border-gray-700 p-2" >Target <br/> from → to </th>
          </tr>
        </thead>
        <tbody>
          <tr class="odd:bg-gray-800 even:bg-gray-900 hover:bg-gray-700" v-for="item in stocks" :key="item.ticker + item.time">
            <td class="border border-gray-700 p-2" >{{ item.ticker }}</td>
            <td class="border border-gray-700 p-2"  >{{ item.company }}</td>
            <td class="border border-gray-700 p-2"  >{{ item.action }}</td>
            <td class="border border-gray-700 p-2"  >{{ item.rating_from }} → {{ item.rating_to }}</td>
            <td class="border border-gray-700 p-2" >{{ item.target_from }} → {{ item.target_to }}</td>
          </tr>
        </tbody>
      </table>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const stocks = ref<StockItem[]>([])
const actions = ref<string[]>([])       // populated from backend or hardcoded
const ratings = ref<string[]>([])       // same
const sortableCols = ['ticker', 'company', 'action', 'raw_time']

// filter refs
const search = ref('')
const action = ref('')
const ratingFrom = ref('')
const ratingTo = ref('')
const minTF = ref<number|undefined>()
const maxTF = ref<number|undefined>()
const dateFrom = ref<string>('')
const dateTo = ref<string>('')
const sortBy = ref('ticker')
const order = ref<'ASC'|'DESC'>('ASC')

async function fetchStocks() {
  const params = new URLSearchParams()
  if (search.value) params.append('search', search.value)
  if (action.value) params.append('action', action.value)
  if (ratingFrom.value) params.append('rating_from', ratingFrom.value)
  if (ratingTo.value)   params.append('rating_to', ratingTo.value)
  if (minTF.value!=null) params.append('min_target_from', minTF.value.toString())
  if (maxTF.value!=null) params.append('max_target_from', maxTF.value.toString())
  if (dateFrom.value) params.append('date_from', dateFrom.value)
  if (dateTo.value)   params.append('date_to', dateTo.value)
  params.append('sort', sortBy.value)
  params.append('order', order.value)

  const res = await fetch(`http://localhost:8081/stocks?${params}`)
  const body = await res.json()
  stocks.value = body.items
}

onMounted(async () => {
  // fetch initial dropdown options, if you want, then:
  await fetchStocks()
})
</script>
