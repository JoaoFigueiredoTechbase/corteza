<template>
  <wrap v-bind="$props" v-on="$listeners">
    <div class="cdr-container">
      <!-- Header Section -->
      <div class="header-section">
        <h2 class="title">
          <i class="icon-phone"></i>
          Call Detail Records
        </h2>
        <div class="stats-row">
          <div class="stat-card">
            <span class="stat-value">{{ totalRecords }}</span>
            <span class="stat-label">Total Calls</span>
          </div>
          <div class="stat-card">
            <span class="stat-value">{{ totalDuration }}</span>
            <span class="stat-label">Total Duration</span>
          </div>
          <div class="stat-card">
            <span class="stat-value">{{ successRate }}%</span>
            <span class="stat-label">Success Rate</span>
          </div>
        </div>
      </div>

      <!-- Controls Section -->
      <div class="controls-section">
        <div class="search-filter-row">
          <div class="search-box">
            <input
              v-model="searchQuery"
              type="text"
              placeholder="Search calls..."
              class="search-input"
            >
            <i class="search-icon">🔍</i>
          </div>

          <div class="filter-group">
            <select v-model="statusFilter" class="filter-select">
              <option value="">All Status</option>
              <option value="ANSWERED">Answered</option>
              <option value="NO ANSWER">No Answer</option>
              <option value="BUSY">Busy</option>
              <option value="FAILED">Failed</option>
            </select>

            <select v-model="directionFilter" class="filter-select">
              <option value="">All Directions</option>
              <option value="Inbound">Inbound</option>
              <option value="Outbound">Outbound</option>
              <option value="Internal">Internal</option>
            </select>
          </div>

          <div class="action-buttons">
            <button
              class="btn btn-primary"
              :disabled="loading"
              @click="refreshData"
            >
              <span v-if="loading" class="spinner"></span>
              {{ loading ? 'Refreshing...' : 'Refresh' }}
            </button>
            <button class="btn btn-secondary" @click="exportData">
              Export CSV
            </button>
          </div>
        </div>

        <div class="date-range-row">
          <label>Date Range:</label>
          <input v-model="dateFrom" type="date" class="date-input">
          <input v-model="timeFrom" type="time" class="time-input">
          <span>to</span>
          <input v-model="dateTo" type="date" class="date-input">
          <input v-model="timeTo" type="time" class="time-input">
          <button class="btn btn-sm" @click="applyDateFilter">Apply</button>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="loading-container">
        <div class="loading-spinner"></div>
        <p>Loading CDR data...</p>
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="error-container">
        <div class="error-icon">⚠️</div>
        <h3>Error Loading CDR Data</h3>
        <p>{{ errorMessage }}</p>
        <button class="btn btn-primary" @click="refreshData">Try Again</button>
      </div>

      <!-- Data Table -->
      <div v-else class="table-container">
        <div class="table-header">
          <span class="results-count">
            Showing {{ filteredData.length }} of {{ totalRecords }} records
          </span>
          <div class="pagination-info">
            Page {{ currentPage }} of {{ totalPages }}
          </div>
        </div>

        <div class="table-wrapper">
          <table class="data-table">
            <thead>
              <tr>
                <th
                  v-for="header in visibleHeaders"
                  :key="header.key"
                  :class="{ 'sortable': header.sortable, 'sorted': sortColumn === header.key }"
                  @click="sortBy(header.key)"
                >
                  {{ header.label }}
                  <span v-if="sortColumn === header.key" class="sort-indicator">
                    {{ sortDirection === 'asc' ? '↑' : '↓' }}
                  </span>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(row, index) in paginatedData"
                :key="index"
                :class="{ 'row-highlight': row.disposition === 'FAILED' }"
              >
                <td v-for="header in visibleHeaders" :key="header.key">
                  <span
                    v-if="header.key === 'disposition'"
                    :class="'status-badge status-' + (row[header.key] || '').toLowerCase().replace(' ', '-')"
                  >
                    {{ row[header.key] || 'Unknown' }}
                  </span>
                  <span v-else-if="header.key === 'duration'">
                    {{ formatDuration(row[header.key]) }}
                  </span>
                  <span v-else-if="header.key === 'time'">
                    {{ formatDateTime(row[header.key]) }}
                  </span>
                  <span v-else>
                    {{ row[header.key] || '-' }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Pagination -->
        <div class="pagination-container">
          <button
            class="btn btn-pagination"
            :disabled="currentPage === 1"
            @click="changePage(currentPage - 1)"
          >
            Previous
          </button>

          <span class="page-numbers">
            <button
              v-for="page in visiblePages"
              :key="page"
              :class="{ 'active': page === currentPage }"
              class="btn btn-page"
              @click="changePage(page)"
            >
              {{ page }}
            </button>
          </span>

          <button
            class="btn btn-pagination"
            :disabled="currentPage === totalPages"
            @click="changePage(currentPage + 1)"
          >
            Next
          </button>
        </div>
      </div>
    </div>
  </wrap>
</template>

<script>
import base from './base'
// import YeastarService from '../../services/YeastarService'

export default {
  name: 'EnhancedCDRComponent',
  extends: base,

  data () {
    return {
      loading: true,
      error: false,
      errorMessage: '',
      contentBody: [],
      filteredData: [],
      searchQuery: '',
      statusFilter: '',
      directionFilter: '',
      dateFrom: '',
      dateTo: '',
      sortColumn: 'time',
      sortDirection: 'desc',
      currentPage: 1,
      timeFrom: '00:00',
      timeTo: '23:59',
      itemsPerPage: 50,

      // Define headers with better labels and formatting
      headerConfig: {
        time: { label: 'Call Time', sortable: true },
        call_from: { label: 'Caller ID', sortable: true },
        call_to: { label: 'Callee ID', sortable: true },
        duration: { label: 'Duration', sortable: true },
        disposition: { label: 'Status', sortable: true },
        call_type: { label: 'Direction', sortable: true },
        src_addr: { label: 'Source Address', sortable: false },
        reason: { label: 'Reason', sortable: false },
      },
    }
  },

  computed: {
    visibleHeaders () {
      return Object.keys(this.headerConfig).map(key => ({
        key,
        ...this.headerConfig[key],
      }))
    },

    totalRecords () {
      return this.contentBody.length
    },

    totalDuration () {
      const total = this.contentBody.reduce((sum, record) => {
        return sum + (parseInt(record.duration) || 0)
      }, 0)
      return this.formatDuration(total)
    },

    successRate () {
      if (this.totalRecords === 0) return 0
      const answered = this.contentBody.filter(record =>
        record.status === 'ANSWERED',
      ).length
      return Math.round((answered / this.totalRecords) * 100)
    },

    totalPages () {
      return Math.ceil(this.filteredData.length / this.itemsPerPage)
    },

    paginatedData () {
      const start = (this.currentPage - 1) * this.itemsPerPage
      const end = start + this.itemsPerPage
      return this.filteredData.slice(start, end)
    },

    visiblePages () {
      const pages = []
      const maxVisible = 5
      let start = Math.max(1, this.currentPage - Math.floor(maxVisible / 2))
      const end = Math.min(this.totalPages, start + maxVisible - 1)

      if (end - start + 1 < maxVisible) {
        start = Math.max(1, end - maxVisible + 1)
      }

      for (let i = start; i <= end; i++) {
        pages.push(i)
      }
      return pages
    },
  },

  watch: {
    searchQuery () {
      this.filterData()
    },
    statusFilter () {
      this.filterData()
    },
    directionFilter () {
      this.filterData()
    },
    sortColumn () {
      this.sortData()
    },
    sortDirection () {
      this.sortData()
    },
  },

  async mounted () {
    await this.loadData()
  },

  methods: {
    async loadData () {
      // try {
      //   this.loading = true
      //   this.error = false

      //   const service = new YeastarService()
      //   const response = await service.getCDRDB()

      //   if (Array.isArray(response) && response.length > 0) {
      //     this.contentBody = response
      //     this.filterData()
      //   } else {
      //     this.error = true
      //     this.errorMessage = 'No CDR data available'
      //   }
      // } catch (error) {
      //   console.error('Error fetching CDR data:', error)
      //   this.error = true
      //   this.errorMessage = error.message || 'Failed to load CDR data'
      // } finally {
      //   this.loading = false
      // }
    },

    async refreshData () {
      await this.loadData()
    },

    parseDateTime (dateStr, timeStr = '00:00') {
      if (!dateStr) return null

      // Combine date and time
      const dateTimeStr = `${dateStr} ${timeStr}`

      // Handle DD/MM/YYYY format from API
      if (dateStr.includes('/')) {
        const [day, month, year] = dateStr.split('/')
        return new Date(`${month}/${day}/${year} ${timeStr}`)
      }

      // Fallback to native parsing
      return new Date(dateTimeStr)
    },

    filterData () {
      let filtered = [...this.contentBody]

      // Search filter
      if (this.dateFrom || this.dateTo) {
        const fromDate = this.dateFrom
          ? this.parseDateTime(this.dateFrom, this.timeFrom)
          : null

        const toDate = this.dateTo
          ? this.parseDateTime(this.dateTo, this.timeTo)
          : null

        filtered = filtered.filter(record => {
          const recordDate = this.parseDateTime(record.time.split(' ')[0], record.time.split(' ')[1])

          if (fromDate && recordDate < fromDate) return false
          if (toDate && recordDate > toDate) return false
          return true
        })
      }

      // Status filter
      if (this.statusFilter) {
        filtered = filtered.filter(record => record.disposition === this.statusFilter)
      }

      // Direction filter
      if (this.directionFilter) {
        filtered = filtered.filter(record => record.call_type === this.directionFilter)
      }

      // Date range filter
      if (this.dateFrom || this.dateTo) {
        filtered = filtered.filter(record => {
          const recordDate = new Date(record.time)
          const fromDate = this.dateFrom ? new Date(this.dateFrom) : null
          const toDate = this.dateTo ? new Date(this.dateTo) : null

          if (fromDate && recordDate < fromDate) return false
          if (toDate && recordDate > toDate) return false
          return true
        })
      }

      console.log('Filtered data:', filtered)
      this.filteredData = filtered
      this.sortData()
      this.currentPage = 1
    },

    sortData () {
      if (!this.sortColumn) return

      this.filteredData.sort((a, b) => {
        let aVal = a[this.sortColumn]
        let bVal = b[this.sortColumn]

        // Handle different data types
        if (this.sortColumn === 'duration' || this.sortColumn === 'billing_seconds') {
          aVal = parseInt(aVal) || 0
          bVal = parseInt(bVal) || 0
        } else if (this.sortColumn === 'call_time') {
          aVal = new Date(aVal)
          bVal = new Date(bVal)
        } else {
          aVal = String(aVal || '').toLowerCase()
          bVal = String(bVal || '').toLowerCase()
        }

        if (aVal < bVal) return this.sortDirection === 'asc' ? -1 : 1
        if (aVal > bVal) return this.sortDirection === 'asc' ? 1 : -1
        return 0
      })
    },

    sortBy (column) {
      if (this.sortColumn === column) {
        this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc'
      } else {
        this.sortColumn = column
        this.sortDirection = 'asc'
      }
    },

    changePage (page) {
      if (page >= 1 && page <= this.totalPages) {
        this.currentPage = page
      }
    },

    applyDateFilter () {
      this.filterData()
    },

    formatDuration (seconds) {
      const hrs = Math.floor(seconds / 3600)
      const mins = Math.floor((seconds % 3600) / 60)
      const secs = seconds % 60

      if (hrs > 0) {
        return `${hrs}h ${mins}m ${secs}s`
      } else if (mins > 0) {
        return `${mins}m ${secs}s`
      } else {
        return `${secs}s`
      }
    },

    formatDateTime (dateTime) {
      if (!dateTime) return '-'
      const [datePart, timePart] = dateTime.split(' ')
      const [day, month, year] = datePart.split('/')
      return new Date(`${month}/${day}/${year} ${timePart}`).toLocaleString()
    },

    exportData () {
      const csvContent = this.convertToCSV(this.filteredData)
      const blob = new Blob([csvContent], { type: 'text/csv' })
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.style.display = 'none'
      a.href = url
      a.download = `cdr_export_${new Date().toISOString().split('T')[0]}.csv`
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)
    },

    convertToCSV (data) {
      if (!data.length) return ''

      const headers = Object.keys(this.headerConfig).map(key => this.headerConfig[key].label)
      const rows = data.map(row =>
        Object.keys(this.headerConfig).map(key => {
          const value = row[key] || ''
          return `"${String(value).replace(/"/g, '""')}"`
        }).join(','),
      )

      return [headers.join(','), ...rows].join('\n')
    },
  },
}
</script>

<style scoped>
.cdr-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 1rem;
  font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
}

/* Header Section */
.header-section {
  margin-bottom: 2rem;
}

.title {
  color: #2c3e50;
  font-size: 2rem;
  margin-bottom: 1rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.icon-phone::before {
  content: '📞';
  font-size: 1.5rem;
}

.stats-row {
  display: flex;
  gap: 1rem;
  margin-bottom: 1rem;
}

.stat-card {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 1rem;
  border-radius: 8px;
  text-align: center;
  flex: 1;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
}

.stat-value {
  display: block;
  font-size: 1.5rem;
  font-weight: bold;
  margin-bottom: 0.25rem;
}

.stat-label {
  font-size: 0.875rem;
  opacity: 0.9;
}

/* Controls Section */
.controls-section {
  background: #f8f9fa;
  padding: 1.5rem;
  border-radius: 8px;
  margin-bottom: 2rem;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.search-filter-row {
  display: flex;
  gap: 1rem;
  margin-bottom: 1rem;
  align-items: center;
  flex-wrap: wrap;
}

.search-box {
  position: relative;
  flex: 1;
  min-width: 200px;
}

.search-input {
  width: 100%;
  padding: 0.5rem 2.5rem 0.5rem 0.75rem;
  border: 2px solid #e9ecef;
  border-radius: 6px;
  font-size: 0.875rem;
  transition: border-color 0.2s;
}

.search-input:focus {
  outline: none;
  border-color: #007bff;
}

.search-icon {
  position: absolute;
  right: 0.75rem;
  top: 50%;
  transform: translateY(-50%);
  opacity: 0.5;
}

.filter-group {
  display: flex;
  gap: 0.5rem;
}

.filter-select {
  padding: 0.5rem;
  border: 2px solid #e9ecef;
  border-radius: 6px;
  font-size: 0.875rem;
  background: white;
}

.action-buttons {
  display: flex;
  gap: 0.5rem;
}

.date-range-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.date-input {
  padding: 0.5rem;
  border: 2px solid #e9ecef;
  border-radius: 6px;
  font-size: 0.875rem;
}

/* Buttons */
.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.875rem;
  transition: all 0.2s;
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary {
  background: #007bff;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #0056b3;
}

.btn-secondary {
  background: #6c757d;
  color: white;
}

.btn-secondary:hover {
  background: #545b62;
}

.btn-sm {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
}

.btn-pagination {
  background: #e9ecef;
  color: #495057;
}

.btn-pagination:hover:not(:disabled) {
  background: #dee2e6;
}

.btn-page {
  background: #f8f9fa;
  color: #495057;
  min-width: 2rem;
}

.btn-page.active {
  background: #007bff;
  color: white;
}

.spinner {
  width: 1rem;
  height: 1rem;
  border: 2px solid transparent;
  border-top: 2px solid currentColor;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Loading and Error States */
.loading-container, .error-container {
  text-align: center;
  padding: 3rem;
}

.loading-spinner {
  width: 3rem;
  height: 3rem;
  border: 4px solid #f3f3f3;
  border-top: 4px solid #007bff;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 1rem;
}

.error-container {
  color: #721c24;
}

.error-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
}

/* Table Styles */
.table-container {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  background: #f8f9fa;
  border-bottom: 1px solid #e9ecef;
}

.results-count {
  font-weight: 500;
  color: #495057;
}

.pagination-info {
  color: #6c757d;
  font-size: 0.875rem;
}

.table-wrapper {
  overflow-x: auto;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
}

.data-table th {
  background: #007bff;
  color: white;
  padding: 0.75rem;
  text-align: left;
  font-weight: 500;
  border-bottom: 2px solid #0056b3;
}

.data-table th.sortable {
  cursor: pointer;
  user-select: none;
  transition: background-color 0.2s;
}

.data-table th.sortable:hover {
  background: #0056b3;
}

.data-table th.sorted {
  background: #0056b3;
}

.sort-indicator {
  margin-left: 0.5rem;
  font-size: 0.875rem;
}

.data-table td {
  padding: 0.75rem;
  border-bottom: 1px solid #e9ecef;
  color: #495057;
}

.data-table tr:hover {
  background: #f8f9fa;
}

.row-highlight {
  background: #f8d7da !important;
}

/* Status Badges */
.status-badge {
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
}

.status-answered {
  background: #d4edda;
  color: #155724;
}

.status-no-answer {
  background: #fff3cd;
  color: #856404;
}

.status-busy {
  background: #f8d7da;
  color: #721c24;
}

.status-failed {
  background: #f5c6cb;
  color: #721c24;
}

/* Pagination */
.pagination-container {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 0.5rem;
  padding: 1rem;
  background: #f8f9fa;
  border-top: 1px solid #e9ecef;
}

.page-numbers {
  display: flex;
  gap: 0.25rem;
}

.time-input {
  padding: 0.5rem;
  border: 2px solid #e9ecef;
  border-radius: 6px;
  font-size: 0.875rem;
  margin-left: 0.25rem;
}

/* Responsive Design */
@media (max-width: 768px) {
  .cdr-container {
    padding: 0.5rem;
  }

  .stats-row {
    flex-direction: column;
  }

  .search-filter-row {
    flex-direction: column;
    align-items: stretch;
  }

  .search-box {
    min-width: auto;
  }

  .filter-group {
    flex-direction: column;
  }

  .action-buttons {
    justify-content: center;
  }

  .table-header {
    flex-direction: column;
    align-items: stretch;
    gap: 0.5rem;
  }

  .pagination-container {
    flex-wrap: wrap;
  }
}
</style>
