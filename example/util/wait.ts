import "isomorphic-unfetch"
import { getConfig } from "../util/config"

const config = getConfig()


export function wait(seconds: number) {
  return new Promise(resolve => {
    console.log("Waiting", seconds, "s")

    setTimeout(resolve, seconds * 1000)
  })
}

export function waitTransaction(txHash: string, apiKey: string) {
  let counter = 4

  return new Promise((resolve, reject) => {
    const itv = setInterval(() => {
      getTransactionMiningStatus(txHash, apiKey)
        .then(mined => {
          if (mined) {
            clearInterval(itv)
            return resolve(null)
          }
          else if (counter <= 0) {
            clearInterval(itv)
            return reject(new Error("The transaction hasn't been mined after a while"))
          }

          counter--
        })
        .catch(err => {
          if (counter <= 0) {
            clearInterval(itv)
            return reject(new Error("The transaction status cannot be checked"))
          }

          counter--
        })
    })
  })
}

export async function getTransactionMiningStatus(txHash: string, apiKey: string): Promise<boolean> {
  const url = config.apiUrlPrefix + "/v1/priv/transactions/" + txHash

  const response = await fetch(url, {
    headers: {
      "Authorization": "Bearer " + apiKey,
    },
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()
  console.log("GET", url, responseBody)

  const { error } = responseBody
  if (error) throw new Error(error)

  const { mined } = responseBody

  console.log("Read transaction", txHash, ":", responseBody)
  return mined
}
