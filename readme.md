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
### 2. Create ethereum wallet
Every validater must generate several wallets. One is for the smpc, and others for the evm MutliSignWallet. All the wallets must use one password to encrypt the keystore files. 
One of the keystore file:
```json
{
    "address": "d0928162bd6fe945125e3b3e15f77f6d7de45ff5",
    "crypto": {
        "cipher": "aes-128-ctr",
        "ciphertext": "49e8258ffcd4b9a613aa7730676480f2a49663531c6688a9a66984c12b5af9be",
        "cipherparams": {
            "iv": "dd48c82ea37283dc9f089f5cc45ad1e0"
        },
        "kdf": "scrypt",
        "kdfparams": {
            "dklen": 32,
            "n": 262144,
            "p": 1,
            "r": 8,
            "salt": "9e24a2ed36c4c8a22b7e5c893a1cbdb50a7929ff9c6d297285e2c7d05c0f0ab1"
        },
        "mac": "3e4c790f4b6314d2ffe4cd95c4362085f94f545637d15c86da72186d95147162"
    },
    "id": "2f5495c1-4886-4711-b217-3eb83f0fb72c",
    "version": 3
}
```

### 3. Start the smpc service.
* One of the validaters run the "bootnode" to create a bootnodes info and send it to other validaters.
* All the validaters start the mpc service by running the "gsmpc".
* One of the validaters request a group id and a sub group id and send them to other validaters.
* One of the validaters request some public keys by running "gsmpc-client" and send them to other valildaters.
* Other validaters should accept the request in time when one of them request gourp id or public key.
* The addresses of all the MutliSignWallets must send to TanglePay to create contract on evm.
* TanglePay send the evm contract address to all the validaters.
Detail infomation for this, can see [smpc-node keygen and sign workflow](https://github.com/TanglePay/smpc-node).

### 4. Config the bridge service
Before run this service, you must fill the config/conf.toml file with the right parameter. Detail is in the exmaple conf.toml file.

## Run the Bridge service
```shell
go build
./bwrap -d
```
Stop the service
```shell
./stop.sh
```

## Aes encrypt/decrypt
```go
func CreateKey(seed uint64, nSize uint64) []byte {
	if (nSize != 16) && (nSize != 32) && (nSize != 64) {
		return nil
	}
	data := make([]byte, nSize*4)
	for i := uint64(0); i < nSize*4; i++ {
		d := seed+i
		data[i] = uint8(d % 256)
	}
	var hs hash.Hash
	switch nSize {
	case 16:
		hs = md5.New()
	case 32:
		hs = sha256.New()
	case 64:
		hs = sha512.New()
	default:
		return nil
	}
	hs.Write(data)
	return hs.Sum(nil)
}
```
like the function above, you can rewrite the 
```go
	d := seed+i
```
to more complex formula, as follows
```go
	d := seed*math.Sin(i)
	d = seed * math.Cos(i) + i * seed
```
Think about that as much as possible. The more complex of the formula, the safer of the algorithm.

