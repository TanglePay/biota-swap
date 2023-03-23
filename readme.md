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
1. To hide the real password.
2. Validator can modify the main.go and config/config.go to put the encrypted password in the conf.toml file.
3. 
```go
func AesCBCEncrypt(source string, key []byte, aesCbcIv []byte) string {
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}

	blockSize := block.BlockSize()
	rawData := []byte(source)
	padding := blockSize - len(rawData)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	rawData = append(rawData, padtext...)

	iv := aesCbcIv[:blockSize]

	cipherText := make([]byte, len(rawData))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText, rawData)

	return hex.EncodeToString(cipherText)
}

func AesCBCDecrypt(encrypt string, key []byte, aesCbcIv []byte) string {
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	blockSize := block.BlockSize()

	enData, err := hex.DecodeString(encrypt)
	if err != nil {
		return ""
	}

	if (len(enData) < blockSize) || (len(enData)%blockSize != 0) {
		return ""
	}

	iv := aesCbcIv[:blockSize]

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(enData, enData)

    length := len(enData)
	unpadding := int(enData[length-1])
	if length > unpadding {
		enData = enData[:(length - unpadding)]
	} else {
		enData = nil
	}
	return string(enData)
}

//modify to yourself
func CreateKey(seed uint64, nSize uint64) []byte {
	if (nSize != 16) && (nSize != 32) && (nSize != 64) {
		return nil
	}
	data := make([]byte, nSize*4)
	for i := uint64(0); i < nSize*4; i++ {
		d := int64(123456789.0+float64(2*i)++float64(3*i*i))
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

1. Modify the CreateKey() to yourself, and put the  CreateKey() and AesCBCDecrypt() into main.go.
2. Use the CreateKey() to generate a key and CbcIv . Example as : 
   ```go
   key := CreateKey(0x6345a5326e5ff3df, 32);
   iv := CreateKey(0x1234a5678e5afded, 32);
   ```
3. Use the key and iv to encrypt the password to a string (by using AesCBCDecrypt() in the main.go).
4. Input the encrypt string in the func input() of main.go.
5. Decrypt the pwd := readRand() in the func main() using AesCBCDecrypt.


