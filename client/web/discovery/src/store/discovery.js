const storeTypes = {
  processing: 'processing',
  types: 'types',
  aggregations: 'aggregations',
  modules: 'modules',
  namespaces: 'namespaces',
}

export default function (DiscoveryAPI) {
  return {
    namespaced: true,

    state: {
      processing: false,
      types: [],
      aggregations: [],
      modules: [],
      namespaces: [],
    },

    getters: {
      processing: state => state.processing,
      types: state => state.types,
      aggregations: state => state.aggregations,
      modules: state => state.modules,
      namespaces: state => state.namespaces,
    },

    actions: {
      async fetchData ({ commit }, { query, modules, namespaces, size }) {
        commit(storeTypes.processing, true)

        return DiscoveryAPI.query({ query, modules, namespaces, size }).then((response = {}) => {
          if (response) {
            commit(storeTypes.aggregations, response.aggregations)
          }

          return response
        }).finally(() => {
          commit(storeTypes.processing, false)
        })
      },

      updateTypes ({ commit }, types) {
        commit(storeTypes.types, types)
      },

      updateModules ({ commit }, modules) {
        commit(storeTypes.modules, modules)
      },

      updateNamespaces ({ commit }, namespaces) {
        commit(storeTypes.namespaces, namespaces)
      },
    },

    mutations: {
      [storeTypes.processing] (state, value) {
        state.processing = value
      },

      [storeTypes.types] (state, types) {
        state.types = types
      },

      [storeTypes.aggregations] (state, aggregations) {
        state.aggregations = aggregations
      },

      [storeTypes.modules] (state, modules) {
        state.modules = modules
      },

      [storeTypes.namespaces] (state, namespaces) {
        state.namespaces = namespaces
      },
    },
  }
}
