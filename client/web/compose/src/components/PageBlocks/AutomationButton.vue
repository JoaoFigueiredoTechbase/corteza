<template>
  <b-button
    v-if="button && button.workflowID"
    :variant="button.variant || 'primary'"
    :size="button.size || 'sm'"
    :disabled="processing || !button.enabled"
    @click="handleClick"
  >
    <b-spinner
      v-if="processing"
      small
      class="mr-1"
    />

    <font-awesome-icon
      v-else-if="button.icon"
      :icon="button.icon"
      class="mr-1"
    />

    {{ button.label }}
  </b-button>
</template>

<script>
export default {
  name: 'AutomationButton',

  props: {
    button: {
      type: Object,
      required: true,
    },

    module: {
      type: Object,
      required: true,
    },

    record: {
      type: Object,
      required: true,
    },

    namespace: {
      type: Object,
      required: true,
    },

    extraEventArgs: {
      type: Object,
      default: () => ({}),
    },

    size: {
      type: String,
      default: 'md',
    },
  },

  data () {
    return {
      processing: false,
    }
  },

  methods: {
    async handleClick () {
      if (!this.button.workflowID || this.processing) {
        return
      }

      this.processing = true

      try {
        // Prepare the arguments to send to the workflow
        const args = {
          record: this.record,
          recordID: this.record.recordID,
          module: this.module,
          namespace: this.namespace,
          ...this.extraEventArgs,
        }

        // Execute the workflow
        await this.$AutomationAPI.workflowExec({
          workflowID: this.button.workflowID,
          args,
        })

        this.toastSuccess(this.$t('notification:automation.workflowExecuted'))

        // Emit refresh event to parent component
        this.$emit('refresh')
      } catch (error) {
        this.toastErrorHandler(this.$t('notification:automation.workflowExecutionFailed'))(error)
      } finally {
        this.processing = false
      }
    },
  },
}
</script>
