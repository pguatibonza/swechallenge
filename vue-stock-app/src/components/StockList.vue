
<template>
  <div class="app-container">
    <!-- Sidebar with filters -->
    <aside class="sidebar">
      <!-- Action filter -->
      <div class="filter-group">
        <label for="action">Action</label>
        <select id="action" v-model="action" @change="fetchStocks">
          <option value="">All</option>
          <option v-for="opt in actionOptions" :key="opt" :value="opt">{{ opt }}</option>
        </select>
      </div>

      <!-- Rating filter -->
      <div class="filter-group">
        <label for="ratingFrom">Rating</label>
        <div class="rating-row">
          <select id="ratingFrom" v-model="ratingFrom" @change="fetchStocks">
            <option value="">From</option>
            <option v-for="opt in ratingOptions" :key="opt" :value="opt">{{ opt }}</option>
          </select>
          <span class="arrow">→</span>
          <select id="ratingTo" v-model="ratingTo" @change="fetchStocks">
            <option value="">To</option>
            <option v-for="opt in ratingOptions" :key="opt" :value="opt">{{ opt }}</option>
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
    <main class="main-content">
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
      <table class="stock-table">
        <thead>
          <tr>
            <th>Ticker</th>
            <th>Company</th>
            <th> Brokerage </th>
            <th>Action</th>
            <th>Rating <br/> from → to</th>
            <th>Target <br/> from → to </th>
            <th> time </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in stocks" :key="item.ticker + item.time" @click="viewDetail(item.ticker)">
            <td>{{ item.ticker }}</td>
            <td>{{ item.company }}</td>
            <td>{{ item.brokerage}} </td>
            <td>{{ item.action }}</td>
            <td>{{ item.rating_from }} → {{ item.rating_to }}</td>
            <td>{{ item.target_from }} → {{ item.target_to }}</td>
            <td> {{ item.time }}</td>
          </tr>
        </tbody>
      </table>
    </main>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { ref, onMounted } from 'vue'

const router = useRouter()

function viewDetail(ticker: string) {
  router.push({ name: 'StockDetail', params: { ticker } })
}

const stocks = ref<StockItem[]>([])
const actionOptions = ref<string[]>([])
const ratingOptions = ref<string[]>([])
const sortableCols = ['ticker', 'company', 'action', 'brokerage' , 'rating_from', 'rating_to', 'target_from', 'target_to' ,'time']

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
    if (dateFrom.value) {
    // start of day in ISO
    const start = new Date(dateFrom.value + 'T00:00:00')
    params.append('date_from', start.toISOString())
  }
  if (dateTo.value) {
    // end of day (23:59:59.999) in ISO
    const end = new Date(dateTo.value + 'T00:00:00')
    end.setHours(23, 59, 59, 999)
    params.append('date_to', end.toISOString())
  }
  params.append('sort', sortBy.value)
  params.append('order', order.value)

  const res = await fetch(`http://localhost:8081/stocks?${params}`)
  const body = await res.json()
  stocks.value = body.items
    
    // derive options from first fetch
    if (!actionOptions.value.length && stocks.value.length) {
        actionOptions.value = Array.from(new Set(stocks.value.map(i => i.action)))
        ratingOptions.value = Array.from(new Set([...stocks.value.map(i => i.rating_from), ...stocks.value.map(i => i.rating_to)]))
    }
}

onMounted(async () => {

  await fetchStocks()
})
</script>
