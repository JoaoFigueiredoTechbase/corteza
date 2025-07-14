<template>
  <wrap
    v-bind="$props"
    v-on="$listeners"
  >
    <div class="button-box">
      <button @click="handleClick">
        {{ label }}
      </button>
    </div>
  </wrap>
</template>

<script>
import base from './base'
import YeastarService from '../../services/YeastarService'

export default {
  name: 'SyncBoxBase',
  extends: base,

  // props: {
  //   label: {
  //     type: String,
  //     default: 'Synchronize',
  //   },
  // },

  methods: {
    async handleClick () {
      try {
        const message = await YeastarService.syncALL()
        console.error('message: ', message)
      } catch (error) {
        console.error('Error syncing CDR data:', error)
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

button {
  background-color: #007bff;
  color: white;
  border: none;
  padding: 0.6rem 1.2rem;
  font-size: 1rem;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.2s ease;
}

button:hover {
  background-color: #0056b3;
}
</style>
