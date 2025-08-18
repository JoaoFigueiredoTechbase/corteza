<template>
  <div>
    <b-tab
      data-test-id="record-crud-table-configurator"
      :title="$t('recordCrudTable.label')"
      no-body
    >
      <div class="px-3 pt-3">
        <h5 class="mb-3">
          {{ $t('recordCrudTable.record.generalLabel') }}
        </h5>

        <b-row>
          <b-col cols="12" lg="6">
            <b-form-group
              :label="$t('general.module')"
              variant="primary"
              label-class="text-primary"
            >
              <c-input-select
                v-model="options.moduleID"
                :options="modules"
                label="name"
                :reduce="o => o.moduleID"
                :placeholder="$t('recordList.modulePlaceholder')"
                default-value="0"
                required
              />
            </b-form-group>
          </b-col>
        </b-row>
      </div>

      <template v-if="recordListModule">
        <hr>

        <div class="px-3">
          <div class="mb-3">
            <h5 class="d-flex align-items-center mb-1">
              {{ $t('module:general.fields') }}
            </h5>
            <small class="text-muted">
              {{ $t('recordList.moduleFieldsFootnote') }}
            </small>
          </div>

          <b-row>
            <b-col cols="12">
              <field-picker
                :module="recordListModule"
                :fields.sync="options.fields"
                class="mb-3"
                style="height: 50vh;"
              />
            </b-col>

            <b-col cols="12" lg="6">
              <b-form-group
                :label="$t('recordList.hideConfigureFieldsButton')"
                label-class="text-primary"
              >
                <c-input-checkbox
                  v-model="options.hideConfigureFieldsButton"
                  switch
                  invert
                  :labels="checkboxLabel"
                />
              </b-form-group>
            </b-col>

            <b-col cols="12" md="6">
              <b-form-group
                :label="$t('recordList.record.textStyles')"
                label-class="text-primary"
              >
                <column-picker
                  size="sm"
                  variant="light"
                  :module="recordListModule"
                  :fields="options.textStyles.wrappedFields || []"
                  :field-subset="options.fields.length ? options.fields : recordListModule.fields"
                  @updateFields="onUpdateTextWrapOption"
                >
                  {{ $t('recordList.record.configureWrappedFields') }}
                </column-picker>
              </b-form-group>
            </b-col>
          </b-row>
        </div>

        <hr>

        <div class="px-3">
          <h5 class="mb-3">
            {{ $t('recordCrudTable.rowButtons.label') }}
          </h5>

          <p class="text-muted">
            {{ $t('recordCrudTable.rowButtons.description') }}
          </p>

          <c-form-table-wrapper
            :labels="{
              addButton: $t('general:label.add')
            }"
            @add-item="addRowButton"
          >
            <b-table-simple
              borderless
              small
              responsive
              class="mb-2"
            >
              <draggable
                :list.sync="options.rowButtons"
                group="sort"
                handle=".grab"
                tag="tbody"
              >
                <b-tr
                  v-for="(button, index) in options.rowButtons"
                  :key="index"
                >
                  <b-td
                    class="grab text-center align-middle"
                    style="width: 40px;"
                  >
                    <font-awesome-icon
                      :icon="['fas', 'bars']"
                      class="text-secondary"
                    />
                  </b-td>

                  <b-td style="min-width: 150px;">
                    <b-form-input
                      v-model="button.label"
                      :placeholder="$t('recordCrudTable.rowButtons.labelPlaceholder')"
                    />
                  </b-td>

                  <b-td style="min-width: 200px;">
                    <c-input-select
                      v-model="button.workflowID"
                      :options="workflows"
                      :placeholder="$t('recordCrudTable.rowButtons.workflowPlaceholder')"
                      :reduce="wf => wf.workflowID"
                      label="handle"
                    />
                  </b-td>

                  <b-td style="min-width: 150px;">
                    <b-form-select
                      v-model="button.variant"
                      :options="buttonVariants"
                    />
                  </b-td>

                  <b-td style="min-width: 150px;">
                    <b-form-select
                      v-model="button.size"
                      :options="buttonSizes"
                    />
                  </b-td>

                  <b-td class="text-right align-middle">
                    <c-input-confirm
                      show-icon
                      @confirmed="options.rowButtons.splice(index, 1)"
                    />
                  </b-td>
                </b-tr>
              </draggable>
            </b-table-simple>
          </c-form-table-wrapper>
        </div>

        <hr>

        <div class="px-3">
          <h5 class="mb-3">
            {{ $t('recordList.record.prefilterLabel') }}
          </h5>

          <b-row>
            <b-col cols="12" lg="6">
              <b-form-group
                :label="$t('recordList.record.filterHide')"
                label-class="text-primary"
              >
                <c-input-checkbox
                  v-model="options.hideFiltering"
                  switch
                  invert
                  :labels="checkboxLabel"
                />
              </b-form-group>
            </b-col>

            <b-col cols="12" lg="6">
              <b-form-group
                :label="$t('recordList.record.prefilterHideSearch')"
                label-class="text-primary"
              >
                <c-input-checkbox
                  v-model="options.hideSearch"
                  switch
                  invert
                  :labels="checkboxLabel"
                />
              </b-form-group>
            </b-col>
          </b-row>

          <prefilter
            :record="record"
            :module="recordListModule"
            :namespace="namespace"
            :options="options"
            :page="page"
          />

          <hr>

          <b-row>
            <b-col>
              <b-form-group
                :label="$t('recordList.record.setCustomFilterPresets')"
                label-class="text-primary"
              >
                <c-input-checkbox
                  v-model="options.customFilterPresets"
                  switch
                  :labels="checkboxLabel"
                />
              </b-form-group>
            </b-col>
          </b-row>
        </div>

        <hr>

        <div class="px-3">
          <h5 class="mb-3">
            {{ $t('recordList.record.presortLabel') }}
          </h5>

          <b-row>
            <b-col>
              <b-form-group
                :label="$t('recordList.record.presortHideSort')"
                label-class="text-primary"
              >
                <c-input-checkbox
                  v-model="options.hideSorting"
                  switch
                  invert
                  :labels="checkboxLabel"
                />
              </b-form-group>
            </b-col>
          </b-row>

          <b-row>
            <b-col>
              <c-input-presort
                v-model="options.presort"
                :fields="recordListModuleFields"
                :labels="{
                  ascending: $t('general:label.ascending'),
                  descending: $t('general:label.descending'),
                  none: $t('general:label.none'),
                  placeholder: $t('recordList.record.presortPlaceholder'),
                  footnote: $t('recordList.record.presortFootnote'),
                  toggleInput: $t('recordList.record.presortToggleInput'),
                  addButton: $t('general:label.add'),
                  title: $t('recordList.record.presortInputLabel')
                }"
                allow-text-input
              />
            </b-col>
          </b-row>
        </div>

        <hr>

        <div class="px-3">
          <h5 class="mb-3">
            {{ $t('recordList.record.pagingLabel') }}
          </h5>

          <b-row>
            <b-col cols="12" lg="6">
              <b-form-group
                :label="$t('recordList.record.hidePaging')"
                label-class="text-primary"
              >
                <c-input-checkbox
                  v-model="options.hidePaging"
                  switch
                  invert
                  :labels="checkboxLabel"
                />
              </b-form-group>
            </b-col>

            <b-col cols="12" lg="6">
              <b-form-group
                :label="$t('recordList.record.fullPageNavigation')"
                label-class="text-primary"
              >
                <c-input-checkbox
                  v-model="options.fullPageNavigation"
                  switch
                  :labels="checkboxLabel"
                />
              </b-form-group>
            </b-col>

            <b-col cols="12" lg="6">
              <b-form-group
                :label="$t('recordList.record.perPage')"
                label-class="text-primary"
              >
                <b-form-input
                  v-model.number="options.perPage"
                  type="number"
                  class="mb-2"
                />
              </b-form-group>
            </b-col>

            <b-col cols="12" lg="6">
              <b-form-group
                :label="$t('recordList.record.showRecordPerPageOption')"
                label-class="text-primary"
              >
                <c-input-checkbox
                  v-model="options.showRecordPerPageOption"
                  switch
                  :labels="checkboxLabel"
                />
              </b-form-group>
            </b-col>

            <b-col cols="12" lg="6">
              <b-form-group
                :label="$t('recordList.record.showTotalCount')"
                label-class="text-primary"
              >
                <c-input-checkbox
                  v-model="options.showTotalCount"
                  switch
                  :labels="checkboxLabel"
                />
              </b-form-group>
            </b-col>
          </b-row>
        </div>

        <hr>

        <div class="px-3">
          <h5 class="mb-3">
            {{ $t('recordCrudTable.record.otherOptions') }}
          </h5>

          <b-row>
            <b-col cols="12" lg="6">
              <b-form-group
                :label="$t('recordList.selectable')"
                label-class="text-primary"
              >
                <c-input-checkbox
                  v-model="options.selectable"
                  switch
                  :labels="checkboxLabel"
                />
              </b-form-group>
            </b-col>

            <b-col cols="12" lg="6">
              <b-form-group
                :label="$t('recordList.export.allow')"
                label-class="text-primary"
              >
                <c-input-checkbox
                  v-model="options.allowExport"
                  switch
                  :labels="checkboxLabel"
                />
              </b-form-group>
            </b-col>
          </b-row>
        </div>
      </template>
    </b-tab>
  </div>
</template>

<script>
import { mapGetters } from 'vuex'
import { NoID } from '@cortezaproject/corteza-js'
import Draggable from 'vuedraggable'
import base from './base'
import FieldPicker from 'corteza-webapp-compose/src/components/Common/FieldPicker'
import { components } from '@cortezaproject/corteza-vue'
import Prefilter from './RecordList/Prefilter.vue'
import ColumnPicker from 'corteza-webapp-compose/src/components/Admin/Module/Records/ColumnPicker'

const { CInputPresort } = components

export default {
  i18nOptions: {
    namespaces: 'block',
  },

  name: 'RecordCrudTable',

  components: {
    FieldPicker,
    CInputPresort,
    Draggable,
    ColumnPicker,
    Prefilter,
  },

  extends: base,

  data () {
    return {
      checkboxLabel: {
        on: this.$t('general:label.yes'),
        off: this.$t('general:label.no'),
      },

      buttonVariants: [
        { value: 'primary', text: this.$t('general:label.primary') },
        { value: 'secondary', text: this.$t('general:label.secondary') },
        { value: 'success', text: this.$t('general:label.success') },
        { value: 'danger', text: this.$t('general:label.danger') },
        { value: 'warning', text: this.$t('general:label.warning') },
        { value: 'info', text: this.$t('general:label.info') },
        { value: 'light', text: this.$t('general:label.light') },
        { value: 'dark', text: this.$t('general:label.dark') },
        { value: 'outline-primary', text: this.$t('general:label.outlinePrimary') },
        { value: 'outline-secondary', text: this.$t('general:label.outlineSecondary') },
      ],

      buttonSizes: [
        { value: 'sm', text: this.$t('general:label.small') },
        { value: 'md', text: this.$t('general:label.medium') },
        { value: 'lg', text: this.$t('general:label.large') },
      ],

      workflows: [],
    }
  },

  computed: {
    ...mapGetters({
      getModuleByID: 'module/getByID',
      modules: 'module/set',
    }),

    recordListModule () {
      if (this.options.moduleID !== NoID) {
        return this.getModuleByID(this.options.moduleID)
      } else {
        return undefined
      }
    },

    recordListModuleFields () {
      if (this.recordListModule) {
        return [
          ...this.recordListModule.fields,
          ...this.recordListModule.systemFields().map(sf => {
            return {
              label: this.$t(`field:system.${sf.name}`),
              name: sf.name === 'recordID' ? 'ID' : sf.name,
            }
          }),
        ].map(({ name, label }) => ({ name, label }))
      }

      return []
    },
  },

  watch: {
    'options.moduleID' (newModuleID) {
      this.options.fields = []
      this.options.rowButtons = []
    },
  },

  mounted () {
    this.fetchWorkflows()
  },

  methods: {
    addRowButton () {
      this.options.rowButtons.push({
        label: '',
        workflowID: '',
        variant: 'primary',
        size: 'sm',
        enabled: true,
      })
    },

    onUpdateTextWrapOption (fields = []) {
      if (this.options.textStyles.wrappedFields) {
        this.options.textStyles.wrappedFields = fields.map(f => f.fieldID && f.fieldID !== NoID ? f.fieldID : f.name).filter(f => !!f)
      }
    },

    async fetchWorkflows () {
      try {
        const { set } = await this.$AutomationAPI.workflowList({
          namespaceID: this.namespace.namespaceID,
          limit: 100,
        })
        this.workflows = set || []
      } catch (error) {
        console.error('Failed to fetch workflows:', error)
        this.workflows = []
      }
    },
  },
}
</script>
