<template>
  <wrap v-bind="$props"
    v-on="$listeners">
    <div class="hello-box">
      <div v-if="loading">Loading...</div>
      <div v-else-if="error"><strong>Error loading CDR data.</strong></div>
      <div v-else>
        <table>
          <thead>
            <tr>
              <th v-for="header in tableHeaders" :key="header">{{ header }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in contentBody" :key="index">
              <td v-for="header in tableHeaders" :key="header">{{ row[header] }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </wrap>
</template>

<script>
import base from './base'
import YeastarService from '../../services/YeastarService'

export default {
  name: 'HelloBoxBase',
  extends: base,

  data () {
    return {
      loading: true,
      error: false,
      contentBody: [],
      tableHeaders: [],
    }
  },

  async mounted () {
    try {
      const service = new YeastarService()
      const response = await service.getCDRDB()

      //  await service.fetchCDR()
      const data = response

      // Ensure we received an array
      if (Array.isArray(data) && data.length > 0) {
        this.contentBody = data
        this.tableHeaders = Object.keys(data[0])
      } else {
        this.error = true
      }
    } catch (error) {
      console.error('Error fetching CDR data:', error)
      this.error = true
    } finally {
      this.loading = false
    }
  },
}
</script>

<style scoped>
.hello-box {
  padding: 2rem;
  font-size: 1rem;
  background: #f8f9fa;
  border: 1px solid #ddd;
  border-radius: 6px;
}

table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 1rem;
}

th, td {
  padding: 0.5rem;
  border: 1px solid #ccc;
}

th {
  background-color: #007bff;
  color: white;
}
</style>
