import { PageBlock, PageBlockInput, Registry } from './base'
import { Apply } from '../../../cast'
import { Compose as ComposeAPI, System as SystemAPI } from '../../../api-clients'
import { Record, Module } from '../../'
import { User } from '../../../system'
import { convertRevisionPayloadToRevision, RawRevisionPayload, Revision } from '../revision'

const kind = 'RecordRevisions'
interface Options {
  // do we preload changes or not
  preload: boolean;

  // what fields do we want to display
  // empty array means all fields
  displayedFields: string[];

  // referenced fields (records, users) we want to expand
  expRefFields: string[];
  refreshRate: number;
  showRefresh: boolean;
  magnifyOption: string;
}

const defaults: Readonly<Options> = Object.freeze({
  preload: false,
  displayedFields: [],
  expRefFields: [],
  refreshRate: 0,
  showRefresh: false,
  magnifyOption: '',
})

export class PageBlockRecordRevisions extends PageBlock {
  readonly kind = kind

  options: Options = { ...defaults }

  constructor (i?: PageBlockInput) {
    super(i)
    this.applyOptions(i?.options as Partial<Options>)
  }

  applyOptions (o?: Partial<Options>): void {
    if (!o) return

    Apply(this.options, o, Boolean, 'preload', 'showRefresh')
    Apply(this.options, o, Number, 'refreshRate')
    Apply(this.options, o, String, 'magnifyOption')

    // set new values to displayed fields
    if (Array.isArray(o?.displayedFields)) {
      this.options.displayedFields = o.displayedFields.map(String)
    }

    // set new values to expanded reference fields
    if (Array.isArray(o?.expRefFields)) {
      this.options.expRefFields = o.expRefFields.map(String)
    }
  }

  /**
   * fetch is a utility method on record revision page block
   * that fetches revisions for a record and converts them to RevisionPayload class
   *
   * this function also strips out all fields that should not be dispalyed
   * (as per displayedFields option)
   *
   * @param api Compose API to be used
   * @param record Record to fetch revisions for
   */
  async fetch (api: ComposeAPI, record: Record): Promise<Array<Revision>> {
    const { namespaceID, moduleID, recordID } = record

    return api
      .recordRevisions({ namespaceID, moduleID, recordID })
      .then(payload => convertRevisionPayloadToRevision(
        (payload as unknown) as RawRevisionPayload,
        this.options.displayedFields,
      ))
  }
}

Registry.set(kind, PageBlockRecordRevisions)
