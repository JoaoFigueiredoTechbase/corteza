<template>
  <b-container
    fluid
    class="h-100 mh-100 p-0"
  >
    <split
      direction="horizontal"
      :gutter-size="12"
      class="h-100 overflow-hidden"
    >
      <split-area
        :size="map.show ? 70 : 100"
        :min-size="300"
        class="overflow-hidden"
      >
        <div class="px-3">
          <b-form-group class="mb-0">
            <c-input-search
              :value="query"
              :placeholder="$t('input-placeholder')"
              :autocomplete="'off'"
              :disabled="storeProcessing"
              submittable
              @search="onQuerySubmit"
            />
          </b-form-group>

          <div
            class="d-flex align-items-center px-1 mt-1 mb-2 text-muted"
          >
            <div class="d-flex align-items-center">
              <span
                :class="{ 'discovering': storeProcessing }"
                class="mt-1"
              >
                {{ searchDescription }}
              </span>
            </div>

            <div class="d-flex align-items-center ml-auto">
              <font-awesome-icon
                :icon="['fas', 'grip-lines']"
                class="mt-2 mr-1 pointer"
                :class="{ 'text-primary': viewMode === 'list' }"
                @click="viewMode = 'list'"
              />

              <b-form-checkbox
                v-model="viewMode"
                :value="'grid'"
                :unchecked-value="'list'"
                switch
                class="pointer ml-2"
              />

              <font-awesome-icon
                :icon="['fas', 'grip-horizontal']"
                class="mt-2 ml-1 pointer"
                :class="{ 'text-primary': viewMode === 'grid' }"
                @click="viewMode = 'grid'"
              />
            </div>
          </div>
        </div>

        <div
          class="d-flex flex-wrap gap-3 p-3 w-100 h-100 m-0 mh-100 overflow-auto position-relative"
          :class="{ 'list-view': viewMode === 'list' }"
        >
          <div
            v-if="storeProcessing || !total.actual"
            class="position-absolute d-flex align-items-center justify-content-center w-100 h-50"
            style="opacity: 0.8; z-index: 1; background-color: var(--light);"
          >
            <h5 class="mb-0">
              <b-spinner
                v-if="storeProcessing"
                variant="primary"
                class="p-4"
              />
              <span
                v-else-if="!total.actual"
              >
                {{ $t('no-results') }}
              </span>
            </h5>
          </div>

          <div
            v-for="(hit, i) in filteredHits"
            :key="i"
            class="result-item w-100"
            :class="{ 'grid-view': viewMode === 'grid' }"
          >
            <result
              :id="hit.value.recordID || hit.value.moduleID"
              :index="i"
              :hit="hit"
              :show-map="map.show"
              :class="{ 'border-primary border shadow': map.clickedMarker && [hit.value.recordID, hit.value.moduleID].includes(map.clickedMarker) }"
              @hover="map.hoverIndex = $event"
            />
          </div>
        </div>

        <div
          class="position-fixed map-button"
        >
          <b-button
            v-b-tooltip.noninteractive.hover="{ title: $t('tooltip.map'), container: '#body' }"
            variant="warning"
            class="rounded-circle p-3"
            @click="toggleMap"
          >
            <font-awesome-icon
              :icon="['fas', 'map-marked-alt']"
              class="h5 mb-0"
            />
          </b-button>
        </div>
      </split-area>

      <split-area
        :size="map.show ? 30 : 0"
        :min-size="300"
      >
        <discovery-map
          v-if="map.show"
          :markers="map.markers"
          :hover-index="map.hoverIndex"
          class="pl-3"
          @hover="markerHovered"
        />
      </split-area>
    </split>
  </b-container>
</template>

<script>
import { mapGetters, mapActions } from 'vuex'
import Result from './Results'
import DiscoveryMap from './DiscoveryMap.vue'
import { components } from '@cortezaproject/corteza-vue'
import { Split, SplitArea } from 'vue-split-panel'
const { CInputSearch } = components

export default {
  i18nOptions: {
    namespaces: 'search',
  },

  components: {
    Result,
    DiscoveryMap,
    CInputSearch,
    Split,
    SplitArea,
  },

  data () {
    return {
      query: '',

      hits: [],
      filteredHits: [],

      pagination: {
        limit: 50,
        from: 0,
        size: 50,
      },

      total: {
        all: 0,
        actual: 0,
      },

      initial: false,

      map: {
        show: false,
        markers: [],
        clickedMarker: undefined,
        hoverIndex: undefined,
      },

      viewMode: 'list',
    }
  },

  computed: {
    ...mapGetters({
      storeProcessing: 'discovery/processing',
      storeTypes: 'discovery/types',
      storeModules: 'discovery/modules',
      storeNamespaces: 'discovery/namespaces',
    }),

    searchDescription () {
      if (this.storeProcessing) {
        return this.$t('discovering')
      }

      if (this.total.all > 0) {
        return this.$t('range', { actual: this.total.actual, all: this.total.all })
      }

      return ''
    },
  },

  watch: {
    storeTypes: {
      handler () {
        this.getFilteredData()
      },
    },

    storeModules: {
      handler () {
        if (this.initial) return
        this.pagination.size = this.pagination.limit
        this.getSearchData()
      },
    },

    storeNamespaces: {
      handler () {
        if (this.initial) return
        this.pagination.size = this.pagination.limit
        this.getSearchData()
      },
    },
  },

  created () {
    this.initial = true

    const { query = '', modules = [], namespaces = [], size = 50 } = this.$route.query

    this.query = query
    this.pagination.size = size

    this.updateModules(Array.isArray(modules) ? modules : [modules])
    this.updateNamespaces(Array.isArray(namespaces) ? namespaces : [namespaces])

    this.getSearchData()

    setTimeout(() => {
      this.initial = false
    }, 1000)
  },

  mounted () {
    const listElm = document.querySelector('.results')
    listElm.addEventListener('scroll', e => {
      if (listElm.scrollTop + listElm.clientHeight >= listElm.scrollHeight - 10) {
        if (!this.storeProcessing && this.total.actual < this.total.all) {
          this.getSearchData({ append: true })
        }
      }
    })
  },

  methods: {
    ...mapActions({
      fetchData: 'discovery/fetchData',
      updateModules: 'discovery/updateModules',
      updateNamespaces: 'discovery/updateNamespaces',
    }),

    getSearchData ({ query = this.query, append = false } = {}) {
      if (append) {
        this.pagination.size += this.pagination.limit
      } else {
        this.map.markers = []
        this.hits = []
        this.filteredHits = []
      }

      const modules = this.storeModules
      const namespaces = this.storeNamespaces

      const { size } = this.pagination

      this.updateRouteQuery({ query, modules, namespaces, size })

      this.fetchData({ query, modules, namespaces, size }).then((response = {}) => {
        if (response) {
          this.hits = (response.hits || [])

          this.total.all = response.total_results || 0

          this.getFilteredData()

          this.pagination = {
            ...this.pagination,
            from: response.from || 0,
            size: response.size || 0,
          }

          this.getMarkers()
        }
      }).catch(e => {
        this.toastErrorHandler(this.$t('notification:search.failed'))(e)
        this.hits = []
        this.filteredHits = []
      })
    },

    getFilteredData () {
      let filteredHits = this.hits

      if (this.storeTypes.length > 0 && this.hits.length) {
        filteredHits = this.hits.filter(hit => this.storeTypes.includes(hit.type))
      } else {
        filteredHits = []
      }

      this.filteredHits.splice(0, this.filteredHits.length, ...filteredHits)
      this.total.actual = this.filteredHits.length
    },

    onQuerySubmit (query) {
      if (!this.storeProcessing) {
        this.query = query
        this.pagination.size = 50
        this.getSearchData()
      }
    },

    getMarkers () {
      const markers = []

      this.filteredHits.forEach(({ type, value }) => {
        if (type === 'compose:record' && Array.isArray(value.values)) {
          const id = value.recordID
          value.values.forEach(({ value = [] }) => {
            const isGeometry = value && value.find(v => {
              return v.toString().includes('{"coordinates":[')
            })

            if (isGeometry) {
              value.forEach(coordinates => {
                coordinates = JSON.parse(coordinates || '{}').coordinates
                if (coordinates && coordinates.length) {
                  markers.push({ id, coordinates })
                }
              })
            }
          })
        }
      })

      this.map.markers = markers
    },

    markerHovered (ID) {
      if (ID) {
        document.getElementById(ID).scrollIntoView({
          behavior: 'smooth',
          block: 'center',
        })
      }

      this.map.clickedMarker = ID
    },

    toggleMap () {
      this.map.show = !this.map.show
    },

    updateRouteQuery ({ query = undefined, modules = [], namespaces = [], size = 0 }) {
      if (JSON.stringify(this.$route.query) !== JSON.stringify({ query, modules, namespaces })) {
        this.$router.push({ query: { query: query || undefined, modules, namespaces, size } })
      }
    },
  },
}
</script>

<style lang="scss">
.split .gutter {
  background-color: transparent;
}
</style>

<style lang="scss" scoped>
.map-button {
  bottom: 1rem;
  right: 1rem;
  z-index: 99999;
}

// https://stackoverflow.com/a/40991531/17926309
.discovering::after {
  display: inline-block;
  animation: discovering steps(1, end) 1s infinite;
  content: '';
}

@keyframes discovering {
  0% { content: ''; }
  25% { content: '.'; }
  50% { content: '..'; }
  75% { content: '...'; }
  100% { content: ''; }
}

.result-item {
  &.grid-view {
    min-width: 30rem;
    flex: 1;
  }
}

.list-view {
  .result-item {
    max-width: 100%;
  }
}
</style>
