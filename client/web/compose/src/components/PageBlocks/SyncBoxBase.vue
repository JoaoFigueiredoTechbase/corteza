<template>
  <wrap
    v-bind="$props"
    v-on="$listeners"
  >
    <div class="button-box">
      <button
        :disabled="loading"
        class="sync-button"
        @click="handleClick"
      >
        <span v-if="loading" class="spinner" />
        <span v-else>{{ label }}</span>
      </button>
    </div>
  </wrap>
</template>

<script>
import base from './base'
import YeastarService from '../../services/YeastarService'

const yeastar = new YeastarService()

export default {
  name: 'SyncBoxBase',
  extends: base,

  props: {
    label: {
      type: String,
      default: 'Synchronize',
    },
  },

  data () {
    return {
      loading: false,
    }
  },

  methods: {
    async handleClick () {
      this.loading = true
      try {
        const message = await yeastar.syncALL()
        console.log('Sync successful:', message)
        // Optionally trigger toast/snackbar here
      } catch (error) {
        console.error('Error syncing CDR data:', error)
        // Optionally show error feedback
      } finally {
        this.loading = false
      }
    },
  },
}
</script>

<style scoped>
.button-box {
  padding: 1rem;
  text-align: center;
}

.sync-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background-color: #007bff;
  color: white;
  border: none;
  padding: 0.6rem 1.4rem;
  font-size: 1rem;
  border-radius: 6px;
  cursor: pointer;
  transition: background-color 0.2s ease;
  min-width: 140px;
  position: relative;
}

.sync-button:disabled {
  background-color: #9bbff9;
  cursor: not-allowed;
}

.sync-button:hover:not(:disabled) {
  background-color: #0056b3;
}

.spinner {
  width: 1.2rem;
  height: 1.2rem;
  border: 2px solid white;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
