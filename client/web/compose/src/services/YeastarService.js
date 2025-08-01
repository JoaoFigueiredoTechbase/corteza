export default class YeastarService {
  async syncALL () {
    const baseUrl = window.location.protocol + '//' + window.location.host + '/api'

    try {
      const responseCDR = await fetch(`${baseUrl}/sync/all`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      })

      if (!responseCDR.ok) {
        throw new Error(`Failed to sync CDR: ${responseCDR.statusText}`)
      }

      return 200
    } catch (error) {
      console.error('Error in syncCDR:', error)
      throw error
    }
  }
}
