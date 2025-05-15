<template>
  <b-overlay>
    <b-card-header class="border-bottom">
      <div class="d-flex flex-wrap align-items-center justify-content-between gap-2">
        <h5
          class="text-primary text-capitalize text-truncate mb-0"
        >
          <span
            v-if="hit.value.namespace.name || hit.value.namespace.handle"
          >
            {{ hit.value.namespace.name || hit.value.namespace.handle }}
          </span>
          <span
            v-if="hit.value.namespace.name || hit.value.namespace.handle"
            class="mx-2"
          >
            <b-icon
              icon="arrow-right"
              font-scale="1"
            />
          </span>
          <span>
            {{ hit.value.module.name || hit.value.module.handle }}
          </span>
        </h5>

        <span class="text-nowrap">
          <b-badge
            v-if="Object.keys(hit.value.labels || { }).includes('federation')"
            variant="light"
            class="mr-1 h5 p-2 mb-0"
          >
            {{ $t('general:federated') }}
          </b-badge>
          <b-avatar
            size="sm"
            icon="file-earmark-text"
            class="align-center bg-light text-dark"
          />
          {{ $t('types.record') }}
        </span>
      </div>

      <div class="d-flex justify-content-between small">
        <slot name="header" />
      </div>
    </b-card-header>

    <b-card-body class="d-flex flex-column flex-wrap gap-3">
      <div
        class="d-flex flex-wrap gap-2 flex-grow-1"
      >
        <b-form-group
          v-for="(item, i) in recordValues"
          :key="i"
          :label="item.label || item.name"
          label-class="text-capitalize text-primary"
          class="mb-0"
          style="min-width: 20rem; max-width: 100%; white-space: pre-line;"
        >
          {{ item.value }}
        </b-form-group>
      </div>

      <div class="d-flex flex-wrap gap-2 flex-grow-1">
        <b-form-group
          v-for="item in systemValues"
          :key="item.name"
          :label="item.label"
          label-class="text-capitalize text-primary"
          class="mb-0"
          style="min-width: 20rem; max-width: 100%;"
        >
          {{ item.value }}
        </b-form-group>
      </div>
    </b-card-body>
  </b-overlay>
</template>

<script>
import base from './base'

export default {
  i18nOptions: {
    namespaces: 'filters',
  },

  extends: base,

  computed: {
    recordID () {
      return this.hit.value.recordID
    },

    recordValues () {
      const { values = [] } = this.hit.value

      return (values || []).map(({ name, label, value = [] }) => {
        if (value) {
          value = value.map(v => {
            return v.toString().includes('{"coordinates":[') ? ((JSON.parse(v || '{}') || {}).coordinates || []).join(', ') : v
          }).join('\n')
        }

        return { name, label, value }
      })
    },

    systemValues () {
      return [
        { name: 'recordID', label: this.$t('general:recordID'), value: this.recordID },
        { name: 'createdBy', label: this.$t('general:createdBy'), value: this.createdBy },
        { name: 'createdAt', label: this.$t('general:createdAt'), value: this.createdAt },
        { name: 'updatedAt', label: this.$t('general:updatedAt'), value: this.updatedAt },
      ].filter(v => v.value)
    },
  },
}
</script>
