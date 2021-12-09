import * as assert from "assert"
import { readFileSync, existsSync } from "fs"
import * as YAML from 'yaml'

const CONFIG_PATH = "./config.yaml"

export function getConfig(): Config {
  if (!existsSync(CONFIG_PATH)) throw new Error("Please, copy config-template.yaml into config.yaml and provide your own settings")

  const config: Config = YAML.parse(readFileSync(CONFIG_PATH).toString())
  assert(typeof config == "object", "The config file appears to be invalid")

  assert(typeof config.superadminKey == "string", "config.yaml > superadminKey should be a string")
  assert(typeof config.apiUrlPrefix == "string", "config.yaml > apiUrlPrefix should be a string")
  assert(typeof config.cspUrlPrefix == "string", "config.yaml > cspUrlPrefix should be a string")
  assert(typeof config.cspPublicKey == "string", "config.yaml > cspPublicKey should be a string")

  return config
}

type Config = {
  superadminKey: string
  apiUrlPrefix: string
  cspUrlPrefix: string
  cspPublicKey: string
}
