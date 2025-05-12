<template>
  <div class="d-flex flex-column gap-1">
    <p
      v-if="!!message"
      class="text-break"
      v-html="message"
    />

    <b-form-group
      :label="label"
      label-class="text-primary"
    >
      <b-input
        v-model="value"
        :type="type"
        :disabled="loading"
      />
    </b-form-group>

    <b-button
      :disabled="loading"
      variant="primary"
      class="ml-auto"
      @click="$emit('submit', { value: { '@value': value, '@type': 'String' }})"
    >
      {{ pVal('buttonLabel', 'Submit') }}
    </b-button>
  </div>
</template>

<script lang="js">
import base from './base.vue'

const validTypes = [
  'text',
  'number',
  'email',
  'password',
  'search',
  'date',
  'time',
]

export default {
  name: 'CPromptInput',

  extends: base,

  data () {
    return {
      value: undefined,
    }
  },

  computed: {
    type () {
      const t = this.pVal('type', 'text')
      if (validTypes.indexOf(t) === -1) {
        return 'text'
      }

      return t
    },

    label () {
      return this.pVal('label', '')
    },
  },

  beforeMount () {
    this.value = this.pVal('inputValue')
  },

}
</script>
