package utils

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
)

func GetSHA1(data []byte) string {
	return fmt.Sprintf("%x", sha1.Sum(data))

}

func GetMD5(data []byte) string {
	return fmt.Sprintf("%x", md5.Sum(data))
}
