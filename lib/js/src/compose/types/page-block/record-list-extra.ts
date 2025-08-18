import { PageBlock, PageBlockInput, Registry } from './base'
import { Apply, CortezaID, NoID } from '../../../cast'
import { Compose as ComposeAPI } from '../../../api-clients'
import { Module } from '../module'
import { Button } from './types'

const kind = 'RecordListExtra'

export interface Options {
  moduleID: string;
  prefilter: string;
  presort: string;
  fields: unknown[];
  hideHeader: boolean;
  hideSearch: boolean;
  hidePaging: boolean;
  hideSorting: boolean;
  hideFiltering: boolean;
  allowExport: boolean;
  perPage: number;
  
  fullPageNavigation: boolean;
  showTotalCount: boolean;
  customFilterPresets: boolean;
  refreshRate: number;
  showRefresh: boolean;

  // Are table rows selectable
  selectable: boolean;

  // Ordered list of buttons to display in each row
  rowButtons: Array<Button>;

  showRecordPerPageOption: boolean;
  hideConfigureFieldsButton: boolean;

  textStyles: {
    wrappedFields: Array<string>
  }
}

const defaults: Readonly<Options> = Object.freeze({
  moduleID: NoID,
  prefilter: '',
  presort: 'createdAt DESC',
  fields: [],
  hideHeader: false,
  hideSearch: false,
  hidePaging: false,
  hideSorting: false,
  hideFiltering: false,
  allowExport: false,
  perPage: 20,
  
  fullPageNavigation: false,
  showTotalCount: false,
  customFilterPresets: false,
  refreshRate: 0,
  showRefresh: false,

  selectable: false,
  rowButtons: [],
  showRecordPerPageOption: false,
  hideConfigureFieldsButton: true,

  textStyles: {
    wrappedFields: [],
  },
})

export class PageBlockRecordListExtra extends PageBlock {
  readonly kind = kind

  options: Options = { ...defaults }

  constructor (i?: PageBlockInput) {
    super(i)
    this.applyOptions(i?.options as Partial<Options>)
  }

  applyOptions (o?: Partial<Options>): void {
    if (!o) return

    Apply(this.options, o, CortezaID, 'moduleID')

    Apply(this.options, o, String,
      'prefilter',
      'presort',
    )

    Apply(this.options, o, Number, 'perPage', 'refreshRate')

    if (o.fields) {
      this.options.fields = o.fields
    }

    Apply(this.options, o, Boolean,
      'hideHeader',
      'hideSearch',
      'hidePaging',
      'hideFiltering',
      'fullPageNavigation',
      'showTotalCount',
      'customFilterPresets',
      'hideSorting',
      'allowExport',
      'selectable',
      'showRefresh',
      'showRecordPerPageOption',
      'hideConfigureFieldsButton',
    )

    if (o.rowButtons) {
      this.options.rowButtons = o.rowButtons.map(b => new Button(b))
    }

    if (o.textStyles) {
      this.options.textStyles = {
        ...this.options.textStyles,
        ...o.textStyles,
      }
    }
  }

  async fetch (api: ComposeAPI, recordListModule: Module, filter: {[_: string]: unknown}): Promise<object> {
    if (recordListModule.moduleID !== this.options.moduleID) {
      throw Error('Module incompatible, module mismatch')
    }

    filter.moduleID = this.options.moduleID
    filter.namespaceID = recordListModule.namespaceID

    return api
      .recordList(filter)
      .then(r => {
        const { set: records, filter } = r as { filter: object; set: object[] }
        return { records, filter }
      })
  }
}

Registry.set(kind, PageBlockRecordListExtra)