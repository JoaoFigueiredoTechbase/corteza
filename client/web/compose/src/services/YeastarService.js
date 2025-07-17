export default class YeastarService {
  async getCDR () {
    const apiUrl = 'https://172.26.0.6:8088/'
    const apiUserName = 'eOoVHNLBl0ytb6sM19HVHVDKKwDNoxsS'
    const apiSecret = 'YyclbdWjDcmNBPvviNMG2eeuB3oZAqnj'

    // get token
    const tokenUrl = `${apiUrl}/openapi/v1.0/get_token`

    const response = await fetch(tokenUrl, {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
        'User-Agent': 'OpenAPI',
      },
      body: JSON.stringify({
        username: apiUserName,
        password: apiSecret,
      }),
    })

    const data = await response.json()

    if (data.errcode !== 0) {
      throw new Error('Failed to retrieve token')
    }

    const tokenData = data.access_token
    const cdrUrl = `${apiUrl}/openapi/v1.0/cdr/list?access_token=${tokenData}`

    const responseCdr = await fetch(cdrUrl, {
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'User-Agent': 'OpenAPI',
      },
    })

    if (!responseCdr.ok) {
      if (responseCdr.status === 404) {
        throw new Error('Endpoint not found: cdr')
      }

      throw new Error(`API error ${responseCdr.status}: ${await responseCdr.text()}`)
    }

    console.log('CDR response:', responseCdr)

    return await responseCdr.json()
  }

  async fetchCDR () {
    const baseUrl = window.CortezaAPI = 'http://localhost:80/api'

    const response = await fetch(`${baseUrl}/fetch-cdr`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (!response.ok) {
      throw new Error(`Failed to fetch CDR: ${response.statusText}`)
    }

    return await response.json()
  }

  async getCDRDB () {
    const baseUrl = window.CortezaAPI = 'http://localhost:80/api'

    try {
      const response = await fetch(`${baseUrl}/db-cdr`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      })

      if (!response.ok) {
        throw new Error(`Failed to fetch CDR: ${response.statusText}`)
      }

      // First check the response text
      const text = await response.text()
      console.log('Raw response:', text) // Inspect this in console

      const data = JSON.parse(text)
      return data
    } catch (error) {
      console.error('Error in getCDRDB:', error)
      throw error // Re-throw to handle in calling function
    }
  }

  async syncALL () {
    const baseUrl = window.CortezaAPI = 'http://localhost:80/api'

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
      throw error // Re-throw to handle in calling function
    }
  }
}
