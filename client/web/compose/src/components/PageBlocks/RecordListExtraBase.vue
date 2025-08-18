<template>
  <wrap
    v-bind="$props"
    :scrollable-body="false"
    v-on="$listeners"
    @refreshBlock="refresh(true, false)"
  >
    <template #toolbar>
      <b-container
        v-if="recordListModule"
        ref="toolbar"
        fluid
        class="d-flex flex-column gap-2 p-3 d-print-none"
      >
        <b-row
          no-gutters
          class="d-flex align-items-center justify-content-between gap-1"
        >
          <div class="d-flex align-items-center flex-grow-1 flex-wrap flex-fill-child gap-1">
            <exporter-modal
              v-if="options.allowExport"
              :module="recordListModule"
              :filter="filter.query"
              :selection="selected"
              :selected-all-records="selectedAllRecords"
              :processing="processing"
              :preselected-fields="fields.map(({ moduleField }) => moduleField)"
              @export="onExport"
            />

            <column-picker
              v-if="!options.hideConfigureFieldsButton"
              :module="recordListModule"
              :fields="fields.map(({ moduleField }) => moduleField)"
              @updateFields="onUpdateFields"
            />
          </div>
          <div
            v-if="!options.hideSearch"
            class="flex-fill"
          >
            <c-input-search
              v-model.trim="query"
              :placeholder="$t('general.label.search')"
              :debounce="500"
            />
          </div>
        </b-row>
      </b-container>
    </template>

    <template #default>
      <div
        v-if="recordListModule"
        class="d-flex position-relative h-100"
        :class="{ 'overflow-hidden': !items.length || isProcessing }"
      >
        <b-table-simple
          data-test-id="table-record-crud"
          hover
          responsive
          sticky-header
          class="record-crud-table mh-100 h-100 mb-0"
        >
          <b-thead>
            <b-tr>
              <b-th
                v-if="options.selectable"
                style="width: 0%;"
                class="d-print-none"
              >
                <b-checkbox
                  :disabled="disableSelectAll"
                  :checked="areAllRowsSelected && !disableSelectAll"
                  class="ml-1"
                  @change="handleSelectAllOnPage({ isChecked: $event })"
                />
              </b-th>

              <b-th
                v-for="field in fields"
                :key="field.key"
                sticky-column
              >
                <div class="d-flex align-items-center">
                  <div
                    :class="{ required: field.required }"
                    class="d-flex align-self-center text-nowrap"
                  >
                    {{ field.label }}
                  </div>

                  <b-button
                    v-if="field.sortable"
                    v-b-tooltip.noninteractive.hover="{ title: $t('recordList.sort.tooltip'), container: '#body' }"
                    variant="outline-extra-light"
                    class="d-flex align-items-center text-secondary d-print-none border-0 px-1 ml-1"
                    @click="handleSort(field)"
                  >
                    <font-awesome-layers>
                      <font-awesome-icon
                        :icon="['fas', 'angle-up']"
                        class="mb-1"
                        :class="{ 'text-primary': isSortedBy(field, 'ASC') }"
                      />
                      <font-awesome-icon
                        :icon="['fas', 'angle-down']"
                        class="mt-1"
                        :class="{ 'text-primary': isSortedBy(field, 'DESC') }"
                      />
                    </font-awesome-layers>
                  </b-button>
                </div>
              </b-th>

              <!-- Actions columns - one for each button -->
              <b-th
                v-for="(button, index) in options.rowButtons"
                :key="`action-${index}`"
                class="text-center"
                style="width: 120px;"
              >
                {{ button.label }}
              </b-th>
            </b-tr>
          </b-thead>

          <b-tbody v-if="items.length && !isProcessing && !resizing">
            <b-tr
              v-for="item in items"
              :key="item.r.recordID"
            >
              <b-td
                v-if="options.selectable"
                class="pr-0 d-print-none"
                @click.stop
              >
                <b-form-checkbox
                  class="ml-1"
                  :checked="selected.includes(item.id)"
                  @change="onSelectRow($event, item)"
                />
              </b-td>

              <b-td
                v-for="field in fields"
                :key="field.key"
              >
                <div
                  v-if="field.moduleField.canReadRecordValue"
                  class="d-flex mb-0"
                  style="min-width: 10rem;"
                >
                  <field-viewer
                    :field="field.moduleField"
                    value-only
                    :record="item.r"
                    :module="module"
                    :namespace="namespace"
                    :extra-options="options"
                    include-styles
                  />
                </div>

                <i
                  v-else
                  class="text-primary"
                >
                  {{ $t('field.noPermission') }}
                </i>
              </b-td>

              <!-- Action buttons - one column per button -->
              <b-td
                v-for="(button, index) in options.rowButtons"
                :key="`action-${index}-${item.r.recordID}`"
                class="text-center"
              >
                <automation-button
                  :button="button"
                  :module="recordListModule"
                  :record="item.r"
                  :namespace="namespace"
                  :extra-event-args="{ recordID: item.r.recordID, record: item.r }"
                  size="sm"
                  class="px-2"
                  v-bind="$props"
                  @refresh="refresh()"
                />
              </b-td>
            </b-tr>
          </b-tbody>

          <div
            v-else
            class="position-absolute text-center mt-5 d-print-none"
            style="left: 0; right: 0; bottom: calc(50% - 33px);"
          >
            <b-spinner
              v-if="isProcessing"
            />

            <p
              v-else-if="!items.length"
              class="mb-0 mx-2"
            >
              {{ $t('recordList.noRecords') }}
            </p>
          </div>
        </b-table-simple>
      </div>

      <label
        v-else
        class="text-primary p-3"
      >
        {{ $t('recordList.noModule') }}
      </label>
    </template>

    <template
      v-if="recordListModule && showFooter"
      #footer
    >
      <div
        v-if="showPagination"
        class="record-crud-footer d-flex align-items-center flex-wrap justify-content-between px-3 py-2 gap-1"
      >
        <div class="d-flex align-items-center flex-wrap gap-3 gap-col-3">
          <div
            v-if="options.showTotalCount"
            class="text-nowrap text-truncate"
          >
            <span
              v-if="pagination.count > recordsPerPage"
              data-test-id="pagination-range"
            >
              {{ $t('recordList.pagination.showing', getPagination) }}
            </span>

            <span
              v-else
              data-test-id="pagination-single-number"
            >
              {{ $t('recordList.pagination.single', getPagination) }}
            </span>
          </div>

          <div
            v-if="options.showRecordPerPageOption"
            class="d-flex align-items-center gap-1 text-nowrap"
          >
            <span>
              {{ $t('recordList.pagination.recordsPerPage') }}
            </span>

            <b-form-select
              v-model="recordsPerPage"
              :options="perPageOptions"
              size="sm"
              @change="handlePerPageChange"
            />
          </div>
        </div>

        <div
          v-if="showPageNavigation"
          class="d-flex align-items-center justify-content-end"
        >
          <b-pagination
            v-if="options.fullPageNavigation"
            data-test-id="pagination"
            align="right"
            class="m-0 d-print-none"
            pills
            :disabled="isProcessing"
            :value="getPagination.page"
            :per-page="getPagination.perPage"
            :total-rows="getPagination.count"
            @change="goToPage"
          />

          <b-button-group v-else class="gap-1">
            <b-button
              :disabled="!hasPrevPage || isProcessing"
              variant="outline-extra-light"
              class="d-flex align-items-center justify-content-center text-dark border-0 p-1"
              @click="goToPage()"
            >
              <font-awesome-icon :icon="['fas', 'angle-double-left']" />
            </b-button>

            <b-button
              :disabled="!hasPrevPage || isProcessing"
              variant="outline-extra-light"
              class="d-flex align-items-center justify-content-center text-dark border-0 p-1"
              @click="goToPage('prevPage')"
            >
              <font-awesome-icon :icon="['fas', 'angle-left']" class="mr-1" />
              {{ $t('recordList.pagination.prev') }}
            </b-button>

            <b-button
              :disabled="!hasNextPage || isProcessing"
              variant="outline-extra-light"
              class="d-flex align-items-center justify-content-center text-dark border-0 p-1"
              @click="goToPage('nextPage')"
            >
              {{ $t('recordList.pagination.next') }}
              <font-awesome-icon :icon="['fas', 'angle-right']" class="ml-1" />
            </b-button>
          </b-button-group>
        </div>
      </div>
    </template>
  </wrap>
</template>

<script>
import axios from 'axios'
import { mapGetters, mapActions } from 'vuex'
import base from 'corteza-webapp-compose/src/components/PageBlocks/base'
import FieldViewer from 'corteza-webapp-compose/src/components/ModuleFields/Viewer'
import ExporterModal from 'corteza-webapp-compose/src/components/Public/Record/Exporter'
import AutomationButton from 'corteza-webapp-compose/src/components/PageBlocks/AutomationButton.vue'
import { compose } from '@cortezaproject/corteza-js'
import users from 'corteza-webapp-compose/src/mixins/users'
import records from 'corteza-webapp-compose/src/mixins/records'
import { components } from '@cortezaproject/corteza-vue'
import ColumnPicker from 'corteza-webapp-compose/src/components/Admin/Module/Records/ColumnPicker'

const { CInputSearch } = components

export default {
  i18nOptions: {
    namespaces: 'block',
  },

  components: {
    ExporterModal,
    AutomationButton,
    FieldViewer,
    ColumnPicker,
    CInputSearch,
  },

  extends: base,

  mixins: [
    users,
    records,
  ],

  data () {
    return {
      uniqueID: undefined,
      processing: false,
      prefilter: undefined,
      query: null,

      filter: {
        query: '',
        sort: '',
        limit: 10,
        pageCursor: '',
        prevPage: '',
        nextPage: '',
      },

      pagination: {
        pages: [],
        page: 1,
        count: 0,
      },

      selected: [],
      selectedAllRecords: false,
      sortBy: undefined,
      sortDirecton: undefined,
      ctr: 0,
      items: [],
      abortableRequests: [],
      recordsPerPage: undefined,
      customConfiguredFields: [],
    }
  },

  computed: {
    ...mapGetters({
      getModuleByID: 'module/getByID',
    }),

    showPagination () {
      return this.showPageNavigation || this.options.showTotalCount || this.options.showRecordPerPageOption
    },

    showFooter () {
      return this.showPagination
    },

    perPageOptions () {
      const defaultText = this.options.perPage === 0 ? this.$t('general:label.all') : this.options.perPage.toString()
      return [
        { text: defaultText, value: this.options.perPage },
        { text: '25', value: 25 },
        { text: '50', value: 50 },
        { text: '100', value: 100 },
      ].filter((v, i) => i === 0 || v.value !== this.options.perPage).sort((a, b) => {
        if (a.value === 0) return 1
        if (b.value === 0) return -1
        return a.value - b.value
      })
    },

    getPagination () {
      const { page = 1, count = 0 } = this.pagination
      const perPage = this.recordsPerPage

      return {
        from: ((page - 1) * perPage) + 1,
        to: perPage > 0 ? Math.min((page * perPage), count) : count,
        page,
        perPage,
        count,
      }
    },

    hasPrevPage () {
      return this.filter.prevPage
    },

    hasNextPage () {
      return this.filter.nextPage
    },

    showPageNavigation () {
      return !this.options.hidePaging
    },

    disableSelectAll () {
      return this.items.length === 0
    },

    areAllRowsSelected () {
      return this.selected.length === this.items.length
    },

    recordListModule () {
      if (this.options.moduleID) {
        return this.getModuleByID(this.options.moduleID)
      } else {
        return undefined
      }
    },

    allFields () {
      return [
        ...this.recordListModule.fields,
        ...this.recordListModule.systemFields(),
      ]
    },

    fields () {
      let fields = []

      if (!this.options.hideConfigureFieldsButton && this.customConfiguredFields.length > 0) {
        fields = this.recordListModule.filterFields(this.customConfiguredFields)
      } else if (this.options.fields.length > 0) {
        fields = this.recordListModule.filterFields(this.options.fields)
      } else {
        fields = [...this.recordListModule.fields.slice(0, 5), ...this.recordListModule.systemFields()]
      }

      return fields.map(mf => ({
        key: mf.name,
        label: mf.isSystem ? this.$t(`field:system.${mf.name}`) : mf.label || mf.name,
        moduleField: mf,
        sortable: !this.options.hideSorting && !mf.isMulti && mf.isSortable,
        tdClass: 'record-value',
        required: mf.isRequired,
      }))
    },
  },

  watch: {
    query: {
      handler () {
        this.refresh(true)
      },
    },

    options: {
      deep: true,
      handler () {
        if (!this.loadingRecord) {
          this.prepRecordList()
          this.refresh(true)
        }
      },
    },

    'record.recordID': {
      immediate: true,
      handler () {
        this.createEvents()
        this.prepRecordList()
        this.refresh(true)
      },
    },
  },

  beforeDestroy () {
    this.abortRequests()
    this.destroyEvents()
    this.setDefaultValues()
  },

  created () {
    this.refreshBlock(this.refresh, false, true)
  },

  methods: {
    ...mapActions({
      updateRecordSet: 'record/updateRecords',
    }),

    createEvents () {
      const { pageID = '0' } = this.page
      const { recordID = '0' } = this.record || {}

      this.uniqueID = [pageID, recordID, this.block.blockID, this.magnified].map(v => v || '0').join('-')
    },

    prepRecordList () {
      const { moduleID, presort, prefilter, perPage } = this.options

      if (!moduleID || !this.recordListModule) {
        throw Error(this.$t('record.moduleOrPageNotSet'))
      }

      this.recordsPerPage = perPage

      const sort = presort || 'createdAt DESC'
      const filter = []

      if (prefilter) {
        // Process prefilter logic here
        filter.push(`(${prefilter})`)
      }

      this.prefilter = filter.join(' AND ')

      this.filter = {
        limit: this.recordsPerPage,
        sort,
      }
    },

    wrapRecord (r, id) {
      if (r.id) {
        id = r.id
        r = r.r
      }

      return {
        r,
        id: id || (r.recordID !== '0' ? r.recordID : `${this.uniqueID}:${this.ctr++}`),
      }
    },

    onSelectRow (selected, item) {
      if (selected) {
        if (this.selected.includes(item.id)) {
          return
        }
        this.selected.push(item.id)
      } else {
        const i = this.selected.indexOf(item.id)
        if (i < 0) {
          return
        }
        this.selected.splice(i, 1)
        this.selectedAllRecords = false
      }
    },

    isSortedBy ({ key }, dir) {
      const { sort = '' } = this.filter
      const sortedFields = (sort.includes(',') ? sort.split(',') : [sort])

      return sortedFields.map(v => v.trim()).some(value => {
        let valueDir = 'ASC'

        if (value.includes(' ')) {
          value = value.split(' ')[0]
          valueDir = 'DESC'
        }

        return valueDir === dir && value === key
      })
    },

    handleSort ({ key, sortable }) {
      if (!sortable) {
        return
      }

      if (this.sortBy !== key) {
        this.filter.sort = `${key}`
        this.sortDirecton = 'ASC'
      } else {
        if (this.sortDirecton === 'ASC') {
          this.filter.sort = `${key} DESC`
          this.sortDirecton = 'DESC'
        } else {
          this.filter.sort = `${key}`
          this.sortDirecton = 'ASC'
        }
      }
      this.sortBy = key
      this.refresh(true)
    },

    goToPage (page) {
      if (page >= 1) {
        this.filter.pageCursor = (this.pagination.pages[page - 1] || {}).cursor
        this.pagination.page = page
      } else {
        this.filter.pageCursor = this.filter[page]
        if (this.filter.pageCursor) {
          this.pagination.page += page === 'nextPage' ? 1 : -1
        } else {
          this.pagination.page = 1
        }
      }
      this.refresh()
    },

    handleSelectAllOnPage ({ isChecked }) {
      if (isChecked) {
        this.selected = this.items.map(({ id }) => id)
      } else {
        this.selected = []
        this.selectedAllRecords = isChecked
      }
    },

    handlePerPageChange () {
      this.filter.limit = this.recordsPerPage
      this.refresh(true)
    },

    onUpdateFields (fields = []) {
      this.options.fields = [...fields]
      this.customConfiguredFields = fields.map((f) => f.isSystem ? f.name : f.fieldID).filter(f => !!f)
      this.$emit('save-fields', this.options.fields)
    },

    onExport (e) {
      this.processing = true
      // Export logic here
      this.processing = false
    },

    async refresh (resetPagination = false) {
      await this.$nextTick()
      return this.pullRecords(resetPagination)
    },

    async pullRecords (resetPagination = false) {
      if (!this.recordListModule) {
        return
      }

      this.processing = true
      this.selected = []

      const query = this.prefilter
      const { moduleID, namespaceID } = this.recordListModule

      let paginationOptions = {}

      if (resetPagination) {
        this.filter.pageCursor = undefined
        const { fullPageNavigation = false, showTotalCount = false } = this.options
        paginationOptions = {
          incPageNavigation: fullPageNavigation,
          incTotal: showTotalCount,
        }
      } else if (this.filter.pageCursor) {
        this.filter.sort = ''
      }

      const { response, cancel } = this.$ComposeAPI.recordListCancellable({
        ...this.filter,
        moduleID,
        namespaceID,
        query,
        ...paginationOptions,
      })

      this.abortableRequests.push(cancel)

      return response().then(({ set, filter }) => {
        const records = set.map(r => new compose.Record(r, this.recordListModule))

        this.updateRecordSet(records)

        this.filter = { ...this.filter, ...filter }
        this.filter.nextPage = filter.nextPage
        this.filter.prevPage = filter.prevPage

        if (resetPagination) {
          let count = this.pagination.count || 0

          if (paginationOptions.incTotal) {
            count = filter.total || 0
          }

          if (paginationOptions.incPageNavigation) {
            const pages = filter.pageNavigation || []
            this.pagination.pages = pages

            if (!paginationOptions.incTotal) {
              if (pages.length > 1) {
                const lastPageCount = pages[pages.length - 1].items
                count = ((pages.length - 1) * this.recordsPerPage) + lastPageCount
              } else {
                count = records.length
              }
            }
          }

          this.pagination.count = count
          this.pagination.page = 1
        }

        const fields = this.fields.filter(f => f.moduleField).map(f => f.moduleField)

        return Promise.all([
          this.fetchUsers(fields, records),
          this.fetchRecords(namespaceID, fields, records),
        ]).then(() => {
          this.items = records.map(r => this.wrapRecord(r))
        })
      }).catch((e) => {
        if (!axios.isCancel(e)) {
          this.toastErrorHandler(this.$t('notification:record.listLoadFailed'))(e)
        }
      }).finally(() => {
        setTimeout(() => {
          this.processing = false
        }, 300)
      })
    },

    setDefaultValues () {
      this.uniqueID = undefined
      this.processing = false
      this.prefilter = undefined
      this.query = null
      this.filter = {}
      this.pagination = {}
      this.selected = []
      this.sortBy = undefined
      this.sortDirecton = undefined
      this.ctr = 0
      this.items = []
      this.selectedAllRecords = false
      this.abortableRequests = []
      this.customConfiguredFields = []
    },

    abortRequests () {
      this.abortableRequests.forEach((cancel) => {
        cancel()
      })
    },

    destroyEvents () {
      // Destroy events logic here
    },
  },
}
</script>

<style lang="scss" scoped>
.record-crud-table {
  .actions {
    padding-top: 8px;
    width: 1%;
    font-family: var(--font-regular) !important;
  }
}

.record-crud-footer {
  font-family: var(--font-medium);
}
</style>
