import { PageBlock, Registry } from './base'

const kind = 'SyncBox'

interface Options {
  // Add any options you want your block to have
}

const defaults: Readonly<Options> = Object.freeze({})

export class PageBlockSyncBox extends PageBlock {
  readonly kind = kind
  options: Options = { ...defaults }

  constructor(i?: PageBlock | Partial<PageBlock>) {
    super(i)
    this.applyOptions(i?.options as Partial<Options>)
  }

  applyOptions(options?: Partial<Options>) {
    if (options) {
      this.options = { ...this.options, ...options }
    }
  }

  // Optional: custom validation
  validate(): Array<string> {
    return []
  }
}

Registry.set(kind, PageBlockSyncBox)