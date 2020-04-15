package core

import (
	"strings"
)

const BOX_LENGTH int = 512

const digits = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-=!@#$"

type RC4Encrypt struct {
	key    []int
	srcKey string
}

func (this *RC4Encrypt) Init(pass string) {
	this.key = this.getKey(pass)
	this.srcKey = pass
}
func (this *RC4Encrypt) DoEncrypt(text string) string {
	var resultByte []byte
	resultByte = this.RC4(text)
	return bytesToText(resultByte, 64)
}

func (this *RC4Encrypt) DoDecrypt(text string) string {
	var resultByte []byte
	resultByte = textToBytes(text, 64)
	resultByte = this.RC4(string(resultByte))
	return string(resultByte)
}


func ByteToHex(datas []byte) string {
	length := len(datas)
	var resultDatas []byte = make([]byte, 2*length)
	var temp2 int
	for i := 0; i < length; i++ {
		temp2 = int(datas[i]) / 16
		if temp2 > 9 {
			temp2 += (int('A') - 10)
		} else {
			temp2 += int('0')
		}
		resultDatas[2*i] = byte(temp2)

		temp2 = (int(datas[i])) % 16
		if temp2 > 9 {
			temp2 += (int('A') - 10)
		} else {
			temp2 += int('0')
		}
		resultDatas[i*2+1] = byte(temp2)
	}
	return string(resultDatas)
}

func HexToByte(text string) []byte {
	var textBytes []byte = []byte(text)
	var length int = len(textBytes) / 2
	var result []byte = make([]byte, length)

	var temp1 int
	var temp2 int
	for i := 0; i < length; i++ {
		temp1 = int(textBytes[2*i])
		if temp1 >= int('A') {
			temp1 -= (int('A') - 10)
		} else {
			temp1 -= int('0')
		}

		temp2 = int(textBytes[2*i+1])
		if temp2 >= int('A') {
			temp2 -= (int('A') - 10)
		} else {
			temp2 -= int('0')
		}
		result[i] = byte(16*temp1 + temp2)
	}
	return result
}

func (this *RC4Encrypt) RC4(textData string) []byte {
	//当每次加密或解密需要充值key的时候才调用该函数，同时取消注释下面for循环中的交换函数
	//	key := this.getKey(this.srcKey)
	key := this.key
	var data []byte = []byte(textData)
	length := len(data)
	var x int = 0
	var y int = 0
	for i := 0; i < length; i++ {
		x = (x + 1) % BOX_LENGTH
		y = (key[x] + y) % BOX_LENGTH
		//		temp := key[x]
		//		key[x] = key[y]
		//		key[y] = temp
		data[i] = data[i] ^ byte(key[(key[x]+key[y])%BOX_LENGTH])
	}

	return data
}

func (this *RC4Encrypt) getKey(pass string) []int {
	var passBytes []byte = []byte(pass)
	length := len(pass)
	var i int
	var key []int = make([]int, BOX_LENGTH)
	for i = 0; i < BOX_LENGTH; i++ {
		key[i] = i
	}
	var j int = 0
	for i = 0; i < BOX_LENGTH; i++ {
		j = (int(passBytes[i%length]) + key[i] + j) % BOX_LENGTH
		temp := key[i]
		key[i] = key[j]
		key[j] = temp
	}
	return key
}

func textToBytes(text string, radix uint64) []byte {
	var array []string
	array = strings.Split(text, "-")
	var length int = len(array)
	var datas []byte = make([]byte, (int(radix/8))*length)

	var number uint64
	var pos1 byte
	var pos2 byte
	var pos3 byte
	var pos4 byte
	var pos5 byte
	var pos6 byte
	var pos7 byte
	var pos8 byte
	for i := 0; i < length; i++ {
		number = textToUint64(array[i], radix)
		pos1 = byte(number & 0xFFFFFFFFFFFFFFFF >> 56)
		pos2 = byte(number & 0xFFFFFFFFFFFFFF >> 48)
		pos3 = byte(number & 0xFFFFFFFFFFFF >> 40)
		pos4 = byte(number & 0xFFFFFFFFFF >> 32)
		pos5 = byte(number & 0xFFFFFFFF >> 24)
		pos6 = byte(number & 0xFFFFFF >> 16)
		pos7 = byte(number & 0xFFFF >> 8)
		pos8 = byte(number & 0xFF)
		datas[i*8] = pos8
		datas[i*8+1] = pos7
		datas[i*8+2] = pos6
		datas[i*8+3] = pos5
		datas[i*8+4] = pos4
		datas[i*8+5] = pos3
		datas[i*8+6] = pos2
		datas[i*8+7] = pos1
	}
	var j int = len(datas) - 1
	for ; j > 0; j-- {
		if datas[j] != 0 {
			break
		}
	}
	return datas[:j+1]
	return datas
}

func textToUint64(v string, radix uint64) uint64 {
	var bytes []byte = []byte(v)
	var result uint64
	for i := len(bytes); i > 0; i-- {
		var curr int
		curr = strings.Index(digits, string(bytes[len(bytes)-i]))

		result += digit(uint64(curr), i-1, radix)
	}
	return result
}

func digit(v uint64, n int, radix uint64) uint64 {
	var result uint64 = v
	for i := 0; i < n; i++ {
		result = result * radix
	}
	return result
}

func bytesToText(datas []byte, radix uint64) string {
	var length int = len(datas)

	var size int = length / 4
	if length%4 != 0 {
		size += 1
	}

	var pos1 uint64
	var pos2 uint64
	var pos3 uint64
	var pos4 uint64
	var pos5 uint64
	var pos6 uint64
	var pos7 uint64
	var pos8 uint64
	var value uint64
	var totalByte []byte = []byte{}
	var i int = 0
	for ; i < length; i += 8 {

		if (i + 7) >= length {
			pos1 = 0
		} else {
			pos1 = uint64(datas[i+7]) << 56
		}
		if (i + 6) >= length {
			pos2 = 0
		} else {
			pos2 = uint64(datas[i+6]) << 48
		}
		if (i + 5) >= length {
			pos3 = 0
		} else {
			pos3 = uint64(datas[i+5]) << 40
		}
		if (i + 4) >= length {
			pos4 = 0
		} else {
			pos4 = uint64(datas[i+4]) << 32
		}
		if (i + 3) >= length {
			pos5 = 0
		} else {
			pos5 = uint64(datas[i+3]) << 24
		}
		if (i + 2) >= length {
			pos6 = 0
		} else {
			pos6 = uint64(datas[i+2]) << 16
		}
		if (i + 1) >= length {
			pos7 = 0
		} else {
			pos7 = uint64(datas[i+1]) << 8
		}
		pos8 = uint64(datas[i])
		value = pos1 | pos2 | pos3 | pos4 | pos5 | pos6 | pos7 | pos8

		totalByte = append(totalByte, uint64ToByte(value, radix)...)
		totalByte = append(totalByte, []byte("-")...)

	}
	totalByte = totalByte[0 : len(totalByte)-1]
	return string(totalByte)
}

func uint64ToByte(i uint64, radix uint64) []byte {
	var bytes []byte = make([]byte, 33)
	var charPos int = 32
	for i >= radix {
		bytes[charPos] = digits[(i % radix)]
		charPos--
		i = i / radix
	}
	bytes[charPos] = digits[i]
	return bytes[charPos:]
}
