<script>
export default {
  props: {
    index: {
      type: Number,
      required: true,
    },
    hit: {
      type: Object,
      required: true,
    },
    showMap: {
      type: Boolean,
      required: true,
    },
  },

  data () {
    return {
      defaultBlacklistedFields: ['deleted', 'created', 'updated', 'security'],
    }
  },

  computed: {
    blacklistedFields () {
      return this.defaultBlacklistedFields
    },

    query () {
      return [this.$route.query.query || '']
    },

    createdBy () {
      const { by } = this.hit.value.created || {}
      return by
    },

    createdAt () {
      const { at } = this.hit.value.created || {}
      return at ? new Date(at).toLocaleString() : at
    },

    updatedAt () {
      const { at } = this.hit.value.updated || {}
      return at ? new Date(at).toLocaleString() : at
    },
  },

  methods: {
    limitData () {
      const out = {}
      const limit = 5

      if (this.hit.value) {
        let counter = 0

        for (const key in this.hit.value) {
          const value = this.hit.value[key]

          if (counter < limit && !!value && this.blacklistedFields.indexOf(key) < 0) {
            out[key] = value
            counter++
          }
        }
      }

      if (this.createdBy) {
        out.createdBy = this.createdBy
      }

      if (this.createdAt) {
        out.createdAt = this.createdAt
      }

      if (this.updatedAt) {
        out.updatedAt = this.updatedAt
      }

      return out
    },
  },
}
</script>
