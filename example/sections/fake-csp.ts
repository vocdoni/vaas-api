import * as NodeRSA from 'node-rsa'

const FAKE_CSP_PRIV_KEY = `-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgHBhoIO6WsbQR6Dr+fyzwdUfrqz4G1s4fKvcQR1NqfvGchXHTZZp
ly7P+1NZnO4UX8z7T9VoMRSoS7lM8jdIeOjoyZuk0WmNHZXGFeDNhoWtX/IZwy7z
/e4qUD+rt1xVU3jjJqkQBSyar1FB+x9tG2qMGPhC4cKjDWyJtRlopwbtAgMBAAEC
gYAHYWH5RLPRerw5hUXVoriIFpySH3ksdHk7kCt2kTMopc+4Pm6KAkU7fc0znB8C
Q7RG8fo8Oat/f835TWRa3ReTnckvHTr6TfuOu3Po7jpRK34QQRZrGLLvnXifsMT2
IWrN5afYmlpd7klRVmgw5yiGDbCubuxhHLXCHPhD9E2OwQJBAL4+C2JLDDDxwatN
wW+IliKQnaAzhyMJkEDdtObMuIIVIbWWSCHDrlb0SeBVrxuZoPWhbyLrRiQP6bPh
bX5haA8CQQCXOedupzmdfOqHHYMqaI8T5XiwXH3+wHtjuSJ6aM1RwqnyN/oE61mW
GFCtNpwiUHdf0izWCngmE3g0YHbTV4VDAkB2hDibV52UsEey7JHhZfoCNo28S92Y
WlDf2D7mugsIHxoNAj6Vqk5mJXIQq9CXJTI9VADkhCYCOVeilIGeBhjJAkEAjlJS
epMq6Aqd9hdSUGEi9oip8uC5Oz4PYiTkS+vB/8aChpEj3elY4Kd1le6lNq4gCrAU
vkQQG1WLdU+rxO7DXQJAZ+nxMhu7KQ43Vtj3Weo1iqiEh64cy4bSsGdPzugVgFSN
AvdMQJigt7jpGS4j7Bz4VQtk530s3HzPqOmyDXSliw==
-----END RSA PRIVATE KEY-----
`
const FAKE_CSP_PUB_KEY = `-----BEGIN PUBLIC KEY-----
MIGeMA0GCSqGSIb3DQEBAQUAA4GMADCBiAKBgHBhoIO6WsbQR6Dr+fyzwdUfrqz4
G1s4fKvcQR1NqfvGchXHTZZply7P+1NZnO4UX8z7T9VoMRSoS7lM8jdIeOjoyZuk
0WmNHZXGFeDNhoWtX/IZwy7z/e4qUD+rt1xVU3jjJqkQBSyar1FB+x9tG2qMGPhC
4cKjDWyJtRlopwbtAgMBAAE=
-----END PUBLIC KEY-----`


export function fakeSign(electionId: string, voterId: string): string {
  const key = new NodeRSA(FAKE_CSP_PRIV_KEY)

  const message = Buffer.from(electionId + voterId, "hex")

  return Buffer.from(key.sign(message)).toString("hex")
}
