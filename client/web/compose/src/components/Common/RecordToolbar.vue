<template>
  <c-toolbar
    :class="{ 'shadow border-top': !inModal }"
    style="min-height: 73px;"
  >
    <template #start>
      <b-button
        v-if="!(hideBack || settings.hideBack)"
        data-test-id="button-back"
        :disabled="isProcessing"
        variant="outline-light"
        size="lg"
        class="border-0 text-dark back"
        @click.prevent="$emit('back')"
      >
        <span class="d-flex align-items-center gap-1">
          <font-awesome-icon
            :icon="['fas', hasBack ? 'chevron-left' : 'times']"
            :class="hasBack ? 'back-icon' : ''"
          />
          {{ backLabel }}
        </span>
      </b-button>

      <slot name="start-actions" />
    </template>

    <template #center>
      <div
        v-if="isCreated && (recordNavigation.prev || recordNavigation.next)"
        class="d-flex align-items-center fill-width gap-1"
      >
        <span v-b-tooltip.noninteractive.hover="{ title: $t('recordNavigation.prev'), container: '#body' }">
          <b-button
            pill
            size="lg"
            variant="outline-primary"
            :disabled="!record || isProcessing || !recordNavigation.prev"
            class="w-100"
            @click="navigateToRecord(recordNavigation.prev)"
          >
            <font-awesome-icon :icon="['fas', 'angle-left']" />
          </b-button>
        </span>

        <span v-b-tooltip.noninteractive.hover="{ title: $t('recordNavigation.next'), container: '#body' }">
          <b-button
            pill
            size="lg"
            variant="outline-primary"
            :disabled="!record || isProcessing || !recordNavigation.next"
            class="w-100"
            @click="navigateToRecord(recordNavigation.next)"
          >
            <font-awesome-icon :icon="['fas', 'angle-right']" />
          </b-button>
        </span>
      </div>

      <slot name="center-actions" />
    </template>

    <template
      v-if="module"
      #end
    >
      <slot name="end-actions" />

      <c-input-confirm
        v-if="(processingAction === 'delete' || isCreated) && !(isDeleted || hideDelete || settings.hideDelete) && canDeleteRecord"
        :disabled="!record || isProcessing"
        :processing="processingAction === 'delete'"
        :text="labels.delete || $t('label.delete')"
        size="lg"
        size-confirm="lg"
        variant="danger"
        @confirmed="$emit('delete')"
      />

      <c-input-confirm
        v-else-if="(processingAction === 'undelete' || isDeleted) && !(hideDelete || settings.hideDelete) && canUndeleteRecord"
        :disabled="!record || isProcessing"
        :processing="processingAction === 'undelete'"
        :text="$t('label.restore')"
        size="lg"
        size-confirm="lg"
        variant="warning"
        variant-ok="warning"
        @confirmed="$emit('undelete')"
      />

      <b-button
        v-if="isCreated && module.canCreateRecord && !(hideClone || settings.hideClone)"
        data-test-id="button-clone"
        variant="light"
        size="lg"
        :disabled="!record || isProcessing"
        class="text-nowrap"
        @click.prevent="$emit('clone')"
      >
        {{ labels.clone || $t('label.saveAsCopy') }}
      </b-button>

      <b-button
        v-if="!inEditing && isCreated && !(hideEdit || settings.hideEdit) && canManageRecord"
        data-test-id="button-edit"
        :disabled="!record || isProcessing"
        variant="light"
        size="lg"
        @click.prevent="$emit('edit')"
      >
        {{ labels.edit || $t('label.edit') }}
      </b-button>

      <b-button
        v-else-if="inEditing && isCreated && !(hideEdit || settings.hideEdit)"
        data-test-id="button-view"
        :disabled="!record || isProcessing"
        variant="light"
        size="lg"
        @click.prevent="$emit('view')"
      >
        {{ labels.edit || $t('label.view') }}
      </b-button>

      <b-button
        v-if="!inEditing && module.canCreateRecord && !(hideNew || settings.hideNew)"
        data-test-id="button-add-new"
        variant="primary"
        size="lg"
        :disabled="!record || isProcessing"
        class="text-nowrap"
        @click.prevent="$emit('add')"
      >
        {{ labels.new || $t('label.addNew') }}
      </b-button>

      <c-button-submit
        v-if="inEditing && !(hideSubmit || settings.hideSubmit) && canManageRecord"
        data-test-id="button-save"
        :disabled="!record || isProcessing"
        :processing="processingAction === 'submit'"
        :text="labels.submit || $t('label.save')"
        size="lg"
        @submit="$emit('submit')"
      />
    </template>
  </c-toolbar>
</template>

<script>
import { components } from '@cortezaproject/corteza-vue'
import { compose, NoID } from '@cortezaproject/corteza-js'
import { throttle } from 'lodash'
const { CToolbar } = components

export default {
  i18nOptions: {
    namespaces: 'general',
  },

  components: {
    CToolbar,
  },

  props: {
    module: {
      type: compose.Module,
      required: false,
      default: undefined,
    },

    record: {
      type: compose.Record,
      required: false,
      default: undefined,
    },

    labels: {
      type: Object,
      default: () => ({}),
    },

    processing: {
      type: Boolean,
      default: false,
    },

    processingAction: {
      type: String,
      default: '',
    },

    isCreated: {
      type: Boolean,
      required: true,
    },

    inEditing: {
      type: Boolean,
      required: true,
    },

    hideBack: {
      type: Boolean,
      default: () => true,
    },

    hideDelete: {
      type: Boolean,
      default: () => true,
    },

    hideNew: {
      type: Boolean,
      default: () => true,
    },

    hideClone: {
      type: Boolean,
      default: () => true,
    },

    hideEdit: {
      type: Boolean,
      default: () => true,
    },

    hideSubmit: {
      type: Boolean,
      default: () => true,
    },

    inModal: {
      type: Boolean,
      required: false,
    },

    recordNavigation: {
      type: Object,
      required: false,
      default: () => ({}),
    },

    hasBack: {
      type: Boolean,
      default: true,
    },
  },

  computed: {
    isDeleted () {
      return this.record && this.record.deletedAt
    },

    isProcessing () {
      return this.processing || !!this.processingAction
    },

    settings () {
      return this.$Settings.get('compose.ui.record-toolbar', {})
    },

    canManageRecord () {
      if (!this.module || !this.record) {
        return false
      }

      return this.record.recordID === NoID
        ? this.module.canCreateRecord
        : this.record.canUpdateRecord
    },

    canDeleteRecord () {
      if (!this.module || !this.record) {
        return false
      }

      return !this.isDeleted && this.record.canDeleteRecord && this.record.recordID !== NoID
    },

    canUndeleteRecord () {
      if (!this.module || !this.record) {
        return false
      }

      return this.isDeleted && this.record.canUndeleteRecord && this.record.recordID !== NoID
    },

    backLabel () {
      if (this.inModal) {
        return this.hasBack ? this.$t('label.back') : this.$t('label.close')
      }

      return this.hasBack ? this.labels.back || this.$t('label.back') : this.$t('label.home')
    },
  },

  methods: {
    navigateToRecord: throttle(function (recordID) {
      this.$emit('update-navigation', recordID)
    }, 500),
  },
}
</script>
