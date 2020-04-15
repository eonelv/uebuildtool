package core

import "encoding/binary"

type Encrypt struct {
	Pos1   int
	Pos2   int
	enCode *EncryptCode
}

type EncryptCode struct {
	BufferEncrypt1 []byte
	BufferEncrypt2 []byte
}

func (this *EncryptCode) initEncrypt(a1, b1, c1, fst1, a2, b2, c2, fst2 int) {
	this.BufferEncrypt1 = make([]byte, 256)
	this.BufferEncrypt2 = make([]byte, 256)
	code := byte(fst1)
	for i := 0; i < 256; i++ {
		this.BufferEncrypt1[i] = code
		code = byte((a1*int(code)*int(code) + b1*int(code) + c1) % 256)
	}

	code = byte(fst2)
	for i := 0; i < 256; i++ {
		this.BufferEncrypt2[i] = code
		code = byte((a2*int(code)*int(code) + b2*int(code) + c2) % 256)
	}
}

func (this *Encrypt) InitEncrypt(a1, b1, c1, fst1, a2, b2, c2, fst2 int) {
	enCode := &EncryptCode{}
	enCode.initEncrypt(a1, b1, c1, fst1, a2, b2, c2, fst2)
	this.enCode = enCode
}

func (this *Encrypt) Encrypt(buff []byte, begin int, length int, move bool) {
	/*
		defer func() {
			if x := recover(); x != nil {
				LogError("Encrypt error:", x)
			}
		}()

		oldPos1 := this.Pos1
		oldPos2 := this.Pos2
		for i := begin; i < begin+length; i++ {
			buff[i] ^= this.enCode.BufferEncrypt1[this.Pos1]
			buff[i] ^= this.enCode.BufferEncrypt2[this.Pos2]
			this.Pos1++
			if this.Pos1 >= 256 {
				this.Pos1 = 0
				this.Pos2++
				if this.Pos2 >= 256 {
					this.Pos2 = 0
				}
			}
		}

		if !move {
			this.Pos1 = oldPos1
			this.Pos2 = oldPos2
		}
	*/
}

func (this *Encrypt) Reset() {
	this.Pos1 = 0
	this.Pos2 = 0
}

func (this *Encrypt) ChangeCode(data int) {
	data2 := uint32(data * data)

	for i := 0; i < 256; i += 4 {
		temp := binary.BigEndian.Uint32(this.enCode.BufferEncrypt1[i : i+4])
		temp ^= data2

		binary.BigEndian.PutUint32(this.enCode.BufferEncrypt1[i:i+4], temp)

		temp = binary.BigEndian.Uint32(this.enCode.BufferEncrypt2[i : i+4])
		temp ^= data2

		binary.BigEndian.PutUint32(this.enCode.BufferEncrypt2[i:i+4], temp)
	}
}
