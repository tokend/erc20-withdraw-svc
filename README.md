# ERC20 withdraw service
ERC20 withdraw service is a bridge between TokenD and Ethereum blockchain which allows 
to withdraw tokens from TokenD to Ethereum blockchain.

## Usage

Enviromental variable `KV_VIPER_FILE` must be set and contain path to desired config file.

```bash
erc20-withdraw-svc run withdraw
```

## Watchlist

In order for service to start watching withdrawals in specific asset, asset details in TokenD must have entry of the following form: 
```json5
{ 
//...
  "erc20": {
   "withdraw": true, 
   "address": "0x0000000000000000000",  //contract address
   },
//...
}
```
Service will only listen for withdraw requests with `2048` pending tasks flag set and `4096` flag not set.
So, either value by key `withdrawal_tasks:*`, or `withdrawal_tasks:ASSET_CODE`  must contain `2048` flag and must not contain flag `4096`.

## Config

```yaml
horizon:
  endpoint: "SOME_VALID_ADDRESS"
  signer: "G_ASSET_OWNER_SECRET_KEY" # Issuer of assets

withdraw:
  signer: "S_ASSET_OWNER_SECRET_KEY"
  owner: "G_ASSET_OWNER_ADDRESS"
  
rpc:
  endpoint: "ws://ETH_NODE_ADDRESS"

transfer:
  seed: "SECRET_SEED"
  address: "SOURCE_ADDRESS"


log:
  level: debug
  disable_sentry: true
```

## Ethereum node

Node must be configured to accept connections through websockets. 
Origin must be explicitly or implicitly whitelisted:
either `--wsorigins "some_origin"`, or `--wsorigins *` to accept all connections.