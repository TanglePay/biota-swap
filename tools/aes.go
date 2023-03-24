package tools

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"math"
)

func GenerateRandomSeed() uint64 {
	var n uint32
	if err := binary.Read(rand.Reader, binary.LittleEndian, &n); err != nil {
		panic(err)
	}
	seed := uint64(32) << uint64(n)
	if err := binary.Read(rand.Reader, binary.LittleEndian, &n); err != nil {
		panic(err)
	}
	seed += uint64(n)
	return seed
}

func CreateKey(seed uint64, nSize uint64) []byte {
	if (nSize != 16) && (nSize != 32) && (nSize != 64) {
		return nil
	}
	data := make([]byte, nSize*4)
	for i := uint64(0); i < nSize*4; i++ {
		d := int64(float64(123456 + float64(i)*math.Sin(float64(i))))
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

func MD5(str string) []byte {
	data := md5.Sum([]byte(str))
	return data[0:16]
}

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

func GetEncryptString(source string, seeds [4]uint64) []byte {
	key, iv := getKeyAndIv(seeds)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}

	blockSize := block.BlockSize()
	rawData := []byte(source)
	padding := blockSize - len(rawData)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	rawData = append(rawData, padtext...)

	cipherText := make([]byte, len(rawData))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText, rawData)

	return cipherText
}

func GetDecryptString(encrypt string, seeds [4]uint64) []byte {
	key, iv := getKeyAndIv(seeds)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}
	blockSize := block.BlockSize()

	enData, err := hex.DecodeString(encrypt)
	if err != nil {
		return nil
	}

	if (len(enData) < blockSize) || (len(enData)%blockSize != 0) {
		return nil
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(enData, enData)

	length := len(enData)
	unpadding := int(enData[length-1])
	if length > unpadding {
		enData = enData[:(length - unpadding)]
	} else {
		enData = nil
	}
	return enData
}

func getKeyAndIv(seeds [4]uint64) ([]byte, []byte) {
	key := CreateKey(seeds[0], 16)
	key = append(key, CreateKey(seeds[1], 16)[:8]...)
	key = append(key, CreateKey(seeds[2], 16)[8:]...)
	iv := CreateKey(seeds[3], 16)
	return key, iv
}
