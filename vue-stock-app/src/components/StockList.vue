<template>
  <div>
    <div class="mb-4 flex space-x-2">
      <input
        v-model="search"
        @keyup.enter="loadStocks"
        type="text"
        placeholder="Search (ticker, company, brokerage...)"
        class="border p-2 flex-grow"
      />
      <button @click="loadStocks" class="bg-blue-600 text-white px-4 py-2">Search</button>
    </div>

    <table class="min-w-full bg-white border">
      <thead>
        <tr class="bg-gray-200">
          <th class="px-4 py-2 text-left">Ticker</th>
          <th class="px-4 py-2 text-left">Company</th>
          <th class="px-4 py-2 text-left">Brokerage</th>
          <th class="px-4 py-2 text-left">Action</th>
          <th class="px-4 py-2 text-left">Rating <br/>From → To</th>
          <th class="px-4 py-2 text-left">Target <br/> From → To</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in stocks" :key="item.ticker + item.time" class="hover:bg-gray-100">
          <td class="px-4 py-2">{{ item.ticker }}</td>
          <td class="px-4 py-2">{{ item.company }}</td>
          <td class="px-4 py-2">{{ item.brokerage }}</td>
          <td class="px-4 py-2">{{ item.action }}</td>
          <td class="px-4 py-2">{{ item.rating_from }} → {{ item.rating_to }}</td>
          <td class="px-4 py-2">{{ item.target_from }} → {{ item.target_to }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue';

type StockItem = {
  ticker: string;
  company: string;
  brokerage: string;
  action: string;
  rating_from: string;
  rating_to: string;
  target_from: string;
  target_to: string;
  time: string;
};

const search = ref('');
const stocks = ref<StockItem[]>([]);

async function loadStocks() {
  const params = new URLSearchParams();
  if (search.value) params.append('search', search.value);

  const res = await fetch(`http://localhost:8081/stocks?${params.toString()}`);
  const json = await res.json();
  stocks.value = json.items;
}

// Initial load
loadStocks();
</script>

<style scoped>
table { border-collapse: collapse; width: 100%; }
th, td { border: 1px solid #e2e8f0; }
</style>