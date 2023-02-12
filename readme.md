# Bridge From IOTA to Shimmer evm

## Prepare for this service
### 1. One of the validaters must install mysql service. The database name is "smpc". Create a table.
```sql
CREATE TABLE `swap_order` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `txid` varchar(512) NOT NULL,     /*txid is the messageID or hash in the iota network or shimmer evm network*/
  `src_token` varchar(45) NOT NULL,
  `dest_token` varchar(45) NOT NULL,
  `wrap` tinyint NOT NULL DEFAULT '1' COMMENT '1: wrap, -1: unwrap',
  `from` varchar(512) NOT NULL,
  `to` varchar(512) NOT NULL,
  `amount` varchar(45) NOT NULL,
  `hash` varchar(512) NOT NULL DEFAULT '',
  `state` tinyint NOT NULL DEFAULT '0',
  `ts` bigint NOT NULL,
  `order_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `txid` (`txid`),
  KEY `txid_select` (`txid`)
) ENGINE=InnoDB;
```
### 2. Start the smpc service.
* One of the validaters run the "bootnode" to create a bootnodes info and send it to other validaters.
* All the validaters start the mpc service by running the "gsmpc".
* One of the validaters request a group id and a sub group id and send them to other validaters.
* One of the validaters request some public keys by running "gsmpc-client" and send them to other valildaters.
* Other validaters should accept the request in time when one of them request gourp id or public key.
Detail infomation for this, can see [smpc-node keygen and sign workflow](https://github.com/TanglePay/smpc-node).

### 3. Config the bridge service
Before run this service, you must fill the config/conf.toml file with the right parameter. Detail is in the exmaple conf.toml file.

## Run the Bridge service
```shell
go build
./bwrap
```
Stop the service
```shell
./stop.sh
```