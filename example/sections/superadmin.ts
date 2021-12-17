import "isomorphic-unfetch"
import { getConfig } from "../util/config"

const config = getConfig()

export async function createIntegrator(name: string): Promise<{ id: string, apiKey: string }> {
  const url = config.apiUrlPrefix + "/v1/admin/accounts"

  const body = {
    name,
    email: Math.random().toString().slice(2) + "@email.net",
    cspUrlPrefix: config.cspUrlPrefix,
    cspPubKey: config.cspPublicKey
  }

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Authorization": "Bearer " + config.superadminKey,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(body),
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()
  
  const { error } = responseBody
  if (error) throw new Error(error)
  
  const { id, apiKey } = responseBody

  console.log("Created integrator with ID", id, "and API key", apiKey)
  return { id: id.toString(), apiKey }
}

export async function deleteIntegrator(id: string): Promise<void> {
  const url = config.apiUrlPrefix + "/v1/admin/accounts/" + id

  const response = await fetch(url, {
    method: "DELETE",
    headers: {
      "Authorization": "Bearer " + config.superadminKey
    },
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()
  const { error } = responseBody

  if (error) throw new Error(error)

  console.log("Deleted integrator account", id)
}
