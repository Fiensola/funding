<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';

interface ExchangeData {
  [exchange: string]: number;
}

interface TimestampData {
  [exchange: string]: string;
}

interface FundingItem {
  symbol: string;
  exchanges: ExchangeData;
  updated_at: TimestampData;
}

interface FundingApiResponse {
  count: number;
  data: Record<string, Omit<FundingItem, 'symbol'>>;
}

interface TableRow {
  symbol: string;
  minRate: number;
  maxRate: number;
  maxDiff: number;
  [exchange: string]: number | string;
}

const rawItems = ref<Record<string, FundingItem>>({});
const loading = ref(true);
const sortKey = ref<string | null>(null);
const sortDesc = ref(false);

const fetchData = async () => {
  try {
    const res = await fetch('/api/v1/funding-rates');
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const json: FundingApiResponse = await res.json();

    const result: Record<string, FundingItem> = {};
    for (const [symbol, item] of Object.entries(json.data)) {
      result[symbol] = { symbol, ...item };
    }
    rawItems.value = result;
  } catch (err) {
    console.error('Failed to fetch funding rates:', err);
    loading.value = false;
  } finally {
    loading.value = false;
  }
};

let intervalId: number | null = null;

onMounted(() => {
  fetchData();
  intervalId = window.setInterval(fetchData, 30000);
});

onUnmounted(() => {
  if (intervalId) {
    clearInterval(intervalId);
    intervalId = null;
  }
});

const allExchanges = computed(() => {
  const exchanges = new Set<string>();
  Object.values(rawItems.value).forEach(item => {
    Object.keys(item.exchanges).forEach(ex => exchanges.add(ex));
  });
  return Array.from(exchanges).sort();
});

const tableRows = computed(() => {
  return Object.values(rawItems.value).map(item => {
    const rates = Object.values(item.exchanges);
    const minRate = rates.length > 0 ? Math.min(...rates) : 0;
    const maxRate = rates.length > 0 ? Math.max(...rates) : 0;
    let maxDiff = 0;
    if (rates.length > 1) {
      maxDiff = maxRate - minRate;
    }

    const row: TableRow = {
      symbol: item.symbol,
      minRate,
      maxRate,
      maxDiff,
      ...item.exchanges,
    };

    return row;
  });
});

const sortedRows = computed(() => {
  if (!sortKey.value) return tableRows.value;
  return [...tableRows.value].sort((a, b) => {
    const aVal = typeof a[sortKey.value!] === 'number' ? a[sortKey.value!] : -Infinity;
    const bVal = typeof b[sortKey.value!] === 'number' ? b[sortKey.value!] : -Infinity;
    if (typeof aVal === 'number' && typeof bVal === 'number') {
      return sortDesc.value ? bVal - aVal : aVal - bVal;
    }
    return String(aVal).localeCompare(String(bVal));
  });
});

const toggleSort = (key: string) => {
  if (sortKey.value === key) {
    sortDesc.value = !sortDesc.value;
  } else {
    sortKey.value = key;
    sortDesc.value = false;
  }
};

const formatRate = (value: string | number | undefined): string => {
  if (value === undefined || value === null) return '—';
  if (typeof value == "string") {
    return ""
  }
  return value.toFixed(6);
};

const getRateClass = (value: string | number | undefined): string => {
  if (value === undefined || value === null) return '';
  if (typeof value == "string") {
    return ""
  }
  return value > 0 ? 'positive' : value < 0 ? 'negative' : '';
};

const getHighlightClass = (value: string | number | undefined, minRate: number, maxRate: number): string => {
  if (value === undefined || value === null) return '';
  if (value === minRate && minRate !== maxRate) return 'highlight-min';
  if (value === maxRate && minRate !== maxRate) return 'highlight-max';
  return '';
};
</script>

<template>
  <div class="funding-container">
    <h2>Funding Rates</h2>
    <p v-if="loading" class="loading">Loading...</p>
    <table v-else class="funding-table">
      <thead>
        <tr>
          <th @click="toggleSort('symbol')" :class="{ sorted: sortKey === 'symbol' }">
            Symbol {{ sortKey === 'symbol' ? (sortDesc ? '↓' : '↑') : '' }}
          </th>
          <th
            v-for="ex in allExchanges"
            :key="ex"
            @click="toggleSort(ex)"
            :class="{ sorted: sortKey === ex }"
          >
            {{ ex }} {{ sortKey === ex ? (sortDesc ? '↓' : '↑') : '' }}
          </th>
          <th @click="toggleSort('maxDiff')" :class="{ sorted: sortKey === 'maxDiff' }">
            Max Diff {{ sortKey === 'maxDiff' ? (sortDesc ? '↓' : '↑') : '' }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in sortedRows" :key="row.symbol">
          <td class="symbol-cell">{{ row.symbol }}</td>
          <td
            v-for="ex in allExchanges"
            :key="ex"
            :class="[
                getRateClass(row[ex]),
                getHighlightClass(row[ex], row.minRate, row.maxRate)
            ]"
          >
            {{ formatRate(row[ex]) }}
          </td>
          <td class="max-diff-cell">
            {{ row.maxDiff.toFixed(6) }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.funding-container {
  padding: 1.5rem;
  color: #e0e0e0;
  background-color: #121212;
  border-radius: 12px;
  font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
  max-width: 1400px;
  margin: 1.5rem auto;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.5);
}

.funding-container h2 {
  margin-top: 0;
  color: #ffffff;
  font-size: 1.8rem;
}

.loading {
  color: #bbb;
  font-style: italic;
}

.funding-table {
  width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  margin-top: 1rem;
  background-color: #1e1e1e;
  border-radius: 10px;
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
}

.funding-table th,
.funding-table td {
  padding: 0.8rem 0.6rem;
  text-align: right;
  border-bottom: 1px solid #2a2a2a;
}

.funding-table th {
  background-color: #252525;
  color: #ccc;
  font-weight: 600;
  cursor: pointer;
  user-select: none;
  position: sticky;
  top: 0;
}

.funding-table th.sorted {
  background-color: #333;
  color: #fff;
}

.funding-table td.symbol-cell,
.funding-table th:first-child {
  text-align: left;
  font-weight: 600;
  color: #4fc3f7;
}

.funding-table td.max-diff-cell {
  font-weight: 600;
  color: #ffcc00;
}

.funding-table td.positive {
  color: #4caf50;
}

.funding-table td.negative {
  color: #f44336;
}

.funding-table td.highlight-min {
  background-color: rgba(46, 125, 50, 0.1);
  border-radius: 4px;
}

.funding-table td.highlight-max {
  background-color: rgba(198, 40, 40, 0.1);
  border-radius: 4px;
}

@media (max-width: 768px) {
  .funding-container {
    padding: 1rem;
    margin: 1rem;
  }
  .funding-table {
    font-size: 0.85rem;
  }
  .funding-table th,
  .funding-table td {
    padding: 0.6rem 0.4rem;
  }
}
</style>